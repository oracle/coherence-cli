/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"errors"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

//
// main Cobra command entrypoint

// Global flags

var OutputFormat string
var clusterConnection string
var serviceName string
var watchEnabled bool
var watchDelay int32
var readPassStdin bool

// various command constants
const clusterConnectionDescription = "cluster connection name. (not required if context is set)"
const connectionNameOption = "connection"
const clusterNameOptionShort = "c"

const serviceNameDescription = "Service name"
const serviceNameOption = "service"
const serviceNameOptionShort = "s"

const userNameDescription = "basic auth username if authentication is required"
const usernameOption = "username"
const usernameShort = "U"

const clusterKey = "clusters"
const currentContextKey = "currentContext"
const debugContextKey = "debug"
const ignoreCertsContextKey = "ignoreInvalidCerts"
const requestTimeoutKey = "requestTimeout"

const confirmOptionMessage = "automatically confirm the operation"
const timeoutMessage = "timeout in seconds for NS Lookup requests"

const outputFormats = "table, wide, json or jsonpath=\"...\""

// config file related

const configName = "cohctl"
const configType = "yaml"
const configDir = ".cohctl"
const logFile = "cohctl.log"

var (
	// config file
	cfgFile string

	// configuration directory
	cfgDirectory string

	// Config is the CLI config
	Config CoherenceCLIConfig

	// Username contains the current username
	Username string

	// UsingContext indicates if we are using a context via the set context command
	UsingContext bool

	// path to the logfile
	logFilePath string

	Logger *zap.Logger

	// Version is the cohctl version injected by the Go linker at build time.
	Version string
	// Commit is the git commit hash injected by the Go linker at build time.
	Commit string
	// Date is the build timestamp injected by the Go linker at build time.
	Date string
)

// CoherenceCLIConfig describes the details stored in the .cohctl.yaml
type CoherenceCLIConfig struct {
	Version            string              `json:"version"`
	CurrentContext     string              `json:"currentContext"`
	Clusters           []ClusterConnection `mapstructure:"clusters"`
	Debug              bool                `json:"debug"`
	RequestTimeout     int32               `json:"requestTimeout"`
	IgnoreInvalidCerts bool                `json:"ignoreInvalidCerts"`
}

// ClusterConnection describes an individual connection to a cluster
type ClusterConnection struct {
	Name                 string `json:"name"` // the name the user gives to the cluster connection
	DiscoveryType        string `json:"discoveryType"`
	ConnectionType       string `json:"connectionType"`
	ConnectionURL        string `json:"url"`
	NameServiceDiscovery string `json:"nameServiceDiscovery"`
	ClusterVersion       string `json:"clusterVersion"`
	ClusterName          string `json:"clusterName"` // the actual cluster name
	ClusterType          string `json:"clusterType"`
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = createRootCommand()

// createRootCommand creates the root command off which all others are places
func createRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:          "cohctl",
		Short:        "Coherence CLI",
		SilenceUsage: true,
		Long: `The Coherence Command Line Interface (CLI) provides a way to
interact with, and monitor Coherence clusters via a terminal-based interface.`,
	}
	return root
}

// Execute run the root command
func Execute(version string, date string, commit string) {
	Version = version
	Date = date
	Commit = commit
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// SetRootCommandFlags sets all the persistent root command flags
func SetRootCommandFlags(command *cobra.Command) {
	// Global flags for all commands
	command.PersistentFlags().StringVar(&cfgDirectory, "config-dir", "", "config directory (default is $HOME/.cohctl)")
	command.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cohctl/cohctl.yaml)")
	command.PersistentFlags().StringVarP(&OutputFormat, "output", "o", constants.TABLE, "output format: "+outputFormats)
	command.PersistentFlags().BoolVarP(&watchEnabled, "watch", "w", false, "watch output (only available for get commands)")
	command.PersistentFlags().BoolVarP(&readPassStdin, "stdin", "i", false, "read password from stdin")
	command.PersistentFlags().Int32VarP(&watchDelay, "delay", "d", 5, "delay for watching in seconds")
	command.PersistentFlags().StringVarP(&Username, usernameOption, usernameShort, "", userNameDescription)
	command.PersistentFlags().StringVarP(&clusterConnection, connectionNameOption, clusterNameOptionShort, "", clusterConnectionDescription)
}

func init() {
	cobra.OnInitialize(initConfig)
	Initialize(rootCmd)
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
}

