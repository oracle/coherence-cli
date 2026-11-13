/*
 * Copyright (c) 2025, 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	monitoringDirectory     = "monitoring"
	dashboardsDirectory     = "dashboards"
	dashboardBaseURL        = "https://raw.githubusercontent.com/oracle/coherence-operator/refs/heads/main/dashboards/grafana"
	configRepositoryURL     = "https://raw.githubusercontent.com/oracle/coherence-cli"
	grafanaPort             = 3000
	prometheusPort          = 9090
	dockerComposeYAML       = "docker-compose.yaml"
	useDockerComposeMessage = "use the standalone docker-compose executable instead of docker compose"
	dockerCompose           = "docker-compose"
)

var (
	dashboardFiles = [...]string{
		"cache-details-dashboard.json",
		"cache-store-details-dashboard.json",
		"caches-summary-dashboard.json",
		"coherence-dashboard-main.json",
		"elastic-data-summary-dashboard.json",
		"executor-details.json",
		"executors-summary.json",
		"federation-details-dashboard.json",
		"federation-summary-dashboard.json",
		"grpc-proxy-details-dashboard.json",
		"grpc-proxy-summary-dashboard.json",
		"http-servers-summary-dashboard.json",
		"machines-summary-dashboard.json",
		"member-details-dashboard.json",
		"members-summary-dashboard.json",
		"persistence-summary-dashboard.json",
		"proxy-server-detail-dashboard.json",
		"proxy-servers-summary-dashboard.json",
		"service-details-dashboard.json",
		"services-summary-dashboard.json",
		"topic-details-dashboard.json",
		"topic-subscriber-details.json",
		"topic-subscriber-group-details.json",
		"topics-summary-dashboard.json",
	}

	dockerComposeFiles = [...]string{
		grafanaINIFile,
		"dashboards.yaml",
		"datasources.yaml",
		dockerComposeYAML,
		"prometheus.yaml",
	}

	useDockerComposeParam bool
	openHomePageParam     bool

	monitoringReleaseVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?(?:\+[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$`)
)

var initMonitoringCmd = &cobra.Command{
	Use:   "monitoring",
	Short: "initialize local monitoring for Coherence",
	Long: `The 'init monitoring' command initializes local monitoring for Coherence.
It downloads the Grafana dashboards and monitoring configuration, creates a
	generated secure compose bundle, and pulls the Grafana and Prometheus images.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		return initializeMonitoring(cmd, GetURLContents)
	},
}

// initializeMonitoring shares the complete initialization path with the checkout
// integration harness. Production always supplies GetURLContents; the reader is
// an explicit dependency, never a CLI/environment source override.
func initializeMonitoring(cmd *cobra.Command, readURL func(string) ([]byte, error)) error {
	if _, err := monitoringConfigBaseURL(Version); err != nil {
		return err
	}
	bindAddress, err := monitoringBindAddressFromEnvironment()
	if err != nil {
		return err
	}
	cmd.Println("This command will:")
	cmd.Printf("1. Create or replace %s\n", filepath.Join(cfgDirectory, monitoringDirectory))
	cmd.Println("2. Download Grafana dashboards and monitoring configuration")
	cmd.Println("3. Generate a protected Grafana administrator password")
	cmd.Println("4. Pull Grafana and Prometheus images")
	cmd.Println()
	if !confirmOperation(cmd, "Are you sure you want to initialize monitoring? (y/n) ") {
		return nil
	}

	mon := newMonitoring(cmd)
	if err = mon.requireComposeV2(); err != nil {
		return err
	}
	if err = mon.ensureMonitoringIsStopped(); err != nil {
		return err
	}
	if err = ensureDirectory(cfgDirectory); err != nil {
		return err
	}
	stagedDirectory, err := os.MkdirTemp(cfgDirectory, ".monitoring-staging-")
	if err != nil {
		return fmt.Errorf("unable to create monitoring staging directory: %w", err)
	}
	installed := false
	defer func() {
		if !installed {
			_ = os.RemoveAll(stagedDirectory)
		}
	}()
	staged := newMonitoringAt(cmd, stagedDirectory)

	cmd.Println("Staging Grafana dashboards...")
	if err = staged.ensureDirectories(); err != nil {
		return err
	}
	if err = staged.downloadDashboards(readURL); err != nil {
		return err
	}
	cmd.Println("Staging monitoring configuration...")
	if err = staged.downloadDockerComposeFiles(readURL); err != nil {
		return err
	}
	password, err := staged.createGeneratedBundle(bindAddress)
	if err != nil {
		return err
	}
	staged.composeFile = filepath.Join(staged.monitoringDir, generatedDockerComposeYAML)
	if err = staged.discoverImages(); err != nil {
		return err
	}

	cmd.Println("Pulling docker images...")
	if err = staged.dockerCommand([]string{"pull", staged.prometheusImage}); err != nil {
		return err
	}
	if err = staged.dockerCommand([]string{"pull", staged.grafanaImage}); err != nil {
		return err
	}
	if err = mon.installStagedDirectory(stagedDirectory); err != nil {
		return err
	}
	installed = true

	cmd.Println(OperationCompleted)
	if warning := monitoringExposureWarning(bindAddress); warning != "" {
		cmd.Println(warning)
	}
	cmd.Println()
	cmd.Println("Grafana administrator user: admin")
	cmd.Printf("Grafana administrator password: %s\n", password)
	cmd.Printf("The password is stored in %s and will not be displayed again.\n", filepath.Join(mon.monitoringDir, grafanaPasswordFile))
	cmd.Printf("Generated compose configuration: %s\n", filepath.Join(mon.monitoringDir, generatedDockerComposeYAML))
	return nil
}

type grafanaStatus struct {
	Database string `json:"database"`
	Version  string `json:"version"`
}

var getMonitoringCmd = &cobra.Command{
	Use:   "monitoring",
	Short: "return monitoring status",
	Long:  "The 'get monitoring' command gets monitoring status and validates the local monitoring environment.",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		mon := newMonitoring(cmd)
		if err := mon.validateEnvironment(); err != nil {
			return err
		}
		promURL := monitoringURL(mon.bindAddress, prometheusPort, "/")
		promHealthURL := monitoringURL(mon.bindAddress, prometheusPort, "/-/healthy")
		grafanaURL := monitoringURL(mon.bindAddress, grafanaPort, "/d/coh-main/coherence-dashboard-main")
		grafanaHealthURL := monitoringURL(mon.bindAddress, grafanaPort, "/api/health")

		promOutput := ""
		if content, err := GetURLContents(promHealthURL); err != nil {
			promOutput = err.Error()
		} else {
			promOutput = string(content)
		}
		grafanaOutput := ""
		if content, err := GetURLContents(grafanaHealthURL); err != nil {
			grafanaOutput = err.Error()
		} else {
			var status grafanaStatus
			if err = json.Unmarshal(content, &status); err != nil {
				grafanaOutput = err.Error()
			} else {
				grafanaOutput = fmt.Sprintf("%s, version=%s", status.Database, status.Version)
			}
		}

		cmd.Println("Monitoring status")
		cmd.Printf("Grafana:    %s\n", grafanaURL)
		cmd.Printf("  Image:    %s\n", mon.grafanaImage)
		cmd.Printf("  Status:   %s\n", grafanaOutput)
		cmd.Printf("Prometheus: %s\n", promURL)
		cmd.Printf("  Image:    %s\n", mon.prometheusImage)
		cmd.Printf("  Status:   %s\n", promOutput)
		cmd.Printf("Compose:    %s\n", mon.composeFile)
		cmd.Println("Docker")
		return mon.dockerCommand([]string{"ps"})
	},
}

var startMonitoringCmd = &cobra.Command{
	Use:   "monitoring",
	Short: "start the local monitoring stack",
	Long:  "The 'start monitoring' command starts the local Grafana and Prometheus stack using docker compose.",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		mon := newMonitoring(cmd)
		if err := mon.validateEnvironment(); err != nil {
			return err
		}
		if mon.bundleMode == monitoringBundleGenerated {
			if err := mon.requireComposeV2(); err != nil {
				return err
			}
		}
		if err := mon.runCompose(mon.composeFile, mon.bindAddress, []string{"up", "-d"}); err != nil {
			return err
		}

		grafanaURL := monitoringURL(mon.bindAddress, grafanaPort, "/d/coh-main/coherence-dashboard-main")
		cmd.Printf("\nOpen the Grafana dashboard at %s using administrator user admin.\n", grafanaURL)
		if mon.bundleMode == monitoringBundleGenerated {
			cmd.Printf("Read the password from %s.\n\n", filepath.Join(mon.monitoringDir, grafanaPasswordFile))
		} else {
			cmd.Println("Credentials are defined by the legacy grafana.ini configuration.")
			cmd.Println()
		}
		cmd.Println("Note: For Coherence versions before 14.1.2-0-0 or 25.03, set this property on each cluster member:")
		cmd.Println("  -Dcoherence.metrics.legacy.names=false")
		cmd.Println()
		if err := mon.dockerCommand([]string{"ps"}); err != nil {
			return err
		}
		if !openHomePageParam {
			return nil
		}
		time.Sleep(2 * time.Second)
		if runtime.GOOS == "darwin" {
			return exec.Command("open", grafanaURL).Run()
		}
		if isWindows() {
			return exec.Command("rundll32", "url.dll,FileProtocolHandler", grafanaURL).Run()
		}
		return exec.Command("xdg-open", grafanaURL).Run()
	},
}

var stopMonitoringCmd = &cobra.Command{
	Use:   "monitoring",
	Short: "stop the local monitoring stack",
	Long:  "The 'stop monitoring' command stops the local Grafana and Prometheus stack using docker compose.",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		mon := newMonitoring(cmd)
		if !confirmOperation(cmd, "Are you sure you want to stop monitoring? (y/n) ") {
			return nil
		}
		composeFile, bindAddress, generated, err := mon.composeForStop()
		if err != nil {
			return err
		}
		if generated {
			if err = mon.requireComposeV2(); err != nil {
				return err
			}
		}
		if err = mon.runCompose(composeFile, bindAddress, []string{"down"}); err != nil {
			return err
		}
		return mon.dockerCommand([]string{"ps"})
	},
}

type monitoring struct {
	monitoringDir   string
	dashboardsDir   string
	composeFile     string
	bindAddress     string
	bundleMode      monitoringBundleMode
	cmd             *cobra.Command
	grafanaImage    string
	prometheusImage string
}

func newMonitoring(cmd *cobra.Command) *monitoring {
	return newMonitoringAt(cmd, filepath.Join(cfgDirectory, monitoringDirectory))
}

func newMonitoringAt(cmd *cobra.Command, directory string) *monitoring {
	return &monitoring{
		cmd:           cmd,
		monitoringDir: directory,
		dashboardsDir: filepath.Join(directory, dashboardsDirectory),
		bindAddress:   defaultMonitoringAddress,
	}
}

func (m *monitoring) String() string {
	return fmt.Sprintf("monitoringDir=%s, dashboardsDir=%s", m.monitoringDir, m.dashboardsDir)
}

func (m *monitoring) ensureDirectories() error {
	if err := ensureDirectory(m.monitoringDir); err != nil {
		return err
	}
	return ensureDirectory(m.dashboardsDir)
}

func (m *monitoring) validateEnvironment() error {
	if !utils.DirectoryExists(m.monitoringDir) {
		return monitoringNotValid(m.monitoringDir + " does not exist")
	}
	if !utils.DirectoryExists(m.dashboardsDir) {
		return monitoringNotValid(m.dashboardsDir + " does not exist")
	}
	for _, file := range dashboardFiles {
		if destination := filepath.Join(m.dashboardsDir, file); !utils.FileExists(destination) {
			return monitoringNotValid(destination + " does not exist, or is not a file")
		}
	}
	for _, file := range dockerComposeFiles {
		if file == dockerComposeYAML {
			continue
		}
		if destination := filepath.Join(m.monitoringDir, file); !utils.FileExists(destination) {
			return monitoringNotValid(destination + " does not exist, or is not a file")
		}
	}

	mode, err := classifyMonitoringBundle(m.monitoringDir)
	if err != nil {
		return err
	}
	m.bundleMode = mode
	if mode == monitoringBundleGenerated {
		m.composeFile = filepath.Join(m.monitoringDir, generatedDockerComposeYAML)
		m.bindAddress, err = readMonitoringEnv(m.monitoringDir)
		if err != nil {
			return monitoringNotValid(err.Error())
		}
		if err = validateMonitoringSecretFile(filepath.Join(m.monitoringDir, grafanaPasswordFile)); err != nil {
			return monitoringNotValid(err.Error())
		}
		if err = validateGeneratedComposeFile(m.composeFile); err != nil {
			return monitoringNotValid(err.Error())
		}
		if warning := monitoringExposureWarning(m.bindAddress); warning != "" {
			m.cmd.Println(warning)
		}
	} else {
		m.composeFile = filepath.Join(m.monitoringDir, dockerComposeYAML)
		m.warnLegacyConfiguration()
	}
	return m.discoverImages()
}

func (m *monitoring) createGeneratedBundle(bindAddress string) (string, error) {
	composePath := filepath.Join(m.monitoringDir, dockerComposeYAML)
	composeContents, err := os.ReadFile(composePath)
	if err != nil {
		return "", fmt.Errorf("unable to read staged compose configuration: %w", err)
	}
	generated, err := generateSecureCompose(composeContents)
	if err != nil {
		return "", err
	}
	iniPath := filepath.Join(m.monitoringDir, grafanaINIFile)
	iniContents, err := os.ReadFile(iniPath)
	if err != nil {
		return "", fmt.Errorf("unable to read staged Grafana configuration: %w", err)
	}
	// #nosec G306 -- staged Grafana configuration is intentionally readable by Docker.
	if err = os.WriteFile(iniPath, sanitizeGrafanaINI(iniContents), 0644); err != nil {
		return "", fmt.Errorf("unable to sanitize staged Grafana configuration: %w", err)
	}
	if err = writeFileContents(m.monitoringDir, generatedDockerComposeYAML, generated); err != nil {
		return "", err
	}
	if err = writeMonitoringEnv(m.monitoringDir, bindAddress); err != nil {
		return "", err
	}
	password, err := generateGrafanaPassword()
	if err != nil {
		return "", err
	}
	if err = writeGrafanaPassword(m.monitoringDir, password); err != nil {
		return "", err
	}
	return password, nil
}

func (m *monitoring) installStagedDirectory(stagedDirectory string) error {
	info, err := os.Lstat(m.monitoringDir)
	if errors.Is(err, os.ErrNotExist) {
		if err = os.Rename(stagedDirectory, m.monitoringDir); err != nil {
			return fmt.Errorf("unable to install monitoring configuration: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("unable to inspect existing monitoring directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("refusing to replace %s because it is not a regular directory", m.monitoringDir)
	}
	backup := fmt.Sprintf("%s.backup-%d", m.monitoringDir, time.Now().UnixNano())
	if err = os.Rename(m.monitoringDir, backup); err != nil {
		return fmt.Errorf("unable to preserve existing monitoring configuration: %w", err)
	}
	if err = os.Rename(stagedDirectory, m.monitoringDir); err != nil {
		if rollbackErr := os.Rename(backup, m.monitoringDir); rollbackErr != nil {
			return fmt.Errorf("unable to install monitoring configuration: %v; rollback also failed: %w", err, rollbackErr)
		}
		return fmt.Errorf("unable to install monitoring configuration; previous configuration was restored: %w", err)
	}
	if err = os.RemoveAll(backup); err != nil {
		m.cmd.Printf("WARNING: unable to remove monitoring backup %s: %v\n", backup, err)
	}
	return nil
}

// Grafana applies its initial administrator password only to a fresh database.
// Even a stopped container can reuse the old database after secret replacement.
func (m *monitoring) ensureMonitoringIsStopped() error {
	generated := filepath.Join(m.monitoringDir, generatedDockerComposeYAML)
	legacy := filepath.Join(m.monitoringDir, dockerComposeYAML)
	composeFile := ""
	bindAddress := defaultMonitoringAddress
	if utils.FileExists(generated) {
		if err := validateComposeServicesFile(generated); err == nil {
			composeFile = generated
			if value, readErr := readMonitoringEnv(m.monitoringDir); readErr == nil {
				bindAddress = value
			}
		}
	}
	if composeFile == "" && utils.FileExists(legacy) {
		if err := validateComposeServicesFile(legacy); err == nil {
			composeFile = legacy
		}
	}
	if composeFile == "" {
		// Containers can outlive a missing or unusable configuration directory.
		// Match Compose's project override, otherwise the monitoring directory name.
		project := os.Getenv("COMPOSE_PROJECT_NAME")
		if project == "" {
			project = monitoringDirectory
		}
		output, err := executeHostCommandOutput(nil, "docker", "ps", "--all", "-q", "--filter", "label=com.docker.compose.project="+project)
		if err != nil {
			return fmt.Errorf("unable to determine whether monitoring containers remain; remove them with 'cohctl stop monitoring' before reinitializing: %w", err)
		}
		if strings.TrimSpace(output) != "" {
			return errors.New("monitoring containers still exist (including stopped containers); run 'cohctl stop monitoring' to remove them before reinitializing")
		}
		return nil
	}
	output, err := m.composeOutput(composeFile, bindAddress, []string{"ps", "--all", "-q"})
	if err != nil {
		return fmt.Errorf("unable to determine whether monitoring containers remain; remove them with 'cohctl stop monitoring' before reinitializing: %w", err)
	}
	if strings.TrimSpace(output) != "" {
		return errors.New("monitoring containers still exist (including stopped containers); run 'cohctl stop monitoring' to remove them before reinitializing")
	}
	return nil
}

func (m *monitoring) composeForStop() (string, string, bool, error) {
	generated := filepath.Join(m.monitoringDir, generatedDockerComposeYAML)
	legacy := filepath.Join(m.monitoringDir, dockerComposeYAML)
	if utils.FileExists(generated) {
		if err := validateComposeServicesFile(generated); err == nil {
			bindAddress := defaultMonitoringAddress
			if value, readErr := readMonitoringEnv(m.monitoringDir); readErr == nil {
				bindAddress = value
			}
			return generated, bindAddress, true, nil
		}
	}
	if utils.FileExists(legacy) {
		if err := validateComposeServicesFile(legacy); err == nil {
			return legacy, defaultMonitoringAddress, false, nil
		}
	}
	return "", "", false, fmt.Errorf("unable to find a parseable monitoring compose model (checked %s and %s); restore one of these files or stop the monitoring project with Docker Compose directly", generated, legacy)
}

func validateGeneratedComposeFile(file string) error {
	contents, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("unable to read generated compose file %s: %w", file, err)
	}
	var document map[string]any
	if err = yaml.Unmarshal(contents, &document); err != nil {
		return fmt.Errorf("unable to parse generated compose file %s: %w", file, err)
	}
	if err = validateSecureComposeDocument(document); err != nil {
		return fmt.Errorf("generated compose file %s is not secure: %w", file, err)
	}
	return nil
}

func validateComposeServicesFile(file string) error {
	contents, err := readRegularMonitoringFile(file)
	if err != nil {
		return fmt.Errorf("unable to read compose file %s: %w", file, err)
	}
	var document map[string]any
	if err = yaml.Unmarshal(contents, &document); err != nil {
		return fmt.Errorf("unable to parse compose file %s: %w", file, err)
	}
	services, ok := stringMap(document["services"])
	if !ok {
		return fmt.Errorf("compose file %s does not define services", file)
	}
	if _, ok = stringMap(services["grafana"]); !ok {
		return fmt.Errorf("compose file %s does not define Grafana", file)
	}
	if _, ok = stringMap(services["prometheus"]); !ok {
		return fmt.Errorf("compose file %s does not define Prometheus", file)
	}
	return nil
}

func (m *monitoring) warnLegacyConfiguration() {
	m.cmd.Println("WARNING: using a legacy monitoring configuration; rerun 'cohctl init monitoring' to generate a protected Grafana credential and explicit bind address.")
	compose, _ := os.ReadFile(filepath.Join(m.monitoringDir, dockerComposeYAML))
	for _, warning := range legacyPortWarnings(compose) {
		m.cmd.Println(warning)
	}
	ini, _ := os.ReadFile(filepath.Join(m.monitoringDir, grafanaINIFile))
	if regexp.MustCompile(`(?im)^\s*admin_password\s*=\s*(admin)?\s*$`).Match(ini) || !regexp.MustCompile(`(?im)^\s*admin_password\s*=`).Match(ini) {
		m.cmd.Println("WARNING: the legacy Grafana administrator password is missing, empty, or uses the default value.")
	}
}

func (m *monitoring) downloadDashboards(readURL func(string) ([]byte, error)) error {
	for _, file := range dashboardFiles {
		m.cmd.Println(" - ", file)
		contents, err := readURL(fmt.Sprintf("%s/%s", dashboardBaseURL, file))
		if err != nil {
			return fmt.Errorf("error downloading file %s: %w", file, err)
		}
		if err = writeFileContents(m.dashboardsDir, file, contents); err != nil {
			return err
		}
	}
	return nil
}

func (m *monitoring) downloadDockerComposeFiles(readURL func(string) ([]byte, error)) error {
	baseURL, err := monitoringConfigBaseURL(Version)
	if err != nil {
		return err
	}
	for _, file := range dockerComposeFiles {
		m.cmd.Println(" - ", file)
		contents, err := readURL(fmt.Sprintf("%s/%s", baseURL, file))
		if err != nil {
			return fmt.Errorf("error downloading file %s: %w", file, err)
		}
		if err = writeFileContents(m.monitoringDir, file, contents); err != nil {
			return err
		}
	}
	return nil
}

func monitoringConfigBaseURL(version string) (string, error) {
	version = strings.TrimSpace(version)
	if !monitoringReleaseVersionPattern.MatchString(version) {
		return "", fmt.Errorf("unable to download monitoring configuration: cohctl version %q is not a valid release tag", version)
	}
	return fmt.Sprintf("%s/refs/tags/%s/monitoring", configRepositoryURL, version), nil
}

func (m *monitoring) discoverImages() error {
	composeFile := m.composeFile
	if composeFile == "" {
		composeFile = filepath.Join(m.monitoringDir, dockerComposeYAML)
	}
	contents, err := os.ReadFile(composeFile)
	if err != nil {
		return fmt.Errorf("unable to read file %s: %w", composeFile, err)
	}
	prometheus := regexp.MustCompile(`(?m)^\s*image:\s*[\"']?(prom/prometheus:[^\s\"']+)`).FindSubmatch(contents)
	grafana := regexp.MustCompile(`(?m)^\s*image:\s*[\"']?(grafana/grafana:[^\s\"']+)`).FindSubmatch(contents)
	if prometheus == nil {
		return fmt.Errorf("unable to find Prometheus image in %s", composeFile)
	}
	if grafana == nil {
		return fmt.Errorf("unable to find Grafana image in %s", composeFile)
	}
	m.prometheusImage = string(prometheus[1])
	m.grafanaImage = string(grafana[1])
	return nil
}

func (m *monitoring) dockerCommand(args []string) error {
	m.cmd.Printf("Issuing docker %s\n", strings.Join(args, " "))
	return executeHostCommand(m.cmd, nil, "docker", args...)
}

func (m *monitoring) composeInvocation(args []string) (string, []string) {
	if useDockerComposeParam {
		return dockerCompose, args
	}
	return "docker", append([]string{"compose"}, args...)
}

func (m *monitoring) composeArgs(composeFile, bindAddress string, args []string) ([]string, []string) {
	generated := filepath.Base(composeFile) == generatedDockerComposeYAML
	result := make([]string, 0, len(args)+4)
	environment := []string(nil)
	if generated {
		envFile := filepath.Join(filepath.Dir(composeFile), monitoringEnvFile)
		if _, err := readMonitoringEnv(filepath.Dir(composeFile)); err == nil {
			result = append(result, "--env-file", envFile)
		}
		environment = []string{monitoringBindAddressEnv + "=" + bindAddress}
	}
	result = append(result, "-f", composeFile)
	result = append(result, args...)
	return result, environment
}

func (m *monitoring) runCompose(composeFile, bindAddress string, args []string) error {
	composeArgs, environment := m.composeArgs(composeFile, bindAddress, args)
	command, finalArgs := m.composeInvocation(composeArgs)
	m.cmd.Printf("Issuing %s %s\n", command, strings.Join(finalArgs, " "))
	return executeHostCommand(m.cmd, environment, command, finalArgs...)
}

func (m *monitoring) composeOutput(composeFile, bindAddress string, args []string) (string, error) {
	composeArgs, environment := m.composeArgs(composeFile, bindAddress, args)
	command, finalArgs := m.composeInvocation(composeArgs)
	return executeHostCommandOutput(environment, command, finalArgs...)
}

func (m *monitoring) requireComposeV2() error {
	command, args := m.composeInvocation([]string{"version", "--short"})
	output, err := executeHostCommandOutput(nil, command, args...)
	if err != nil {
		return fmt.Errorf("Docker Compose V2 is required for generated monitoring configurations: %w", err)
	}
	major, err := parseComposeMajorVersion(output)
	if err != nil {
		return fmt.Errorf("Docker Compose V2 is required but its version could not be verified; upgrade Docker Compose: %w", err)
	}
	if major < 2 {
		return fmt.Errorf("Docker Compose V2 or newer is required for generated monitoring configurations; found version %s", strings.TrimSpace(output))
	}
	return nil
}

func monitoringNotValid(message string) error {
	return fmt.Errorf("unable to validate monitoring because %s; please run 'cohctl init monitoring'", message)
}

func writeFileContents(base, file string, content []byte) error {
	destination := filepath.Join(base, file)
	// #nosec G306 G703 -- caller-selected local staging directory and fixed asset names; non-secret Docker configuration must be readable on Linux.
	if err := os.WriteFile(destination, content, 0644); err != nil {
		return fmt.Errorf("error writing file %s: %w", destination, err)
	}
	return nil
}

func ensureDirectory(directory string) error {
	if err := utils.EnsureDirectoryWithPerms(directory, 0755); err != nil {
		return fmt.Errorf("unable to create directory %s: %w", directory, err)
	}
	return nil
}

func executeHostCommand(cmd *cobra.Command, environment []string, name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Env = withEnvironment(environment)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout: %w", err)
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr: %w", err)
	}
	if err = command.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}
	go streamOutput(stdout, cmd.OutOrStdout())
	go streamOutput(stderr, cmd.ErrOrStderr())
	if err = command.Wait(); err != nil {
		return fmt.Errorf("command finished with error: %w", err)
	}
	return nil
}

func executeHostCommandOutput(environment []string, name string, args ...string) (string, error) {
	// #nosec G702 -- callers select docker/docker-compose; arguments are separate argv entries, with no shell evaluation.
	command := exec.Command(name, args...)
	command.Env = withEnvironment(environment)
	contents, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s failed: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(contents)))
	}
	return string(contents), nil
}

func withEnvironment(overrides []string) []string {
	environment := os.Environ()
	for _, override := range overrides {
		key, _, _ := strings.Cut(override, "=")
		prefix := key + "="
		filtered := environment[:0]
		for _, existing := range environment {
			if !strings.HasPrefix(existing, prefix) {
				filtered = append(filtered, existing)
			}
		}
		environment = append(filtered, override)
	}
	return environment
}

func streamOutput(reader io.Reader, writer io.Writer) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		_, _ = fmt.Fprintln(writer, scanner.Text())
	}
}

func init() {
	initMonitoringCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	initMonitoringCmd.Flags().BoolVarP(&useDockerComposeParam, dockerCompose, "D", false, useDockerComposeMessage)
	stopMonitoringCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	stopMonitoringCmd.Flags().BoolVarP(&useDockerComposeParam, dockerCompose, "D", false, useDockerComposeMessage)
	startMonitoringCmd.Flags().BoolVarP(&useDockerComposeParam, dockerCompose, "D", false, useDockerComposeMessage)
	startMonitoringCmd.Flags().BoolVarP(&openHomePageParam, "open", "O", false, "open the monitoring page after starting")
}
