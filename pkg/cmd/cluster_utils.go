/*
 * Copyright (c) 2022, 2024 Oracle and/or its affiliates.
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
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net"
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
		{GroupID: "org.jline", Artifact: "jline", IsCoherence: false, Version: "3.25.0"},
	}

	// list of additional coherence artifacts
	validCoherenceArtifacts = []string{"coherence-cdi-server", "coherence-cdi", "coherence-concurrent", "coherence-grpc-proxy",
		"coherence-grpc", "coherence-helidon-client", "coherence-helidon-grpc-proxy", "coherence-http-netty", "coherence-java-client",
		"coherence-jcache", "coherence-jpa", "coherence-management", "coherence-micrometer", "coherence-mp-config", "coherence-metrics",
		"coherence-mp-metrics", "coherence-rest"}
)

// checkCreateRequirements validates that all the necessary requirements are fulfilled
// for creating a cluster. This includes mvn and java executables. Nil is returned to
// indicate everything is ok, otherwise an error is returned.
func checkRuntimeRequirements() error {
	var (
		err error
	)

	processJava := exec.Command(javaExec, "--version")
	if err = processJava.Start(); err != nil {
		return utils.GetError(fmt.Sprintf("unable to get Java version using %s --version: %v", javaExec, processJava), err)
	}

	return nil
}

// checkDepsRequirements checks for either mvn or gradle depending upon the
// setting of Config.useGradle.
func checkDepsRequirements() error {
	var (
		err      error
		execName = getExecType()
	)

	proc := exec.Command(execName, "-v")
	if err = proc.Start(); err != nil {
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
// scaling a cluster, otherwise we are starting one.
func startCluster(cmd *cobra.Command, connection ClusterConnection, serverCount, existingCount int32) error {
	var (
		err              error
		mgmtPort         = connection.ManagementPort
		counter          int32
		metricsStartPort = metricsStartPortParam
		healthStartPort  = healthStartPortParam
		startupProfile   = getProfileValue(profileValueParam)
		profileArgs      = make([]string, 0)
		startupDelay     int64
	)

	startupDelay, err = utils.GetStartupDelayInMillis(startupDelayParam)
	if err != nil {
		return err
	}

	// check if any profiles have been specified
	if profileValueParam != "" {
		// this profile param has already been validated
		profileArgs = strings.Split(startupProfile, " ")
	}

	// if we are scaling then set the http port to -1 so no more management servers are started
	if existingCount > 0 {
		mgmtPort = -1
	}

	if err = checkOperation(connection, startClusterCommand); err != nil {
		return err
	}

	if jmxRemotePortParam > 0 {
		// validate jmx remote port
		if err = utils.ValidatePort(jmxRemotePortParam); err != nil {
			return err
		}
	}

	for counter = existingCount; counter < serverCount+existingCount; counter++ {
		var (
			member        = fmt.Sprintf("storage-%d", counter)
			arguments     = getCommonArguments(connection)
			memberLogFile string
		)

		// check if metrics start port specified
		if metricsStartPort > 0 {
			metricsArgs := []string{"-Dcoherence.metrics.http.enabled=true", fmt.Sprintf("-Dcoherence.metrics.http.port=%d", metricsStartPort)}
			metricsStartPort++
			arguments = append(arguments, metricsArgs...)
		}

		// if -F is specified, and we are the first management or -F is not specified then add profile
		if (profileFirstParam && mgmtPort != -1) || (!profileFirstParam && len(profileArgs) > 0) {
			arguments = append(arguments, profileArgs...)
		}

		// check if health start port specified
		if healthStartPort > 0 {
			arguments = append(arguments, fmt.Sprintf("-Dcoherence.health.http.port=%d", healthStartPort))
			healthStartPort++
		}

		arguments = append(arguments, getCacheServerArgs(connection, member, mgmtPort, connection.ClusterVersion)...)

		// reset so only first member has management enabled
		mgmtPort = -1

		memberLogFile, err = getLogFile(connection.Name, member)
		if err != nil {
			return err
		}

		if backupLogFilesParam {
			// backup log files before starting cluster
			backupLogFile := memberLogFile + ".old"
			input, err1 := os.ReadFile(memberLogFile)
			if err1 != nil {
				// no log file so just ignore error as it may have been removed
			} else {
				err1 = os.WriteFile(backupLogFile, input, 0600)
				if err1 != nil {
					return fmt.Errorf("unable to write to log file %s: %v", backupLogFile, err1)
				}
				cmd.Printf("creating backup log file %s\n", backupLogFile)
			}
		}

		cmd.Printf("Starting cluster member %s...\n", member)
		_, err = runCommandAsync(javaExec, memberLogFile, arguments)
		if err != nil {
			return utils.GetError(fmt.Sprintf("unable to start member %s", member), err)
		}

		if startupDelay > 0 {
			time.Sleep(time.Duration(startupDelay) * time.Millisecond)
		}
	}

	return nil
}

// getCommonArguments returns arguments that are common to clients and servers.
func getCommonArguments(connection ClusterConnection) []string {
	splitArguments := strings.Split(connection.Arguments, " ")
	return append(splitArguments, "-cp", connection.BaseClasspath, getPersistenceProperty(connection.PersistenceMode),
		getLogLevelProperty(logLevelParam))
}

func startClient(cmd *cobra.Command, connection ClusterConnection, class string) error {
	var (
		err            error
		result         string
		startupProfile = getProfileValue(profileValueParam)
		profileArgs    []string
		arguments      = getCommonArguments(connection)
	)

	// check if any profiles have been specified
	if profileValueParam != "" {
		// this profile param has already been validated
		profileArgs = strings.Split(startupProfile, " ")
		arguments = append(arguments, profileArgs...)
	}

	arguments = append(arguments, getClientArgs(class, class)...)

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

func getCacheServerArgs(connection ClusterConnection, member string, httpPort int32, version string) []string {
	var (
		baseArgs  = make([]string, 0)
		heap      string
		mainClass = connection.StartupClass
	)

	// override only if we specify the class
	if serverStartClassParam != "" || connection.StartupClass == "" {
		mainClass = serverStartClassParam
	}

	if httpPort != -1 {
		baseArgs = append(baseArgs, "-Dcoherence.management.http=all", fmt.Sprintf("-Dcoherence.management.http.port=%d", httpPort),
			"-Dcoherence.management=all", "-Dcom.sun.management.jmxremote.ssl=false", "-Dcom.sun.management.jmxremote",
			"-Dcom.sun.management.jmxremote.authenticate=false")

		// add remote JMX port is specified
		if jmxRemotePortParam > 0 {
			rmiHost := jmxRemoteHostParam
			if rmiHost == "" {
				rmiHost = wkaParam
			}
			baseArgs = append(baseArgs, fmt.Sprintf("-Dcom.sun.management.jmxremote.port=%d", jmxRemotePortParam),
				fmt.Sprintf("-Dcom.sun.management.jmxremote.rmi.port=%d", jmxRemotePortParam),
				fmt.Sprintf("-Djava.rmi.server.hostname=%s", rmiHost))
		}
	}

	// if default heap is overridden, then use this
	if heapMemoryParam != defaultHeap {
		heap = heapMemoryParam
	} else {
		// if the default-heap is set in config then use this
		if Config.DefaultHeap != "" {
			heap = Config.DefaultHeap
		} else {
			heap = heapMemoryParam
		}
	}

	baseArgs = append(baseArgs, "-Xms"+heap, "-Xmx"+heap)

	// default the main class if not specified
	if mainClass == "" {
		mainClass = utils.GetCoherenceMainClass(version)
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

func runCommand(command string, arguments []string) (string, error) {
	return runCommandBase(command, "", arguments)
}

func runCommandAsync(command, logFileName string, arguments []string) (string, error) {
	return runCommandBase(command, logFileName, arguments)
}

// runCommandBase runs a command. If logFileName is supplied then this is done async and the
// processId is returned, otherwise the result of the combined stdout and stderr is returned.
func runCommandBase(command, logFileName string, arguments []string) (string, error) {
	var (
		err            error
		result         []byte
		processLogFile *os.File
	)

	if Config.Debug {
		fields := []zapcore.Field{
			zap.String("command", command),
			zap.String("arguments", strings.Join(arguments, " ")),
		}
		Logger.Info("Run command", fields...)
	}

	start := time.Now()
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

	if Config.Debug {
		fields := []zapcore.Field{
			zap.String("time", fmt.Sprintf("%v", time.Since(start))),
		}
		Logger.Info("Duration", fields...)
	}

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

// isPortUsed checks to see if a port on localhost can be connected to.
func isPortUsed(managementPort int32) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", localHost, managementPort), time.Duration(fetcher.RequestTimeout)*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	// if err was nil it means we were successful in connecting to the port
	// as there was something running on it and listening
	return err == nil
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

// retrieveClusterSummary retrieves all the required information used by various commands.
func retrieveClusterSummary(dataFetcher fetcher.Fetcher) (clusterSummaryInfo, []error) {
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
		data, err = dataFetcher.GetMemberDetailsJSON(false)
		if err != nil {
			errorSink.AppendError(err)
			return
		}
		mutexRetrieve.Lock()
		defer mutexRetrieve.Unlock()
		result.membersResult = data
	}()

	go func() {
		defer wg.Done()
		var (
			data     []byte
			machines []config.Machine
		)
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
	}()

	go func() {
		defer wg.Done()
		data, err := dataFetcher.GetServiceDetailsJSON()
		if err != nil {
			errorSink.AppendError(err)
			return
		}

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
	}()

	go func() {
		defer wg.Done()

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
	}()

	go func() {
		defer wg.Done()
		data, err := dataFetcher.GetMembersHealth()
		if err != nil {
			errorSink.AppendError(err)
			return
		}

		mutexRetrieve.Lock()
		defer mutexRetrieve.Unlock()
		result.healthResult = data
	}()

	go func() {
		defer wg.Done()
		data, err := dataFetcher.GetStorageDetailsJSON()
		if err != nil {
			errorSink.AppendError(err)
		}

		mutexRetrieve.Lock()
		defer mutexRetrieve.Unlock()
		result.storageData = data
	}()

	go func() {
		defer wg.Done()
		data, err := dataFetcher.GetProxySummaryJSON()
		if err != nil {
			errorSink.AppendError(err)
		}

		mutexRetrieve.Lock()
		defer mutexRetrieve.Unlock()
		result.proxyResults = data
	}()

	go func() {
		defer wg.Done()
		if !verboseOutput && !monitorCluster {
			return
		}
		data, err := dataFetcher.GetReportersJSON()
		if err != nil {
			errorSink.AppendError(err)
		}

		mutexRetrieve.Lock()
		defer mutexRetrieve.Unlock()
		result.reportersResult = data
	}()

	go func() {
		defer wg.Done()
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
	}()

	go func() {
		defer wg.Done()
		data, err := dataFetcher.GetElasticDataDetails("flash")
		if err != nil {
			errorSink.AppendError(err)
		}

		mutexRetrieve.Lock()
		defer mutexRetrieve.Unlock()
		result.flashResult = data
	}()

	go func() {
		defer wg.Done()
		data, err := dataFetcher.GetElasticDataDetails("ram")
		if err != nil {
			errorSink.AppendError(err)
		}

		mutexRetrieve.Lock()
		defer mutexRetrieve.Unlock()
		result.ramResult = data
	}()

	go func() {
		defer wg.Done()
		data, err := dataFetcher.GetHTTPSessionDetailsJSON()
		if err != nil {
			errorSink.AppendError(err)
		}

		mutexRetrieve.Lock()
		defer mutexRetrieve.Unlock()
		result.http = data
	}()

	go func() {
		defer wg.Done()
		data, err := getExecutorDetails(dataFetcher, true)
		if err != nil {
			errorSink.AppendError(err)
		}

		mutexRetrieve.Lock()
		defer mutexRetrieve.Unlock()
		result.executors = data
	}()

	// wait for all data fetchers requests to complete
	wg.Wait()

	return result, errorSink.GetErrors()
}