// InitConfig reads in config file and ENV variables if set.
func initConfig() {
	var (
		err  error
		home string
	)

	if cfgDirectory != "" {
		// --config-dir set so ensure we create the directory if it doesn't exist
		if stat, err := os.Stat(cfgDirectory); err != nil {
			if os.IsNotExist(err) {
				err = os.Mkdir(cfgDirectory, 0700)
				if err != nil {
					rootCmd.Println("unable to create config directory " + cfgDirectory + " : " + err.Error())
					os.Exit(1)
				}
			} else if !stat.IsDir() {
				rootCmd.Println("config directory specified is not a directory " + cfgDirectory)
				os.Exit(1)
			}
		}
	} else {
		// No configuration directory set so use the default of $HOME/.cohctl
		home, err = os.UserHomeDir()
		cobra.CheckErr(err)

		// config will be stored in cfgDirectory
		cfgDirectory = filepath.Join(home, configDir)

		if err = utils.EnsureDirectory(cfgDirectory); err != nil {
			rootCmd.Println(err)
			os.Exit(1)
		}
	}

	// initialize logging
	Logger, err = initLogging(cfgDirectory)
	if err != nil {
		rootCmd.Println(utils.GetError("unable to initialize logger", err))
		os.Exit(1)
	}

	defer func() {
		_ = Logger.Sync()
	}()

	fetcher.Logger = Logger
	fetcher.UnableToFindClusterMsg = UnableToFindClusterMsg
	fetcher.ReadPassStdin = readPassStdin
	utils.Logger = Logger

	fields := []zapcore.Field{
		zap.String("version", Version),
		zap.String("date", Date),
		zap.String("commit", Commit),
		zap.String("os", runtime.GOOS),
		zap.String("platform", runtime.GOARCH),
	}

	Logger.Info("CLI Details", fields...)

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in cfgDirectory + ".cohctl.yaml"
		viper.AddConfigPath(cfgDirectory)
		viper.SetConfigType(configType)
		viper.SetConfigName(configName)
		cfgFile = filepath.Join(cfgDirectory, configName+"."+configType)
	}

	viper.AutomaticEnv() // read in environment variables that match

	if err := viper.ReadInConfig(); err != nil {
		_, foundError := err.(viper.ConfigFileNotFoundError)
		_, pathError := err.(*os.PathError)

		if foundError || pathError {
			// It doesn't exist, so create it
			if _, err := os.Create(cfgFile); err != nil {
				rootCmd.Println(utils.GetError("unable to create config file "+cfgFile, err))
				os.Exit(1)
			}

			// config file not found - create a default one
			Config := CoherenceCLIConfig{Version: Version, Clusters: make([]ClusterConnection, 0),
				Debug: false, RequestTimeout: 30, IgnoreInvalidCerts: false}
			viper.Set("version", Config.Version)
			viper.Set(currentContextKey, Config.CurrentContext)
			viper.Set(debugContextKey, Config.Debug)
			viper.Set(ignoreCertsContextKey, Config.IgnoreInvalidCerts)
			viper.Set(requestTimeoutKey, Config.RequestTimeout)

			err = WriteConfig()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if Config.Debug {
				Logger.Info("Creating new config file", []zapcore.Field{zap.String("configFile", cfgFile)}...)
			}
		} else {
			// Config file was found but another error was produced
			rootCmd.Println("unable to read config " + cfgFile)
			os.Exit(1)
		}
	} else {
		// load the config
		if err := viper.Unmarshal(&Config); err != nil {
			rootCmd.Println(err)
			os.Exit(1)
		}
	}

	Logger.Info("Configuration loaded", []zapcore.Field{
		zap.String("configFile", cfgFile),
		zap.String("configDir", cfgDirectory),
	}...)

	// check for newer version of cohctl and carry out any upgrade tasks
	if Config.Version != Version {
		Logger.Info("Upgrading cohctl version", []zapcore.Field{
			zap.String("oldVersion", Config.Version),
			zap.String("newVersion", Version)}...)

		// carry out any upgrade logic here
		Config.Version = Version
		viper.Set("version", Config.Version)
		err = WriteConfig()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	fetcher.DebugEnabled = Config.Debug
	fetcher.IgnoreInvalidCerts = Config.IgnoreInvalidCerts
	utils.DebugEnabled = Config.Debug
	fetcher.RequestTimeout = Config.RequestTimeout

	if Config.IgnoreInvalidCerts {
		msg := "WARNING: SSL Certificate validation has been explicitly disabled"
		Logger.Info(msg)
		fmt.Fprintln(os.Stderr, msg)
	}
}

// GetClusterConnection returns the URL for a given a cluster connection name
func GetClusterConnection(connectionName string) (bool, ClusterConnection) {
	for _, cluster := range Config.Clusters {
		if cluster.Name == connectionName {
			return true, cluster
		}
	}
	return false, ClusterConnection{}
}

