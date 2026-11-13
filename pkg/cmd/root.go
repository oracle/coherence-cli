/*
 * Copyright (c) 2021, 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"errors"
	"fmt"
	"github.com/go-viper/mapstructure/v2"
	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/oracle/coherence-go-client/v2/coherence/discovery"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

//
// main Cobra command entrypoint

// Global flags

var (
	OutputFormat          string
	clusterConnection     string
	serviceName           string
	watchEnabled          bool
	watchClearEnabled     bool
	watchDelay            int32
	readPassStdin         bool
	limitOutput           bool
	httpManagementURL     string
	httpManagementCluster string

	bFormat  bool
	kbFormat bool
	mbFormat bool
	gbFormat bool
	tbFormat bool

	// #nosec G404
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// various command constants
const (
	clusterConnectionDescription = "cluster connection name or cluster host:port (not required if context is set)"
	connectionNameOption         = "connection"
	clusterNameOptionShort       = "c"

	serviceNameDescription    = "Service name"
	serviceNameOption         = "service"
	serviceNameOptionShort    = "s"
	ignoreCachesDescription   = "ignore system caches or caches with $"
	includePercentDescription = "show percentage as bar"

	userNameDescription = "basic auth username if authentication is required"
	usernameOption      = "username"
	usernameShort       = "U"

	clusterKey            = "clusters"
	currentContextKey     = "currentContext"
	debugContextKey       = "debug"
	colorContextKey       = "color"
	defaultStyleKey       = "defaultStyle"
	useGradleContextKey   = "useGradle"
	ignoreCertsContextKey = "ignoreInvalidCerts"
	requestTimeoutKey     = "requestTimeout"
	defaultBytesFormatKey = "defaultBytesFormat"
	defaultLayoutKey      = "defaultLayout"
	defaultHeapKey        = "defaultHeap"
	profilesKey           = "profiles"
	panelsKey             = "panels"

	confirmOptionMessage     = "automatically confirm the operation"
	timeoutMessage           = "timeout in seconds for NS Lookup requests"
	heapMemoryMessage        = "heap memory to allocate for JVM if default-heap not set"
	startupDelayMessage      = "startup delay in millis for each server"
	heapMemoryArg            = "heap-memory"
	profileFirstArg          = "profile-first"
	profileArg               = "profile"
	startClassArg            = "start-class"
	logDestinationArg        = "log-destination"
	logLevelArg              = "log-level"
	startupDelayArg          = "startup-delay"
	serverCountMessage       = "number of replicas"
	dynamicHTTPMessage       = "enable dynamic http port for other members"
	metricsPortMessage       = "starting port for metrics"
	healthPortMessage        = "starting port for health"
	jmxPortMessage           = "JMX registry/RMI port for the initial management member (0 disables; saved after successful startup)"
	jmxHostMessage           = "advertised JMX host (defaults to the JMX bind address, independent of WKA)"
	machineMessage           = "the machine name to use"
	rackMessage              = "the rack name to use"
	siteMessage              = "the site name to use"
	roleMessage              = "the role name to use"
	cacheConfigMessage       = "cache configuration file"
	operationalConfigMessage = "override override file"
	cacheConfigArg           = "cache-config"
	operationalConfigArg     = "override-config"
	metricsPortArg           = "metrics-port"
	backupLogFilesArg        = "backup-logs"
	healthPortArg            = "health-port"
	jmxPortArg               = "jmx-port"
	jmxHostArg               = "jmx-host"
	machineArg               = "machine"
	rackArg                  = "rack"
	siteArg                  = "site"
	roleArg                  = "role"
	logLevelMessage          = "coherence log level"
	profileMessage           = "profile to add to cluster startup command line"
	backupLogFilesMessage    = "backup old cache server log files"
	startClassMessage        = "class name to start server with (default com.tangosol.net.Coherence)"
	profileFirstMessage      = "only apply profile to the first member starting"
	logDestinationMessage    = "root directory to place log files in"
	commaSeparatedIDMessage  = "comma separated node ids to target"

	outputFormats = "table, wide, json or jsonpath=\"...\""

	OperationCompleted = "operation completed"

	// config file related
	configName        = "cohctl"
	logsDirName       = "logs"
	configType        = "yaml"
	configDir         = ".cohctl"
	logFile           = "cohctl.log"
	winFileSinkScheme = "winfile"
	winFileSinkURL    = winFileSinkScheme + ":///" + logFile
)

var (
	// config file
	cfgFile string

	// configuration directory
	cfgDirectory string

	// logs directory
	logsDirectory string

	// Config is the CLI config
	Config CoherenceCLIConfig

	// Username contains the current username
	Username string

	// UsingContext indicates if we are using a context via the set context command
	UsingContext bool

	// path to the logfile
	logFilePath string

	// global sorting
	tableSorting string

	// global sort direction
	descendingFlag bool

	// global, include percentage as bar
	includePercentageBar bool

	// global percentage bar width
	percentageBarWidth int

	Logger *zap.Logger

	// Version is the cohctl version injected by the Go linker at build time.
	Version string
	// Commit is the git commit hash injected by the Go linker at build time.
	Commit string
	// Date is the build timestamp injected by the Go linker at build time.
	Date string
)

// CoherenceCLIConfig describes the details stored in the .cohctl.yaml.
// IgnoreInvalidCerts is retained only for legacy migration.
type CoherenceCLIConfig struct {
	Version            string              `json:"version"`
	CurrentContext     string              `json:"currentContext"`
	Clusters           []ClusterConnection `mapstructure:"clusters"`
	Debug              bool                `json:"debug"`
	Color              string              `json:"color"`
	RequestTimeout     int32               `json:"requestTimeout"`
	IgnoreInvalidCerts bool                `json:"ignoreInvalidCerts"`
	DefaultBytesFormat string              `json:"defaultBytesFormat"`
	DefaultHeap        string              `json:"defaultHeap"`
	UseGradle          bool                `json:"useGradle"`
	Profiles           []ProfileValue      `mapstructure:"profiles"`
	Panels             []Panel             `mapstructure:"panels"`
	DefaultStyle       string              `json:"defaultStyle"`
}

// ProfileValue describes a profile to be used for creating and starting clusters.
type ProfileValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Panel describes a panel and layout that can be used by the monitor cluster command.
type Panel struct {
	Name   string `json:"name"`
	Layout string `json:"layout"`
}

// ClusterConnection describes an individual connection to a cluster.
type ClusterConnection struct {
	Name                 string `json:"name"` // the name the user gives to the cluster connection
	DiscoveryType        string `json:"discoveryType"`
	ConnectionType       string `json:"connectionType"` // currently only valid value is "http"
	ConnectionURL        string `json:"url"`
	NameServiceDiscovery string `json:"nameServiceDiscovery"`
	ClusterVersion       string `json:"clusterVersionParam"`
	ClusterName          string `json:"clusterName"` // the actual cluster name
	ClusterType          string `json:"clusterType"`

	// IgnoreInvalidCerts records an optional connection-specific choice; nil stays absent in saved YAML.
	IgnoreInvalidCerts *bool `json:"ignoreInvalidCerts,omitempty" mapstructure:"ignoreinvalidcerts,omitempty" yaml:"ignoreinvalidcerts,omitempty"`

	// the following attributes are specific to manually created clusters
	ManuallyCreated     bool               `json:"manuallyCreated"`     // indicates if this was created by the create cluster command
	BaseClasspath       string             `json:"baseClasspath"`       // the minimum required classes coherence.jar and coherence-json
	AdditionalClasspath string             `json:"additionalClasspath"` // additional classpath provided by the user
	Arguments           string             `json:"arguments"`           // arguments to start cluster with including cluster name, etc
	ManagementPort      int32              `json:"managementPort"`      // arguments to start cluster with including cluster name, etc
	PersistenceMode     string             `json:"persistenceMode"`
	LoggingDestination  string             `json:"loggingDestination"` // logging destination, if empty then place under ~/.cohctl/logs
	ManagementAvailable bool               // only used when using -o wide option
	StartupClass        string             `json:"startupClass"`
	Management          *ClusterManagement `json:"management,omitempty" mapstructure:"management,omitempty" yaml:"management,omitempty"`
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = createRootCommand()

// createRootCommand creates the root command off which all others are places.
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

// SetRootCommandFlags sets all the persistent root command flags.
func SetRootCommandFlags(command *cobra.Command) {
	// Global flags for all commands
	command.PersistentFlags().StringVar(&cfgDirectory, "config-dir", "", "config directory (default is $HOME/.cohctl)")
	command.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cohctl/cohctl.yaml)")
	command.PersistentFlags().StringVarP(&OutputFormat, "output", "o", constants.TABLE, "output format: "+outputFormats)
	command.PersistentFlags().BoolVarP(&watchEnabled, "watch", "w", false, "watch output (only available for get commands)")
	command.PersistentFlags().BoolVarP(&watchClearEnabled, "watch-clear", "W", false, "watch output with clear")
	command.PersistentFlags().BoolVarP(&readPassStdin, "stdin", "i", false, "read password from stdin")
	command.PersistentFlags().BoolVarP(&limitOutput, "limit", "", false, "limit table output to screen size")
	command.PersistentFlags().Int32VarP(&watchDelay, "delay", "d", 5, "delay for watching in seconds")
	command.PersistentFlags().StringVarP(&Username, usernameOption, usernameShort, "", userNameDescription)
	command.PersistentFlags().StringVarP(&clusterConnection, connectionNameOption, clusterNameOptionShort, "", clusterConnectionDescription)
	command.PersistentFlags().StringVarP(&tableSorting, "sort", "", "", "specify a sorting column name or number for tables")
	command.PersistentFlags().BoolVarP(&descendingFlag, "desc", "", false, "indicates descending sort for tables, default is ascending")
	command.PersistentFlags().BoolVarP(&includePercentageBar, "percent-bar", "", false, includePercentDescription)
	command.PersistentFlags().IntVarP(&percentageBarWidth, "percent-bar-width", "", 30, "set percentage bar width")

	command.PersistentFlags().BoolVarP(&kbFormat, "kb", "k", false, "show sizes in kilobytes (default is bytes)")
	command.PersistentFlags().BoolVarP(&mbFormat, "mb", "m", false, "show sizes in megabytes (default is bytes)")
	command.PersistentFlags().BoolVarP(&gbFormat, "gb", "g", false, "show sizes in gigabytes (default is bytes)")
	command.PersistentFlags().BoolVarP(&tbFormat, "tb", "", false, "show sizes in terabytes (default is bytes)")
	command.PersistentFlags().BoolVarP(&bFormat, "bytes", "b", false, "show sizes in bytes")
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

	// if cfgDirectory not specified then check for env variable COHCTL_HOME to override default
	if cfgDirectory == "" {
		cfgDirectory = os.Getenv("COHCTL_HOME")
	}

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

	if err = viper.ReadInConfig(); err != nil {
		_, foundError := err.(viper.ConfigFileNotFoundError)

		if foundError || os.IsNotExist(err) {
			// Create only a missing file; never truncate an unreadable/racing one.
			file, createErr := os.OpenFile(cfgFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if createErr != nil {
				rootCmd.Println(utils.GetError("unable to create config file "+cfgFile, createErr))
				os.Exit(1)
			}
			if err = file.Close(); err != nil {
				rootCmd.Println(utils.GetError("unable to create config file "+cfgFile, err))
				os.Exit(1)
			}

			// config file not found - create a default one
			Config = CoherenceCLIConfig{Version: Version, Clusters: make([]ClusterConnection, 0),
				Debug: false, RequestTimeout: 30, IgnoreInvalidCerts: false, Color: "on", DefaultBytesFormat: "m"}
			viper.Set("version", Config.Version)
			viper.Set(currentContextKey, Config.CurrentContext)
			viper.Set(debugContextKey, Config.Debug)
			viper.Set(colorContextKey, Config.Color)
			viper.Set(ignoreCertsContextKey, Config.IgnoreInvalidCerts)
			viper.Set(requestTimeoutKey, Config.RequestTimeout)
			viper.Set(defaultBytesFormatKey, Config.DefaultBytesFormat)

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
			fmt.Fprintln(os.Stderr, utils.GetError("unable to read config "+cfgFile, err))
			os.Exit(1)
		}
	} else {
		// Migrate the selected file before color/version writes or command work.
		// The helper reads persisted values without AutomaticEnv overlays.
		var loaded CoherenceCLIConfig
		loaded, err = loadExistingConfig(viper.GetViper(), os.Stderr, defaultConfigMigrationOps())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		Config = loaded
	}

	if Config.Color == "" || Config.Color == "1" {
		// default to "on"
		if err = writeStartupConfigValue(viper.GetViper(), colorContextKey, on); err != nil {
			rootCmd.Println(err)
			os.Exit(1)
		}
		Config.Color = on
		viper.Set(colorContextKey, on)
	}

	// default defaultStyle
	if Config.DefaultStyle == "" {
		Config.DefaultStyle = "default"
	}

	// setup logs directory
	logsDirectory = filepath.Join(cfgDirectory, logsDirName)
	err = ensureLogsDir()
	if err != nil {
		fmt.Printf("unable to create logs directory: %s, %v", logsDirectory, err)
		os.Exit(1)
	}

	Logger.Info("Configuration loaded", []zapcore.Field{
		zap.String("configFile", cfgFile),
		zap.String("configDir", cfgDirectory),
		zap.String("logsDir", logsDirectory),
	}...)

	// check for newer version of cohctl and carry out any upgrade tasks
	if Config.Version != Version {
		Logger.Info("Upgrading cohctl version", []zapcore.Field{
			zap.String("oldVersion", Config.Version),
			zap.String("newVersion", Version)}...)

		// carry out any upgrade logic here
		err = writeStartupConfigValue(viper.GetViper(), "version", Version)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		Config.Version = Version
		viper.Set("version", Config.Version)
	}

	fetcher.DebugEnabled = Config.Debug
	utils.DebugEnabled = Config.Debug
	fetcher.RequestTimeout = Config.RequestTimeout
}

// GetClusterConnection returns the URL for a given a cluster connection name.
func GetClusterConnection(connectionName string) (bool, ClusterConnection) {
	for _, cluster := range Config.Clusters {
		if cluster.Name == connectionName {
			return true, cluster
		}
	}
	return false, ClusterConnection{}
}

// configurationClusters uses configuration decoder names, independently of the
// public JSON output tags on ClusterConnection. Decode maps directly too, so
// effective settings loaded from a file retain fields unknown to this CLI.
func configurationClusters(clusters any) ([]map[string]any, error) {
	var settings []map[string]any
	if err := mapstructure.Decode(clusters, &settings); err != nil {
		return nil, err
	}
	return settings, nil
}

type configurationEncoderRegistry struct {
	viper.EncoderRegistry
	clusters []map[string]any
}

func (r configurationEncoderRegistry) Encoder(format string) (viper.Encoder, error) {
	encoder, err := r.EncoderRegistry.Encoder(format)
	if err != nil {
		return nil, err
	}
	return configurationEncoder{encoder, r.clusters}, nil
}

type configurationEncoder struct {
	viper.Encoder
	clusters []map[string]any
}

func (e configurationEncoder) Encode(settings map[string]any) ([]byte, error) {
	// Viper supplies a fresh AllSettings map to the encoder. Never normalize
	// its live override/config maps or replace them with the stale Config.
	if _, present := settings[clusterKey]; present {
		settings[clusterKey] = e.clusters
	}
	return e.Encoder.Encode(settings)
}

// WriteConfig writes effective Viper settings using configuration field names.
func WriteConfig() error {
	clusters, err := configurationClusters(viper.Get(clusterKey))
	if err != nil {
		return fmt.Errorf("unable to write config: encode cluster settings: %w", err)
	}
	// Change only the encoder option for this call. A shallow value snapshot
	// restores that option without mutating settings, bindings or defaults.
	// Keep Viper's ordinary filename/codec/filesystem/write semantics intact.
	current := viper.GetViper()
	previous := *current
	defer func() {
		selectedFile := current.ConfigFileUsed()
		*current = previous
		// A successful first write can discover the new default file. Retain
		// that selection for a managed save in the same CLI invocation.
		if err == nil && selectedFile != "" {
			current.SetConfigFile(selectedFile)
		}
	}()
	viper.SetOptions(viper.WithEncoderRegistry(configurationEncoderRegistry{viper.NewCodecRegistry(), clusters}))
	if err = viper.WriteConfig(); err != nil {
		return fmt.Errorf("unable to write config: %w", err)
	}
	return nil
}

// GetConfigDir returns the configuration directory.
func GetConfigDir() string {
	return cfgDirectory
}

func GetLogDirectory() string {
	return logsDirectory
}

func ensureLogsDir() error {
	return utils.EnsureDirectory(GetLogDirectory())
}

// Initialize initializes the command hierarchy - required for tests
// if command is nil then a new command is created otherwise the existing
// one is used.
func Initialize(command *cobra.Command) *cobra.Command {
	viper.Reset()
	cfgDirectory = ""
	cfgFile = ""
	Config.CurrentContext = ""
	dumpRoleName = all
	configureRole = ""
	threadDumpRole = all

	Config.Clusters = make([]ClusterConnection, 0)

	if command == nil {
		command = createRootCommand()
	}

	// Reset at execution, not construction: tests and embedded callers may run
	// the same Cobra command repeatedly in one process.
	command.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		fetcher.BeginCommandTLSWarnings(cmd.ErrOrStderr())
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
	getCmd.AddCommand(getProxyMembersCmd)
	getCmd.AddCommand(getHTTPProxiesCmd)
	getCmd.AddCommand(getHTTPServerMembersCmd)
	getCmd.AddCommand(getReportersCmd)
	getCmd.AddCommand(getDebugCmd)
	getCmd.AddCommand(getTimeoutCmd)
	getCmd.AddCommand(getLogsCmd)
	getCmd.AddCommand(getElasticDataCmd)
	getCmd.AddCommand(getSnapshotsCmd)
	getCmd.AddCommand(getHTTPSessionsCmd)
	getCmd.AddCommand(getTopicsCmd)
	getCmd.AddCommand(getTopicMembersCmd)
	getCmd.AddCommand(getSubscribersCmd)
	getCmd.AddCommand(getTopicChannelsCmd)
	getCmd.AddCommand(getSubscriberChannelsCmd)
	getCmd.AddCommand(getSubscriberGroupsCmd)
	getCmd.AddCommand(getSubscriberGroupChannelsCmd)
	getCmd.AddCommand(getJfrsCmd)
	getCmd.AddCommand(getIgnoreCertsCmd)
	getCmd.AddCommand(getExecutorsCmd)
	getCmd.AddCommand(getManagementCmd)
	getCmd.AddCommand(getFederationCmd)
	getCmd.AddCommand(getTracingCmd)
	getCmd.AddCommand(getBytesFormatCmd)
	getCmd.AddCommand(getHealthCmd)
	getCmd.AddCommand(getEnvironmentCmd)
	getCmd.AddCommand(getServiceMembersCmd)
	getCmd.AddCommand(getDefaultHeapCmd)
	getCmd.AddCommand(getProfilesCmd)
	getCmd.AddCommand(getProxyConnectionsCmd)
	getCmd.AddCommand(getUseGradleCmd)
	getCmd.AddCommand(getServiceStorageCmd)
	getCmd.AddCommand(getServiceOwnershipCmd)
	getCmd.AddCommand(getCacheStoresCmd)
	getCmd.AddCommand(getColorCmd)
	getCmd.AddCommand(getNetworkStatsCmd)
	getCmd.AddCommand(getP2PStatsCmd)
	getCmd.AddCommand(getConfigCmd)
	getCmd.AddCommand(getServiceDistributionsCmd)
	getCmd.AddCommand(getClusterConfigCmd)
	getCmd.AddCommand(getServiceDescriptionCmd)
	getCmd.AddCommand(getClusterDescription)
	getCmd.AddCommand(getMemberDescriptionCmd)
	getCmd.AddCommand(getCacheAccessCmd)
	getCmd.AddCommand(getCacheStorageCmd)
	getCmd.AddCommand(getCacheIndexesCmd)
	getCmd.AddCommand(getViewCachesCmd)
	getCmd.AddCommand(getCachePartitionsCmd)
	getCmd.AddCommand(getFederationIncomingCmd)
	getCmd.AddCommand(getFederationOutgoingCmd)
	getCmd.AddCommand(getPanelsCmd)
	getCmd.AddCommand(getDefaultStyleCmd)

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
	setCmd.AddCommand(setBytesFormatCmd)
	setCmd.AddCommand(setExecutorCmd)
	setCmd.AddCommand(setDefaultHeapCmd)
	setCmd.AddCommand(setProfileCmd)
	setCmd.AddCommand(setUseGradleCmd)
	setCmd.AddCommand(setFederationCmd)
	setCmd.AddCommand(setColorCmd)
	setCmd.AddCommand(setClusterCmd)
	setCmd.AddCommand(setDefaultStyleCmd)

	// run command
	command.AddCommand(runCmd)
	runCmd.AddCommand(runReportCmd)

	// restart command
	command.AddCommand(restartCmd)
	restartCmd.AddCommand(restartClusterCmd)

	// clear
	command.AddCommand(clearCmd)
	clearCmd.AddCommand(clearContextCmd)
	clearCmd.AddCommand(clearBytesFormatCmd)
	clearCmd.AddCommand(clearDefaultHeapCmd)
	clearCmd.AddCommand(clearCacheCmd)

	// truncate
	command.AddCommand(truncateCmd)
	truncateCmd.AddCommand(truncateCacheCmd)

	// add
	command.AddCommand(addCmd)
	addCmd.AddCommand(addClusterCmd)
	addCmd.AddCommand(addPanelCmd)

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
	startCmd.AddCommand(startClusterCmd)
	startCmd.AddCommand(startConsoleCmd)
	startCmd.AddCommand(startCohQLCmd)
	startCmd.AddCommand(startClassCmd)

	// stop
	command.AddCommand(stopCmd)
	stopCmd.AddCommand(stopReporterCmd)
	stopCmd.AddCommand(stopJfrCmd)
	stopCmd.AddCommand(stopFederationCmd)
	stopCmd.AddCommand(stopServiceCmd)
	stopCmd.AddCommand(stopClusterCmd)

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
	removeCmd.AddCommand(removeProfileCmd)
	removeCmd.AddCommand(removePanelCmd)

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
	describeCmd.AddCommand(describeFederationCmd)
	describeCmd.AddCommand(describeTopicCmd)
	describeCmd.AddCommand(describeViewCacheCmd)

	// create
	command.AddCommand(createCmd)
	createCmd.AddCommand(createSnapshotCmd)
	createCmd.AddCommand(createClusterCmd)
	createCmd.AddCommand(createStarterCmd)

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
	retrieveCmd.AddCommand(retrieveHeadsCmd)
	retrieveCmd.AddCommand(retrieveRemainingCmd)

	// discover
	command.AddCommand(discoverCmd)
	discoverCmd.AddCommand(discoverClustersCmd)

	// disconnect
	command.AddCommand(disconnectCmd)
	disconnectCmd.AddCommand(disconnectSubscriberCmd)
	disconnectCmd.AddCommand(disconnectAllCmd)

	// connect
	command.AddCommand(connectCmd)
	connectCmd.AddCommand(connectSubscriberCmd)

	// notify
	command.AddCommand(notifyCmd)
	notifyCmd.AddCommand(notifyPopulatedCmd)

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

	// scale
	command.AddCommand(scaleCmd)
	scaleCmd.AddCommand(scaleClusterCmd)

	// reset-statistics
	command.AddCommand(resetCmd)
	resetCmd.AddCommand(resetMemberStatsCmd)
	resetCmd.AddCommand(resetReporterStatsCmd)
	resetCmd.AddCommand(resetRAMJournalStatsCmd)
	resetCmd.AddCommand(resetFlashJournalStatsCmd)
	resetCmd.AddCommand(resetServiceStatsCmd)
	resetCmd.AddCommand(resetCacheStatsCmd)
	resetCmd.AddCommand(resetFederationStatsCmd)
	resetCmd.AddCommand(resetExecutorStatsCmd)
	resetCmd.AddCommand(resetProxyStatsCmd)

	// compact
	command.AddCommand(compactCmd)
	compactCmd.AddCommand(compactElasticDataCmd)

	// monitor
	command.AddCommand(monitorCmd)
	monitorCmd.AddCommand(monitorHealthCmd)
	monitorCmd.AddCommand(monitorClusterCmd)

	// force
	command.AddCommand(forceCmd)
	forceCmd.AddCommand(forceRecoveryCmd)

	// monitoring
	command.AddCommand(initCmd)
	initCmd.AddCommand(initMonitoringCmd)
	getCmd.AddCommand(getMonitoringCmd)
	startCmd.AddCommand(startMonitoringCmd)
	stopCmd.AddCommand(stopMonitoringCmd)

	return command
}

// GetDataFetcher returns a Fetcher given a cluster name.
func GetDataFetcher(clusterName string) (fetcher.Fetcher, error) {
	return getDataFetcher(clusterName, getDiscoveredClusters)
}

// Pass discovery explicitly so the selection policy can be exercised without a
// running Name Service or mutable package-level test hooks.
func getDataFetcher(clusterName string, discoverClusters func(*cobra.Command, []string) ([]discovery.DiscoveredCluster, []int, error)) (fetcher.Fetcher, error) {
	var (
		finalClusterName, finalConnectionURL, finalConnectionType string
		connectionName                                            string
		ignoreInvalidCerts                                        bool
	)

	// check to see if we have a ':' in the cluster, then we assume this is a host:port of
	// the cluster, and we look up the http url dynamically
	if strings.Contains(clusterName, ":") {
		discoveredClusters, _, err := discoverClusters(nil, []string{clusterName})
		if err != nil {
			return nil, err
		}
		if len(discoveredClusters) == 0 {
			return nil, fmt.Errorf("no clusters on host/port %s", clusterName)
		}
		if len(discoveredClusters) > 1 {
			return nil, fmt.Errorf("multiple clusters with HTTP management enabled found on host/port %s", clusterName)
		}

		// we have exactly one cluster found, so check if management over REST enabled.
		managementURLs := discoveredClusters[0].ManagementURLs
		discoveredClusterName := discoveredClusters[0].ClusterName
		if len(managementURLs) == 0 {
			return nil, fmt.Errorf("cluster %s does not have management over REST enabled", discoveredClusterName)
		}

		// extract one of the URLS randomly
		finalConnectionURL = randomize(managementURLs)
		finalClusterName = discoveredClusterName
		finalConnectionType = httpType
		httpManagementURL = finalConnectionURL
		httpManagementCluster = finalClusterName
	} else {
		// normal path, retrieve from config
		found, connection := GetClusterConnection(clusterName)
		if !found {
			return nil, errors.New(UnableToFindClusterMsg + clusterName)
		}
		finalClusterName = connection.ClusterName
		finalConnectionType = connection.ConnectionType
		finalConnectionURL = connection.ConnectionURL
		connectionName = connection.Name
		ignoreInvalidCerts = connection.IgnoreInvalidCerts != nil && *connection.IgnoreInvalidCerts
		httpManagementURL = ""
		httpManagementCluster = ""
	}

	return fetcher.GetFetcherForConnection(finalConnectionType, finalConnectionURL, Username, finalClusterName, connectionName, ignoreInvalidCerts)
}

func randomize(arr []string) string {
	if len(arr) == 1 {
		return arr[0]
	}
	if len(arr) == 0 {
		return ""
	}
	return arr[rnd.Intn(len(arr))]
}

// GetConnectionNameFromContextOrArg returns the connection name from the '-c' option
// or the current context if set.
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
	return "", errors.New("you must provide a connection name if you have not set the current context")
}

// GetConnectionAndDataFetcher returns the connection and dataFetcher.
func GetConnectionAndDataFetcher() (string, fetcher.Fetcher, error) {
	var (
		err          error
		connection   string
		dataFetcher  fetcher.Fetcher
		optionsCount = 0
	)

	// do validation of bytes format
	if kbFormat {
		optionsCount++
	}

	if mbFormat {
		optionsCount++
	}

	if gbFormat {
		optionsCount++
	}

	if optionsCount > 1 {
		return "", nil, errors.New("you can only supply one size format of -k, -m or -g")
	}

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

// checkOutputFormat checks for valid output formats.
func checkOutputFormat() error {
	if OutputFormat != constants.TABLE && OutputFormat != constants.JSON && OutputFormat != constants.WIDE &&
		!strings.Contains(OutputFormat, constants.JSONPATH) {
		return fmt.Errorf("you must specify one of the following output formats: " + outputFormats)
	}
	return nil
}

// newWinFileSink binds the Windows sink to the selected directory, not a URL
// containing its native path. Relative selections are resolved once so later
// working-directory changes cannot redirect the factory.
func newWinFileSink(directory string) (func(*url.URL) (zap.Sink, error), error) {
	if directory == "" {
		return nil, errors.New("missing CLI log directory")
	}
	rootDirectory, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	return func(u *url.URL) (zap.Sink, error) {
		if u == nil || u.Scheme != winFileSinkScheme || u.Path != "/"+logFile ||
			u.Host != "" || u.User != nil || u.Opaque != "" || u.OmitHost ||
			u.RawPath != "" || u.ForceQuery || u.RawQuery != "" || u.Fragment != "" || u.RawFragment != "" {
			return nil, errors.New("invalid Windows log sink URL")
		}
		root, err := os.OpenRoot(rootDirectory)
		if err != nil {
			return nil, err
		}
		file, openErr := root.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		closeErr := root.Close()
		if openErr != nil {
			return nil, errors.Join(openErr, closeErr)
		}
		if closeErr != nil {
			return nil, errors.Join(closeErr, file.Close())
		}
		return file, nil
	}, nil
}

// initLogging initializes the logging.
func initLogging(homeDir string) (*zap.Logger, error) {
	cfg := zap.NewDevelopmentConfig()

	logFilePath = filepath.Join(homeDir, logFile)
	if isWindows() {
		// workaround for logging issue on Windows
		factory, err := newWinFileSink(homeDir)
		if err != nil {
			return nil, err
		}
		err = zap.RegisterSink(winFileSinkScheme, factory)
		if err != nil {
			return nil, fmt.Errorf("unable to register sink %v", err)
		}
		cfg.OutputPaths = []string{winFileSinkURL}
	} else {
		cfg.OutputPaths = []string{logFilePath}
	}

	return cfg.Build()
}

// isWindows returns true if the OS is Windows.
func isWindows() bool {
	return runtime.GOOS == "windows"
}

// confirmOperation displays a confirmation message and will return true if
// the operation is confirmed to continue via either the -y option or the
// user answers "y".
func confirmOperation(cmd *cobra.Command, message string) bool {
	var (
		response string
		err      error
	)

	if automaticallyConfirm {
		return true
	}

	cmd.Print(message)
	_, err = fmt.Fscanln(cmd.InOrStdin(), &response)
	if response != "y" || err != nil {
		cmd.Println(constants.NoOperation)
		return false
	}
	return true
}

// processJSONOutput processes JSON output and either outputs the JSONPath or JSON results.
func processJSONOutput(cmd *cobra.Command, jsonData []byte) error {
	if strings.Contains(OutputFormat, constants.JSONPATH) {
		result, err := utils.GetJSONPathResults(jsonData, OutputFormat)
		if err != nil {
			return err
		}
		cmd.Println(result)
		return nil
	}
	cmd.Println(string(jsonData))
	return nil
}

// isJSONPathOrJSON returns true of the output is JSONPath or JSON.
func isJSONPathOrJSON() bool {
	return strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON
}
