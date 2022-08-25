/*
 * Copyright (c) 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"fmt"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const consoleClass = "com.tangosol.net.CacheFactory"
const cohQLClass = "com.tangosol.coherence.dslquery.QueryPlus"
const ceGroupID = "com.oracle.cohrence.ce"

// default Jars to use
var (
	defaultJars = []*config.DefaultDependency{
		{GroupID: ceGroupID, Artefact: "coherence", IsCoherence: true},
		{GroupID: ceGroupID, Artefact: "coherence-json", IsCoherence: true},
		{GroupID: "org.jline", Artefact: "jline", IsCoherence: false, Version: "3.20.0"},
	}

	// list of additional coherence artefacts
	validCoherenceArtefacts = []string{"coherence-cdi-server", "coherence-cdi", "coherence-concurrent", "coherence-grpc-proxy",
		"coherence-grpc", "coherence-helidon-client", "coherence-helidon-grpc-proxy", "coherence-http-netty", "coherence-java-client",
		"coherence-jcache", "coherence-jpa", "coherence-management", "coherence-micrometer", "coherence-mp-config",
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

	cmd.Println("Ensuring dependencies")
	for _, entry := range defaultJars {
		cmd.Printf("- groupId=%s, artefact=%s, version=%s\n", entry.GroupID, entry.Artefact, entry.Version)
		result, err = runCommand(mvnExec, getDependencyArgs(entry.GroupID, entry.Artefact, entry.Version))
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

// startCluster starts a cluster
func startCluster(cmd *cobra.Command, connection ClusterConnection, serverCount int32) ([]string, error) {
	var (
		err        error
		processIDs = make([]string, 0)
		mgmtPort   = connection.ManagementPort
		counter    int32
	)

	if err = checkOperation(connection, startClusterCommand); err != nil {
		return processIDs, err
	}

	for counter = 0; counter < serverCount; counter++ {
		var (
			member        = fmt.Sprintf("storage-%d", counter)
			arguments     = getCommonArguments(connection)
			memberLogFile string
			PID           string
		)

		arguments = append(arguments, getCacheServerArgs(member, mgmtPort)...)

		// reset so only first member has management enabled
		mgmtPort = -1

		memberLogFile, err = getLogFile(connection.Name, member)
		if err != nil {
			return processIDs, err
		}

		cmd.Printf("Starting cluster member %s...\n", member)
		PID, err = runCommandAsync(getJavaExec(), memberLogFile, arguments)
		if err != nil {
			return processIDs, utils.GetError(fmt.Sprintf("unable to start member %s", member), err)
		}
		processIDs = append(processIDs, PID)

		time.Sleep(time.Duration(startupDelayParam) * time.Second)
	}

	return processIDs, nil
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

// convertProcessIDs converts an array of string processes to int array
func convertProcessIDs(processIDs []string) []int {
	PIDS := make([]int, 0)
	for _, v := range processIDs {
		pid, _ := strconv.Atoi(v)
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

func getDependencyArgs(groupID, artefact, version string) []string {
	return []string{"-DgroupId=" + groupID, "-DartifactId=" + artefact, "-Dversion=" + version, "dependency:get"}
}

func getCoherenceGroupID() string {
	if useCommercialParam {
		return "com.oracle.coherence"
	}
	return "com.oracle.coherence.ce"
}

func getMavenClasspath(groupID, artefact, version string) (string, error) {
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
	return filepath.Join(baseDir, artefact, version, fmt.Sprintf("%s-%s.jar", artefact, version)), nil
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

// updateConnectionPIDS updates PIDS for a given connection
func updateConnectionPIDS(connectionName string, PIDs []int) error {
	clusters := Config.Clusters

	for i, cluster := range clusters {
		if cluster.Name == connectionName {
			Config.Clusters[i].ProcessIDs = PIDs
		}
	}

	viper.Set("clusters", Config.Clusters)
	err := WriteConfig()

	return err
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

func getProcesses(PIDS []int, members config.Members) config.Processes {
	var (
		proc        *os.Process
		processList = make([]config.Process, 0)
		err         error
		running     bool
	)

	for _, v := range PIDS {
		running = false
		member := config.Member{}

		proc, err = os.FindProcess(v)
		if err == nil {
			// signal the process as FindProcess always returns true on POSIX
			if err = signalProcess(proc); err == nil {
				running = true

				// as the member is running, try to find the member with the same process name
				var procID = fmt.Sprintf("%v", v)

				for _, m := range members.Members {
					if procID == m.ProcessName {
						member = m
						break
					}
				}
			}
		}

		processList = append(processList, config.Process{
			ProcessID:  v,
			Running:    running,
			NodeID:     member.NodeID,
			RoleName:   member.RoleName,
			MemberName: member.MemberName,
		})
	}

	return config.Processes{ProcessList: processList}
}
