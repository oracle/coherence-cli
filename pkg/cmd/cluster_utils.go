/*
 * Copyright (c) 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const consoleClass = "com.tangosol.net.CacheFactory"
const cohQLClass = "com.tangosol.coherence.dslquery.QueryPlus"
const ceGroupID = "com.oracle.cohrence.ce"
const fileTypeJar = "jar"
const fileTypePom = "pom"

// default Jars to use
var (
	defaultJars = []*config.DefaultDependency{
		{GroupID: ceGroupID, Artifact: "coherence", IsCoherence: true},
		{GroupID: ceGroupID, Artifact: "coherence-json", IsCoherence: true},
		{GroupID: "org.jline", Artifact: "jline", IsCoherence: false, Version: "3.20.0"},
	}

	// list of additional coherence artifacts
	validCoherenceArtifacts = []string{"coherence-cdi-server", "coherence-cdi", "coherence-concurrent", "coherence-grpc-proxy",
		"coherence-grpc", "coherence-helidon-client", "coherence-helidon-grpc-proxy", "coherence-http-netty", "coherence-java-client",
		"coherence-jcache", "coherence-jpa", "coherence-management", "coherence-micrometer", "coherence-mp-config", "coherence-metrics",
		"coherence-mp-metrics", "coherence-rest"}
)

// checkCreateRequirements validates that all the necessary requirements are fulfilled
// for creating a cluster. This includes mvn and java executables. Nil is returned to
// indicate everything is ok, otherwise an error is returned
func checkCreateRequirements() error {
	var (
		javaExec = getJavaExec()
		mvnExec  = getMvnExec()
		err      error
	)

	processJava := exec.Command(javaExec, "-v")
	if err = processJava.Start(); err != nil {
		return utils.GetError(fmt.Sprintf("unable to get Java version using %s -v: %v", javaExec, processJava), err)
	}

	processMaven := exec.Command(mvnExec, "-v")
	if err = processMaven.Start(); err != nil {
		return utils.GetError(fmt.Sprintf("unable to get Maven version using %s -v, %v", mvnExec, processMaven), err)
	}

	return nil
}

func getJavaExec() string {
	return "java"
}

func getMvnExec() string {
	return "mvn"
}

// getCoherenceDependencies runs the mvn dependency:get command to download coherence.jar and coherence-json.jar
// which are the minimum requirements to create a cluster with management over rest enabled
func getCoherenceDependencies(cmd *cobra.Command) error {
	var (
		mvnExec = getMvnExec()
		err     error
		result  string
	)

	// sort the defaultJars
	sort.Slice(defaultJars, func(p, q int) bool {
		if defaultJars[p].GroupID == defaultJars[q].GroupID {
			return strings.Compare(defaultJars[p].Artifact, defaultJars[q].Artifact) < 0
		}
		return strings.Compare(defaultJars[p].GroupID, defaultJars[q].GroupID) < 0
	})

	for _, entry := range defaultJars {
		cmd.Printf("- %s:%s:%s\n", entry.GroupID, entry.Artifact, entry.Version)
		result, err = runCommand(mvnExec, getDependencyArgs(entry.GroupID, entry.Artifact, entry.Version))
		if err != nil {
			cmd.Println(result)
			return err
		}
	}

	return nil
}

func validateLogLevel(logLevel int32) error {
	if logLevel < 0 || logLevel > 9 {
		return fmt.Errorf("log level must be between 0 and 9")
	}
	return nil
}

func updateDefaultJars() {
	groupID := getCoherenceGroupID()

	for _, entry := range defaultJars {
		if entry.IsCoherence {
			entry.GroupID = groupID
			entry.Version = clusterVersionParam
		}
	}
}

// startCluster starts a cluster. If existingCount > 1 then this means we are
// scaling a cluster, otherwise we are starting one
func startCluster(cmd *cobra.Command, connection ClusterConnection, serverCount, existingCount int32) error {
	var (
		err      error
		mgmtPort = connection.ManagementPort
		counter  int32
	)

	// if we are scaling then set the http port to -1 so no more management servers are started
	if existingCount > 0 {
		mgmtPort = -1
	}

	if err = checkOperation(connection, startClusterCommand); err != nil {
		return err
	}

	for counter = existingCount; counter < serverCount+existingCount; counter++ {
		var (
			member        = fmt.Sprintf("storage-%d", counter)
			arguments     = getCommonArguments(connection)
			memberLogFile string
		)

		arguments = append(arguments, getCacheServerArgs(member, mgmtPort)...)

		// reset so only first member has management enabled
		mgmtPort = -1

		memberLogFile, err = getLogFile(connection.Name, member)
		if err != nil {
			return err
		}

		cmd.Printf("Starting cluster member %s...\n", member)
		_, err = runCommandAsync(getJavaExec(), memberLogFile, arguments)
		if err != nil {
			return utils.GetError(fmt.Sprintf("unable to start member %s", member), err)
		}

		time.Sleep(time.Duration(startupDelayParam) * time.Second)
	}

	return nil
}

// getCommonArguments returns arguments that are common to clients and servers
func getCommonArguments(connection ClusterConnection) []string {
	splitArguments := strings.Split(connection.Arguments, " ")
	return append(splitArguments, "-cp", connection.BaseClasspath, getPersistenceProperty(connection.PersistenceMode),
		getLogLevelProperty(logLevelParam))
}

func startClient(cmd *cobra.Command, connection ClusterConnection, class string) error {
	var (
		err       error
		result    string
		arguments = getCommonArguments(connection)
	)

	arguments = append(arguments, getClientArgs(class, class)...)

	cmd.Printf("Starting client %s...\n", class)
	process := exec.Command(getJavaExec(), arguments...) // #nosec G204
	process.Stdout = cmd.OutOrStdout()
	process.Stdin = cmd.InOrStdin()
	process.Stderr = cmd.ErrOrStderr()
	err = process.Start()
	if err != nil {
		return utils.GetError(fmt.Sprintf("unable to start %s: %v", class, result), err)
	}

	// handle CTRL-C
	//handleCTRLC()

	return process.Wait()
}

func getCacheServerArgs(member string, httpPort int32) []string {
	baseArgs := make([]string, 0)
	if httpPort != -1 {
		baseArgs = append(baseArgs, "-Dcoherence.management.http=all", fmt.Sprintf("-Dcoherence.management.http.port=%d", httpPort),
			"-Dcoherence.management=all")
	}
	baseArgs = append(baseArgs, "-Xms"+heapMemoryParam, "-Xmx"+heapMemoryParam)

	return append(baseArgs, getMemberProperty(member), "-Dcoherence.log.level=6", "com.tangosol.net.Coherence")
}

// getClientArgs returns the arguments for starting a Coherence process such as
// console or cohQL
func getClientArgs(member, class string) []string {
	baseArgs := make([]string, 0)
	baseArgs = append(baseArgs, "-Xms"+heapMemoryParam, "-Xmx"+heapMemoryParam)

	if class == cohQLClass && extendClientParam {
		// only works with default Cache config
		baseArgs = append(baseArgs, "-Dcoherence.client=remote")
	}

	return append(baseArgs, getMemberProperty("client"), "-Dcoherence.log.level=5",
		"-Dcoherence.distributed.localstorage=false", class)
}

func getMemberProperty(member string) string {
	return fmt.Sprintf("-Dcoherence.member=%s", member)
}

func getPersistenceProperty(persistenceMode string) string {
	return fmt.Sprintf("-Dcoherence.distributed.persistence.mode=%s", persistenceMode)
}

func getLogLevelProperty(logLevel int32) string {
	return fmt.Sprintf("-Dcoherence.log.level=%d", logLevel)
}

// getRunningProcesses returns the running process ID's for a cluster
// connection from a dataFetcher. Returns an empty slice if none are running
func getRunningProcesses(dataFetcher fetcher.Fetcher) []int {
	var (
		PIDS          = make([]int, 0)
		err           error
		membersResult []byte
		members       = config.Members{}
	)

	membersResult, err = dataFetcher.GetMemberDetailsJSON(false)
	if err != nil {
		return PIDS
	}

	// unmarshall and assume any errors means no PIDS are running
	err = json.Unmarshal(membersResult, &members)
	if err != nil {
		return PIDS
	}

	for _, v := range members.Members {
		pid, _ := strconv.Atoi(v.ProcessName)
		PIDS = append(PIDS, pid)
	}

	return PIDS
}

func checkOperation(connection ClusterConnection, operation string) error {
	if connection.ManuallyCreated {
		return nil
	}
	return fmt.Errorf("cluster %s was not manually created, unable to perform operation %s", connection.Name, operation)
}

// getTransitiveClasspath returns the transitive classpath by using mvn dependency:build-classpath,
// outputting to temp file and reading in
func getTransitiveClasspath(groupID, artifact, version string) (string, error) {
	var (
		err       error
		pomFile   string
		file      *os.File
		classpath string
		output    string
		data      []byte
		mvnExec   = getMvnExec()
		arguments = []string{"dependency:build-classpath", "-DincludeScope=runtime", "-f"}
	)

	pomFile, err = getMavenClasspath(groupID, artifact, version, fileTypePom)
	if err != nil {
		return classpath, err
	}

	file, err = os.CreateTemp("", "classpath")
	if err != nil {
		return classpath, utils.GetError("unable to create temporary file", err)
	}

	defer os.Remove(file.Name())

	// execute the build-classpath command
	output, err = runCommand(mvnExec, append(arguments, pomFile, fmt.Sprintf("-Dmdep.outputFile=%s", file.Name())))

	if err != nil {
		return output, utils.GetError("unable to run dependency:build-classpath", err)
	}

	// no we have a valid file, read it in
	data, err = ioutil.ReadFile(file.Name())
	if err != nil {
		return output, utils.GetError("unable to read from temp file", err)
	}

	return string(data), nil
}

func getDependencyArgs(groupID, artifact, version string) []string {
	gavArgs := getGAVArgs(groupID, artifact, version)
	return append(gavArgs, "dependency:get", "-Dtransitive=true")
}

func getGAVArgs(groupID, artifact, version string) []string {
	return []string{"-DgroupId=" + groupID, "-DartifactId=" + artifact, "-Dversion=" + version}
}

func getCoherenceGroupID() string {
	if useCommercialParam {
		return "com.oracle.coherence"
	}
	return "com.oracle.coherence.ce"
}

// getMavenClasspath returns the maven classpath for the given GAV and fileType
func getMavenClasspath(groupID, artifact, version, fileType string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", utils.GetError("unable to get user home directory", err)
	}

	// split the groupID
	groupIDSplit := strings.Split(groupID, ".")
	baseDir := filepath.Join(home, ".m2", "repository")
	for _, v := range groupIDSplit {
		baseDir = filepath.Join(baseDir, v)
	}
	return filepath.Join(baseDir, artifact, version, fmt.Sprintf("%s-%s.%s", artifact, version, fileType)), nil
}

func runCommand(command string, arguments []string) (string, error) {
	return runCommandBase(command, "", arguments)
}

func runCommandAsync(command, logFileName string, arguments []string) (string, error) {
	return runCommandBase(command, logFileName, arguments)
}

// runCommandBase runs a command. If logFileName is supplied then this is done async and the
// processId is returned, otherwise the result of the combined stdout and stderr is returned
func runCommandBase(command, logFileName string, arguments []string) (string, error) {
	var (
		err            error
		result         []byte
		processLogFile *os.File
	)

	process := exec.Command(command, arguments...)
	if len(logFileName) > 0 {
		// a log file was supplied, so we are assuming this command will be async and
		// stdout and stderr should be redirected to log file specified
		processLogFile, err = os.Create(logFileName)
		if err != nil {
			return "", utils.GetError("unable to create log file"+logFileName, err)
		}
		process.Stdout = processLogFile
		process.Stderr = processLogFile
		// detach the process from the cohctl executable
		setForkProcess(process)
		if err = process.Start(); err != nil {
			return "", utils.GetError(fmt.Sprintf("unable to start process %v", process), err)
		}
		return fmt.Sprintf("%d", process.Process.Pid), nil
	}
	// wait for result
	result, err = process.CombinedOutput()

	if err != nil {
		return "", utils.GetError(fmt.Sprintf("unable to start process %s, %v\n%s", command, process, string(result)), err)
	}

	return string(result), nil
}

func getLogFile(clusterName, processName string) (string, error) {
	clusterLogsDir := filepath.Join(logsDirectory, clusterName)
	if err := utils.EnsureDirectory(clusterLogsDir); err != nil {
		return "", err
	}

	clusterLogFile := filepath.Join(clusterLogsDir, processName+".log")
	return clusterLogFile, nil
}

func getClasspathSeparator() string {
	if isWindows() {
		return ";"
	}
	return ":"
}

// getConnection returns a ClusterConnection
func getConnection(connectionName string) (bool, ClusterConnection) {
	for _, cluster := range Config.Clusters {
		if cluster.Name == connectionName {
			return true, cluster
		}
	}
	return false, ClusterConnection{}
}
