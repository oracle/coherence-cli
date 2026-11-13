/*
 * Copyright (c) 2022, 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	consoleClass  = "com.tangosol.net.CacheFactory"
	cohQLClass    = "com.tangosol.coherence.dslquery.QueryPlus"
	ceGroupID     = "com.oracle.cohrence.ce"
	fileTypeJar   = "jar"
	fileTypePom   = "pom"
	javaExec      = "java"
	mvnExec       = "mvn"
	gradleExec    = "gradle"
	gradleDirName = "gradle-dir-name-cohctl-cli"
)

// a build template for saving the runtime classpath to a file by running
//
// gradle --no-daemon -b build.gradle -q buildClasspath -PfileName=/tmp/file.out
const buildGradleFilePart1 = `
plugins {
    id 'java'
}

repositories {
    mavenCentral()
}

dependencies {
`

const buildGradleFilePart2 = `
}

tasks.register("buildClasspath") {
    dependsOn build
    group = "Execution"
    def fileName = findProperty('fileName') ?: "file.out"
    new File(fileName).text = sourceSets.test.runtimeClasspath.getAsPath()
}
`

// default Jars to use
var (
	defaultJars = []*config.DefaultDependency{
		{GroupID: ceGroupID, Artifact: "coherence", IsCoherence: true},
		{GroupID: ceGroupID, Artifact: "coherence-json", IsCoherence: true},
		{GroupID: "org.jline", Artifact: "jline", IsCoherence: false, Version: "3.26.3"},
	}

	// list of additional coherence artifacts
	validCoherenceArtifacts = []string{"coherence-cdi-server", "coherence-cdi", "coherence-concurrent", "coherence-grpc-proxy",
		"coherence-grpc", "coherence-helidon-client", "coherence-helidon-grpc-proxy", "coherence-http-netty", "coherence-java-client",
		"coherence-jcache", "coherence-jpa", "coherence-management", "coherence-micrometer", "coherence-mp-config", "coherence-metrics",
		"coherence-mp-metrics", "coherence-rest", "coherence-hnsw", "coherence-mp-health"}
)

// checkDepsRequirements checks for either mvn or gradle depending upon the
// setting of Config.useGradle.
func checkDepsRequirements() error {
	var (
		err      error
		execName = getExecType()
	)

	proc := exec.Command(execName, "-v")
	if err = proc.Run(); err != nil {
		return utils.GetError(fmt.Sprintf("unable to get depdencies tool using %s -v, %v", execName, proc), err)
	}

	return nil
}

func getExecType() string {
	if Config.UseGradle {
		return gradleExec
	}

	return mvnExec
}

// buildGradleClasspath builds a classpath using gradle by creating a temporary
// build.gradle file and running a custom task.
// this is experimental and if we can find a better way to do this then we can change this.
func buildGradleClasspath() ([]string, error) {
	var (
		err           error
		classpath     = make([]string, 0)
		gradleTempDir string
		gradleFile    string
		outputFile    *os.File
		data          []byte
		output        string
		arguments     = []string{"--no-daemon", "-q", "buildClasspath", "-b"}
		sb            strings.Builder
	)

	// create a temporary directory for gradle file
	gradleTempDir, err = os.MkdirTemp("", gradleDirName)
	if err != nil {
		return classpath, utils.GetError("unable to create temporary directory", err)
	}

	gradleFile = path.Join(gradleTempDir, "build.gradle")

	outputFile, err = os.CreateTemp("", "classpath")
	if err != nil {
		return classpath, utils.GetError("unable to create temporary file", err)
	}

	defer os.Remove(gradleFile)
	defer os.Remove(gradleTempDir)
	defer os.Remove(outputFile.Name())

	// build the gradle dependencies
	for _, v := range defaultJars {
		sb.WriteString(fmt.Sprintf("implementation '%s:%s:%s'\n", v.GroupID, v.Artifact, v.Version))
	}

	finalGradleFile := buildGradleFilePart1 + sb.String() + buildGradleFilePart2

	// write the gradle file
	err = os.WriteFile(gradleFile, []byte(finalGradleFile), 0600)
	if err != nil {
		return classpath, utils.GetError("unable to write to temporary file", err)
	}

	// now we have the build.gradle file, run it to get the classpath in the outputFle
	output, err = runCommand(gradleExec, append(arguments, gradleFile, fmt.Sprintf("-PfileName=%s", outputFile.Name())))

	if err != nil {
		return classpath, utils.GetError(fmt.Sprintf("unable to run gradle command.\n%s", output), err)
	}

	// now we have a valid file, read it in
	data, err = os.ReadFile(outputFile.Name())
	if err != nil {
		return classpath, utils.GetError("unable to read from temp file", err)
	}

	// go through the generated classpath and remove any entry that contains
	// gradleDirName as these are added by gradle in the temporary directory created
	for _, v := range strings.Split(string(data), getClasspathSeparator()) {
		if !strings.Contains(v, gradleTempDir) {
			classpath = append(classpath, v)
		}
	}

	// convert the full path to a slice
	return classpath, nil
}

// getCoherenceMavenDependencies runs the mvn dependency:get command to download coherence.jar and coherence-json.jar
// which are the minimum requirements to create a cluster with management over rest enabled.
func getCoherenceMavenDependencies(cmd *cobra.Command) error {
	var (
		err    error
		result string
	)

	for _, entry := range defaultJars {
		cmd.Printf("- %s:%s:%s\n", entry.GroupID, entry.Artifact, entry.Version)
		args := getDependencyArgs(entry.GroupID, entry.Artifact, entry.Version)
		if settingsFileParam != "" {
			args = append(args, "-s", settingsFileParam)
		}
		result, err = runCommand(mvnExec, args)
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

// getCommonArguments returns arguments that are common to clients and servers.
func getCommonArguments(connection ClusterConnection) ([]string, error) {
	splitArguments, err := savedJVMOptions(connection)
	if err != nil {
		return nil, fmt.Errorf("saved client arguments: %w", err)
	}
	return append(splitArguments, commonJVMOptions(connection)...), nil
}

func commonJVMOptions(connection ClusterConnection) []string {
	return append([]string{"-cp", connection.BaseClasspath}, commonRuntimeOptions(connection)...)
}

func commonRuntimeOptions(connection ClusterConnection) []string {
	return []string{getPersistenceProperty(connection.PersistenceMode),
		getLogLevelProperty(logLevelParam), "-Dcom.oracle.coherence.common.util.Threads.dumpLocks=FULL"}
}

func startClient(cmd *cobra.Command, connection ClusterConnection, class string, extraArgs ...string) error {
	var (
		result         string
		startupProfile = getProfileValue(profileValueParam)
		profileArgs    []string
	)
	arguments, err := getCommonArguments(connection)
	if err != nil {
		return err
	}

	// check if any profiles have been specified
	if profileValueParam != "" {
		// this profile param has already been validated
		profileArgs = strings.Split(startupProfile, " ")
		arguments = append(arguments, profileArgs...)
	}

	if extendClientParam && grpcClientParam {
		return fmt.Errorf("-X and -G options are exclusive")
	}

	arguments = append(arguments, getClientArgs(class, class)...)
	if len(extraArgs) != 0 {
		arguments = append(arguments, extraArgs...)
	}

	cmd.Printf("Starting client %s...\n", class)
	if Config.Debug {
		fields := []zapcore.Field{
			zap.String("type", javaExec),
			zap.String("class", class),
			zap.String("arguments", fmt.Sprintf("%v", arguments)),
		}
		Logger.Info("Starting Client", fields...)
	}

	process := exec.Command(javaExec, arguments...) // #nosec G204
	process.Stdout = cmd.OutOrStdout()
	process.Stdin = cmd.InOrStdin()
	process.Stderr = cmd.ErrOrStderr()
	err = process.Start()
	if err != nil {
		return utils.GetError(fmt.Sprintf("unable to start %s: %v", class, result), err)
	}

	return process.Wait()
}

func getCacheServerArgs(connection ClusterConnection, member string, httpPort int32, management ClusterManagement, jmxOwner bool) []string {
	var (
		baseArgs  = make([]string, 0)
		heap      string
		mainClass = connection.StartupClass
	)

	// override only if we specify the class
	if serverStartClassParam != "" {
		mainClass = serverStartClassParam
	}

	baseArgs = append(baseArgs, "-Dcoherence.management.http.address="+management.RESTAddress)
	if httpPort != -1 {
		baseArgs = append(baseArgs, "-Dcoherence.management.http=all", fmt.Sprintf("-Dcoherence.management.http.port=%d", httpPort), "-Dcoherence.management=all")
	} else {
		baseArgs = append(baseArgs, "-Dcoherence.management.http=none")
	}
	if jmxOwner && management.JMXPort > 0 {
		baseArgs = append(baseArgs, "-Dcom.sun.management.jmxremote", "-Dcom.sun.management.jmxremote.ssl=false",
			"-Dcom.sun.management.jmxremote.authenticate=false", "-Dcom.sun.management.jmxremote.host="+management.JMXAddress,
			fmt.Sprintf("-Dcom.sun.management.jmxremote.port=%d", management.JMXPort),
			fmt.Sprintf("-Dcom.sun.management.jmxremote.rmi.port=%d", management.JMXPort), "-Djava.rmi.server.hostname="+management.JMXHost)
	}

	for _, property := range []struct{ key, value string }{
		{"machine", machineParam}, {"rack", rackParam}, {"site", siteParam}, {"role", roleParam},
	} {
		if property.value != "" {
			baseArgs = append(baseArgs, "-Dcoherence."+property.key+"="+property.value)
		}
	}
	heap = heapMemoryParam
	if heapMemoryParam == defaultHeap && Config.DefaultHeap != "" {
		heap = Config.DefaultHeap
	}

	baseArgs = append(baseArgs, "-Xms"+heap, "-Xmx"+heap)

	// default the main class if not specified
	if mainClass == "" {
		mainClass = utils.GetCoherenceMainClass(connection.ClusterVersion)
	}

	return append(baseArgs, getMemberProperty(member), mainClass)
}

// getClientArgs returns the arguments for starting a Coherence process such as
// console or CohQL.
func getClientArgs(member, class string) []string {
	baseArgs := make([]string, 0)
	baseArgs = append(baseArgs, "-Xms"+heapMemoryParam, "-Xmx"+heapMemoryParam)

	if extendClientParam {
		// only works with default Cache config
		baseArgs = append(baseArgs, "-Dcoherence.client=remote", "-Dcoherence.tcmpenabled=false")
	}
	if grpcClientParam {
		// only works with default Cache config
		baseArgs = append(baseArgs, "-Dcoherence.profile=thin", "-Dcoherence.client=grpc", "-Dcoherence.tcmpenabled=false")
	}

	baseArgs = append(baseArgs, getMemberProperty(member), "-Dcoherence.distributed.localstorage=false", class)

	// check -f option to CohQL
	if fileNameParam != "" {
		baseArgs = append(baseArgs, "-f", fileNameParam, "-c")
	}

	if statementParam != "" {
		baseArgs = append(baseArgs, "-l", statementParam, "-c")
	}

	return baseArgs
}

func getMemberProperty(member string) string {
	return fmt.Sprintf("-Dcoherence.member=%s", member)
}

func getPersistenceProperty(persistenceMode string) string {
	return fmt.Sprintf("-Dcoherence.distributed.persistence.mode=%s", persistenceMode)
}

func getCacheConfigProperty() string {
	if cacheConfigParam == "" {
		return ""
	}
	return fmt.Sprintf(" -Dcoherence.cacheconfig=%s", cacheConfigParam)
}

func getOperationalOverrideConfigProperty() string {
	if operationalConfigParam == "" {
		return ""
	}
	return fmt.Sprintf(" -Dcoherence.override=%s", operationalConfigParam)
}

func validateOverrideParams() error {
	if cacheConfigParam != "" {
		if !isRegularFile(cacheConfigParam) {
			return fmt.Errorf("cahce config file %s does not exist", cacheConfigParam)
		}
	}
	if operationalConfigParam != "" {
		if !isRegularFile(operationalConfigParam) {
			return fmt.Errorf("operational override file %s does not exist", operationalConfigParam)
		}
	}
	return nil
}

func isRegularFile(file string) bool {
	stat, err := os.Stat(file)
	if err != nil || stat == nil {
		return false
	}
	return stat.Mode().IsRegular()
}

func getLogLevelProperty(logLevel int32) string {
	return fmt.Sprintf("-Dcoherence.log.level=%d", logLevel)
}

// getRunningProcesses returns the running process ID's for a cluster
// connection from a dataFetcher. Returns an empty slice if none are running.
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
		pid, err := strconv.Atoi(v.ProcessName)
		if err == nil && pid > 0 {
			PIDS = append(PIDS, pid)
		}
	}

	return PIDS
}

// Lifecycle discovery fails closed. A failed query, missing/empty items, or
// even one invalid PID cannot establish the original process set for restart.
// REST-reported local PIDs remain trusted; this is not a process registry.
func discoverLifecycleMembers(dataFetcher fetcher.Fetcher) ([]int, error) {
	result, err := dataFetcher.GetMemberDetailsJSON(false)
	if err != nil {
		return nil, fmt.Errorf("query members: %w", err)
	}
	var members config.Members
	if err = json.Unmarshal(result, &members); err != nil {
		return nil, fmt.Errorf("decode members: %w", err)
	}
	if len(members.Members) == 0 {
		return nil, fmt.Errorf("member snapshot is empty")
	}
	pids := make([]int, 0, len(members.Members))
	seen := map[int]bool{}
	for _, member := range members.Members {
		pid, err := strconv.Atoi(member.ProcessName)
		if err != nil || pid <= 0 {
			return nil, fmt.Errorf("invalid member PID %q", member.ProcessName)
		}
		if seen[pid] {
			return nil, fmt.Errorf("duplicate member PID %d", pid)
		}
		seen[pid] = true
		pids = append(pids, pid)
	}
	return pids, nil
}

func checkOperation(connection ClusterConnection, operation string) error {
	if connection.ManuallyCreated {
		return nil
	}
	return fmt.Errorf("cluster %s was not manually created, unable to perform operation %s", connection.Name, operation)
}

// getTransitiveClasspath returns the transitive classpath by using mvn dependency:build-classpath,
// outputting to temp file and reading in.
func getTransitiveClasspath(groupID, artifact, version string) (string, error) {
	var (
		err       error
		pomFile   string
		file      *os.File
		classpath string
		output    string
		data      []byte
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

	// now we have a valid file, read it in
	data, err = os.ReadFile(file.Name())
	if err != nil {
		return output, utils.GetError("unable to read from temp file", err)
	}

	return string(data), nil
}

func getDependencyArgs(groupID, artifact, version string) []string {
	var (
		gavArgs    = getGAVArgs(groupID, artifact, version)
		transitive = "true"
	)

	if artifact == "coherence" {
		transitive = stringFalse
	}
	// don't bring any additional deps in
	return append(gavArgs, "dependency:get", "-Dtransitive="+transitive)
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

// getMavenClasspath returns the maven classpath for the given GAV and fileType.
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

// runCommand waits for a dependency tool and captures its output.
func runCommand(command string, arguments []string) (string, error) {
	var (
		err    error
		result []byte
	)
	if Config.Debug {
		fields := []zapcore.Field{
			zap.String("command", command),
			zap.String("arguments", strings.Join(arguments, " ")),
		}
		Logger.Info("Run command", fields...)
	}

	startTime := time.Now()
	process := exec.Command(command, arguments...)

	// wait for result
	result, err = process.CombinedOutput()

	if Config.Debug {
		fields := []zapcore.Field{
			zap.String("time", fmt.Sprintf("%v", time.Since(startTime))),
		}
		Logger.Info("Duration", fields...)
	}

	if err != nil {
		return "", utils.GetError(fmt.Sprintf("unable to start process %s, %v\n%s", command, process, string(result)), err)
	}

	return string(result), nil
}

func getLogFile(clusterName, processName string) (string, error) {
	if err := validateClusterLogName(processName); err != nil {
		return "", err
	}
	root, err := openClusterLogs(clusterName)
	if err != nil {
		return "", err
	}
	if err := root.Close(); err != nil {
		return "", err
	}
	return filepath.Join(logsDirectory, clusterName, processName+".log"), nil
}

// Cluster and member names are single path components, including on a host
// loading configuration written on another OS. The selected log root may be
// absolute, relative, or a symlink chosen by the user.
func validateClusterLogName(name string) error {
	if name == "." || !filepath.IsLocal(name) || strings.ContainsAny(name, "/\\:\x00") {
		return fmt.Errorf("invalid cluster log path component %q", name)
	}
	return nil
}

// openClusterLogs confines directory creation and symlink resolution to the
// configured log root, then pins the cluster directory for log operations.
func openClusterLogs(clusterName string) (*os.Root, error) {
	if err := validateClusterLogName(clusterName); err != nil {
		return nil, err
	}
	logs, err := os.OpenRoot(logsDirectory)
	if err != nil {
		return nil, err
	}
	if err := logs.Mkdir(clusterName, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, errors.Join(err, logs.Close())
	}
	cluster, openErr := logs.OpenRoot(clusterName)
	closeErr := logs.Close()
	if openErr != nil {
		return nil, errors.Join(openErr, closeErr)
	}
	if closeErr != nil {
		return nil, errors.Join(closeErr, cluster.Close())
	}
	return cluster, nil
}

func backupClusterLogFile(clusterName, processName string) (backedUp bool, err error) {
	if err = validateClusterLogName(processName); err != nil {
		return false, err
	}
	root, err := openClusterLogs(clusterName)
	if err != nil {
		return false, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	name := processName + ".log"
	input, err := root.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("unable to read log file %s: %w", name, err)
	}
	if err := root.WriteFile(name+".old", input, 0600); err != nil {
		return false, fmt.Errorf("unable to write backup log file %s: %w", name, err)
	}
	return true, nil
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

type clusterSummaryInfo struct {
	machines                   []config.Machine
	finalSummariesDestinations []config.FederationSummary
	finalSummariesOrigins      []config.FederationSummary
	topicsDetails              config.TopicDetails
	topicsMemberDetails        []config.TopicsMemberDetail
	topicsSubscriberDetails    []config.TopicsSubscriberDetail
	cacheSummaryDetail         []config.CacheSummaryDetail
	serviceList                []string
	federatedServices          []string
	clusterResult              []byte
	membersResult              []byte
	servicesResult             []byte
	proxyResults               []byte
	reportersResult            []byte
	ramResult                  []byte
	flashResult                []byte
	cachesResult               []byte
	http                       []byte
	executorsResult            []byte
	machinesData               []byte
	storageData                []byte
	healthResult               []byte
	executors                  config.Executors
}

// the following constants determine if the retrieveClusterSummary should retrieve the data
const (
	memberPanelData       = "members"
	cachesPanelData       = "caches"
	elasticDataPanelData  = "elastic-data"
	executorsPanelData    = "executors"
	healthPanelData       = "health"
	federationPanelData   = "federation"
	proxiesPanelData      = "proxies"
	storagePanelData      = "storage"
	servicesPanelData     = "services"
	reportersPanelData    = "reporters"
	topicsPanelData       = "topics"
	httpSessionsPanelData = "http-sessions"
)

// retrieveClusterSummary retrieves all the required information used by various commands.
// panelData contains an array of data items to retrieve. If len(panelData) == 0 then this
// means to retrieve all data, otherwise only retrieve data which is contained in data array.
// this is to lessen the load on management HTTP server when using monitor cluster command.
func retrieveClusterSummary(dataFetcher fetcher.Fetcher, panelData ...string) (clusterSummaryInfo, []error) {
	var (
		errorSink      = createErrorSink()
		waitGroupCount = 13
		wg             sync.WaitGroup
		mutexRetrieve  sync.Mutex
		result         clusterSummaryInfo
	)

	// retrieve cluster details first so if we are connected
	// to WLS or need authentication, this can be done first
	data1, err1 := dataFetcher.GetClusterDetailsJSON()
	if err1 != nil {
		return result, []error{err1}
	}
	result.clusterResult = data1

	// retrieve the details for the cluster in parallel
	wg.Add(waitGroupCount)

	go func() {
		defer wg.Done()
		var (
			err  error
			data []byte
		)
		if shouldRetrieve(memberPanelData, panelData) {
			data, err = dataFetcher.GetMemberDetailsJSON(false)
			if err != nil {
				errorSink.AppendError(err)
				return
			}
			mutexRetrieve.Lock()
			defer mutexRetrieve.Unlock()
			result.membersResult = data
		}
	}()

	go func() {
		defer wg.Done()
		var (
			data     []byte
			machines []config.Machine
		)
		if shouldRetrieve(memberPanelData, panelData) {
			machinesMap, err := GetMachineList(dataFetcher)
			if err != nil {
				errorSink.AppendError(err)
				return
			}

			machines, err = getMachines(machinesMap, dataFetcher)
			if err != nil {
				errorSink.AppendError(err)
				return
			}

			data, err = getOSJson(machinesMap, dataFetcher)
			if err != nil {
				errorSink.AppendError(err)
				return
			}

			mutexRetrieve.Lock()
			defer mutexRetrieve.Unlock()
			result.machines = machines
			result.machinesData = data
		}
	}()

	go func() {
		defer wg.Done()
		data, err := dataFetcher.GetServiceDetailsJSON()
		if err != nil {
			errorSink.AppendError(err)
			return
		}

		if shouldRetrieve(servicesPanelData, panelData) {
			var services = config.ServicesSummaries{}
			err = json.Unmarshal(data, &services)
			if err != nil {
				errorSink.AppendError(err)
				return
			}

			serviceList := GetListOfCacheServices(services)

			allCachesSummary, err := getCaches(serviceList, dataFetcher)
			if err != nil {
				errorSink.AppendError(err)
				return
			}

			mutexRetrieve.Lock()
			defer mutexRetrieve.Unlock()
			result.servicesResult = data
			result.cacheSummaryDetail = allCachesSummary
			result.serviceList = serviceList
		}
	}()

	go func() {
		defer wg.Done()

		if shouldRetrieve(topicsPanelData, panelData) {
			topicsDetails, err := getTopics(dataFetcher, serviceName)
			if err != nil {
				errorSink.AppendError(err)
				return
			}

			topicsMemberDetails, err := getTopicsMembers(dataFetcher, topicsDetails)
			if err != nil {
				errorSink.AppendError(err)
				return
			}

			topicsSubscriberDetails, err := getTopicsSubscribers(dataFetcher, topicsDetails)
			if err != nil {
				errorSink.AppendError(err)
				return
			}

			enrichTopicsSummary(&topicsDetails, topicsMemberDetails, topicsSubscriberDetails)

			mutexRetrieve.Lock()
			defer mutexRetrieve.Unlock()
			result.topicsDetails = topicsDetails
			result.topicsMemberDetails = topicsMemberDetails
			result.topicsSubscriberDetails = topicsSubscriberDetails
		}
	}()

	go func() {
		defer wg.Done()
		if shouldRetrieve(healthPanelData, panelData) {
			data, err := dataFetcher.GetMembersHealth()
			if err != nil {
				errorSink.AppendError(err)
				return
			}

			mutexRetrieve.Lock()
			defer mutexRetrieve.Unlock()
			result.healthResult = data
		}
	}()

	go func() {
		defer wg.Done()
		if shouldRetrieve(storagePanelData, panelData) {
			data, err := dataFetcher.GetStorageDetailsJSON()
			if err != nil {
				errorSink.AppendError(err)
			}

			mutexRetrieve.Lock()
			defer mutexRetrieve.Unlock()
			result.storageData = data
		}
	}()

	go func() {
		defer wg.Done()
		if shouldRetrieve(proxiesPanelData, panelData) {
			data, err := dataFetcher.GetProxySummaryJSON()
			if err != nil {
				errorSink.AppendError(err)
			}

			mutexRetrieve.Lock()
			defer mutexRetrieve.Unlock()
			result.proxyResults = data
		}
	}()

	go func() {
		defer wg.Done()
		if !verboseOutput && !monitorCluster {
			return
		}
		if shouldRetrieve(reportersPanelData, panelData) {
			data, err := dataFetcher.GetReportersJSON()
			if err != nil {
				errorSink.AppendError(err)
			}

			mutexRetrieve.Lock()
			defer mutexRetrieve.Unlock()
			result.reportersResult = data
		}
	}()

	go func() {
		defer wg.Done()
		if shouldRetrieve(federationPanelData, panelData) {
			federatedServices, err := GetFederatedServices(dataFetcher)
			if err != nil {
				errorSink.AppendError(err)
				return
			}
			finalSummariesDestinations, err := getFederationSummaries(federatedServices, outgoing, dataFetcher)
			if err != nil {
				errorSink.AppendError(err)
				return
			}
			finalSummariesOrigins, err := getFederationSummaries(federatedServices, incoming, dataFetcher)
			if err != nil {
				errorSink.AppendError(err)
				return
			}

			mutexRetrieve.Lock()
			defer mutexRetrieve.Unlock()
			result.federatedServices = federatedServices
			result.finalSummariesDestinations = finalSummariesDestinations
			result.finalSummariesOrigins = finalSummariesOrigins
		}
	}()

	go func() {
		defer wg.Done()
		if shouldRetrieve(elasticDataPanelData, panelData) {
			data, err := dataFetcher.GetElasticDataDetails("flash")
			if err != nil {
				errorSink.AppendError(err)
			}

			mutexRetrieve.Lock()
			defer mutexRetrieve.Unlock()
			result.flashResult = data
		}
	}()

	go func() {
		defer wg.Done()
		if shouldRetrieve(elasticDataPanelData, panelData) {
			data, err := dataFetcher.GetElasticDataDetails("ram")
			if err != nil {
				errorSink.AppendError(err)
			}

			mutexRetrieve.Lock()
			defer mutexRetrieve.Unlock()
			result.ramResult = data
		}
	}()

	go func() {
		defer wg.Done()
		if shouldRetrieve(httpSessionsPanelData, panelData) {
			data, err := dataFetcher.GetHTTPSessionDetailsJSON()
			if err != nil {
				errorSink.AppendError(err)
			}

			mutexRetrieve.Lock()
			defer mutexRetrieve.Unlock()
			result.http = data
		}
	}()

	go func() {
		defer wg.Done()
		if shouldRetrieve(executorsPanelData, panelData) {
			data, err := getExecutorDetails(dataFetcher, true)
			if err != nil {
				errorSink.AppendError(err)
			}

			mutexRetrieve.Lock()
			defer mutexRetrieve.Unlock()
			result.executors = data
		}
	}()

	// wait for all data fetchers requests to complete
	wg.Wait()

	return result, errorSink.GetErrors()
}

// shouldRetrieve returned a boolean indicating if the data for panel should be retrieved.
func shouldRetrieve(panel string, panelData []string) bool {
	if len(panelData) == 0 {
		return true
	}
	return utils.SliceContains(panelData, panel)
}