// WriteConfig writes the viper config and exit if there is an error
func WriteConfig() error {
	err := viper.WriteConfig()
	if err != nil {
		return errors.New("unable to write config: " + err.Error())
	}
	return nil
}

// Initialize initializes the command hierarchy - required for tests
// if command is nil then a new command is created otherwise the existing
// one is used
func Initialize(command *cobra.Command) *cobra.Command {
	viper.Reset()
	cfgDirectory = ""
	cfgFile = ""
	Config.CurrentContext = ""
	dumpRoleName = "all"
	configureRole = ""

	Config.Clusters = make([]ClusterConnection, 0)

	if command == nil {
		command = createRootCommand()
	}

	// set command sub-flags
	SetRootCommandFlags(command)

	// version command
	command.AddCommand(versionCmd)

	// get command
	command.AddCommand(getCmd)
	getCmd.AddCommand(getContextCmd)
	getCmd.AddCommand(getClustersCmd)
	getCmd.AddCommand(getCachesCmd)
	getCmd.AddCommand(getMachinesCmd)
	getCmd.AddCommand(getMembersCmd)
	getCmd.AddCommand(getServicesCmd)
	getCmd.AddCommand(getPersistenceCmd)
	getCmd.AddCommand(getProxiesCmd)
	getCmd.AddCommand(getHTTPProxiesCmd)
	getCmd.AddCommand(getReportersCmd)
	getCmd.AddCommand(getDebugCmd)
	getCmd.AddCommand(getTimeoutCmd)
	getCmd.AddCommand(getLogsCmd)
	getCmd.AddCommand(getElasticDataCmd)
	getCmd.AddCommand(getSnapshotsCmd)
	getCmd.AddCommand(getHTTPSessionsCmd)
	getCmd.AddCommand(getTopicsCmd)
	getCmd.AddCommand(getJfrsCmd)
	getCmd.AddCommand(getIgnoreCertsCmd)
	getCmd.AddCommand(getExecutorsCmd)
	getCmd.AddCommand(getManagementCmd)
	getCmd.AddCommand(getFederationCmd)
	getCmd.AddCommand(getTracingCmd)

	// set command
	command.AddCommand(setCmd)
	setCmd.AddCommand(setContextCmd)
	setCmd.AddCommand(setDebugCmd)
	setCmd.AddCommand(setTimeoutCmd)
	setCmd.AddCommand(setIgnoreCertsCmd)
	setCmd.AddCommand(setMemberCmd)
	setCmd.AddCommand(setCacheCmd)
	setCmd.AddCommand(setManagementCmd)
	setCmd.AddCommand(setServiceCmd)
	setCmd.AddCommand(setReporterCmd)

	// clear
	command.AddCommand(clearCmd)
	clearCmd.AddCommand(clearContextCmd)

	// add
	command.AddCommand(addCmd)
	addCmd.AddCommand(addClusterCmd)

	// replicate
	command.AddCommand(replicateCmd)
	replicateCmd.AddCommand(replicateAllCmd)

	// pause
	command.AddCommand(pauseCmd)
	pauseCmd.AddCommand(pauseFederationCmd)

	// start
	command.AddCommand(startCmd)
	startCmd.AddCommand(startReporterCmd)
	startCmd.AddCommand(startJfrCmd)
	startCmd.AddCommand(startFederationCmd)
	startCmd.AddCommand(startServiceCmd)

	// stop
	command.AddCommand(stopCmd)
	stopCmd.AddCommand(stopReporterCmd)
	stopCmd.AddCommand(stopJfrCmd)
	stopCmd.AddCommand(stopFederationCmd)
	stopCmd.AddCommand(stopServiceCmd)

	// dump
	command.AddCommand(dumpCmd)
	dumpCmd.AddCommand(dumpJfrCmd)
	dumpCmd.AddCommand(dumpClusterHeapCmd)

	// configure
	command.AddCommand(configureCmd)
	configureCmd.AddCommand(configureTracingCmd)

	// log
	command.AddCommand(logCmd)
	logCmd.AddCommand(logClusterStateCmd)

	// remove
	command.AddCommand(removeCmd)
	removeCmd.AddCommand(removeClusterCmd)
	removeCmd.AddCommand(removeSnapshotCmd)

	// describe
	command.AddCommand(describeCmd)
	describeCmd.AddCommand(describeClusterCmd)
	describeCmd.AddCommand(describeCacheCmd)
	describeCmd.AddCommand(describeMemberCmd)
	describeCmd.AddCommand(describeProxyCmd)
	describeCmd.AddCommand(describeServiceCmd)
	describeCmd.AddCommand(describeHTTPProxyCmd)
	describeCmd.AddCommand(describeMachineCmd)
	describeCmd.AddCommand(describeReporterCmd)
	describeCmd.AddCommand(describeElasticDataCmd)
	describeCmd.AddCommand(describeHTTPSessionCmd)
	describeCmd.AddCommand(describeJfrCmd)
	describeCmd.AddCommand(describeExecutorCmd)

	// create
	command.AddCommand(createCmd)
	createCmd.AddCommand(createSnapshotCmd)

	// recover
	command.AddCommand(recoverCmd)
	recoverCmd.AddCommand(recoverSnapshotCmd)

	// archive
	command.AddCommand(archiveCmd)
	archiveCmd.AddCommand(archiveSnapshotCmd)

	// retrieve
	command.AddCommand(retrieveCmd)
	retrieveCmd.AddCommand(retrieveSnapshotCmd)
	retrieveCmd.AddCommand(retrieveThreadDumpsCmd)

	// discover
	command.AddCommand(discoverCmd)
	discoverCmd.AddCommand(discoverClustersCmd)

	// nslookup
	command.AddCommand(nsLookupCmd)

	// suspend
	command.AddCommand(suspendCmd)
	suspendCmd.AddCommand(suspendServiceCmd)

	// resume
	command.AddCommand(resumeCmd)
	resumeCmd.AddCommand(resumeServiceCmd)

	// shutdown
	command.AddCommand(shutdownCmd)
	shutdownCmd.AddCommand(shutdownServiceCmd)
	shutdownCmd.AddCommand(shutdownMemberCmd)

	return command
}

// GetDataFetcher returns a Fetcher given a cluster name
func GetDataFetcher(clusterName string) (fetcher.Fetcher, error) {
	found, connection := GetClusterConnection(clusterName)
	if !found {
		return nil, errors.New(UnableToFindClusterMsg + clusterName)
	}
	return fetcher.GetFetcherOrError(connection.ConnectionType, connection.ConnectionURL, Username,
		connection.ClusterName)
}

// GetConnectionNameFromContextOrArg returns the connection name from the '-c' option
// or the current context if set
func GetConnectionNameFromContextOrArg() (string, error) {
	// firstly check for '-c' which will override everything
	if clusterConnection != "" {
		UsingContext = false
		return clusterConnection, nil
	}

	// next check if the current context is set
	clusterNameContext := Config.CurrentContext
	if clusterNameContext != "" {
		UsingContext = true
		return clusterNameContext, nil
	}

	// otherwise, must be an error
	return "", errors.New("you must supply a connection name if you have not set the current context")
}

// GetConnectionAndDataFetcher returns the connection and dataFetcher
func GetConnectionAndDataFetcher() (string, fetcher.Fetcher, error) {
	var (
		err         error
		connection  string
		dataFetcher fetcher.Fetcher
	)

	// do validation for OutputFormat
	err = checkOutputFormat()
	if err != nil {
		return "", nil, err
	}

	// retrieve the current context or the value from "-c"
	connection, err = GetConnectionNameFromContextOrArg()
	if err != nil {
		return "", nil, err
	}

	// retrieve the data fetcher for the cluster type
	dataFetcher, err = GetDataFetcher(connection)
	return connection, dataFetcher, err
}

// checkOutputFormat checks for valid output formats
func checkOutputFormat() error {
	if OutputFormat != constants.TABLE && OutputFormat != constants.JSON && OutputFormat != constants.WIDE &&
		!strings.Contains(OutputFormat, constants.JSONPATH) {
		return fmt.Errorf("you must specify one of the following output formats: " + outputFormats)
	}
	return nil
}

// newWinFileSink returns a Zap.Sink to get around windows issue
func newWinFileSink(u *url.URL) (zap.Sink, error) {
	return os.OpenFile(u.Path[1:], os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
}

// initLogging initializes the logging
func initLogging(homeDir string) (*zap.Logger, error) {
	cfg := zap.NewDevelopmentConfig()

	logFilePath = filepath.Join(homeDir, logFile)
	if isWindows() {
		// workaround for logging issue on Windows
		err := zap.RegisterSink("winfile", newWinFileSink)
		if err != nil {
			return nil, fmt.Errorf("unable to register sink %v", err)
		}
		cfg.OutputPaths = []string{"winfile:///" + logFilePath}
	} else {
		cfg.OutputPaths = []string{logFilePath}
	}

	return cfg.Build()
}

// isWindows returns true if the OS is Windows
func isWindows() bool {
	return runtime.GOOS == "windows"
}
