/*
 * Copyright (c) 2025 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	monitoringDirectory = "monitoring"
	dashboardsDirectory = "dashboards"
	dashboardBaseURL    = "https://raw.githubusercontent.com/oracle/coherence-operator/refs/heads/main/dashboards/grafana"
	configBaseURL       = "https://raw.githubusercontent.com/oracle/coherence-cli/refs/heads/observability/monitoring"
	//configBaseURL          = "https://raw.githubusercontent.com/oracle/coherence-cli/refs/heads/main/monitoring"
	grafanaPort       = 3000
	prometheusPort    = 9090
	dockerComposeYAML = "docker-compose.yaml"
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
		"grafana.ini",
		"dashboards.yaml",
		"datasources.yaml",
		dockerComposeYAML,
		"prometheus.yaml",
	}
)

// initMonitoringCmd represents the init monitoring command.
var initMonitoringCmd = &cobra.Command{
	Use:   "monitoring",
	Short: "initializes local monitoring for Coherence",
	Long: `The 'init monitoring' initializes local monitoring for Coherence. 
This involves downloading Grafana and Prometheus docker images and downloading
docker compose files and related dashboards to ~/.cohctl/monitoring directory.
Use the 'start monitoring' command to start local Grafana and Prometheus to 
monitor local Coherence clusters.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		// confirm the operation
		cmd.Println("This command will:")
		cmd.Printf("1. Create a directory %s/%s\n", cfgDirectory, monitoringDirectory)
		cmd.Println("2. Download Grafana dashboards")
		cmd.Println("2. Download docker compose files")
		cmd.Println("3. Pull Grafana and Prometheus images")
		cmd.Println()
		if !confirmOperation(cmd, "Are you sure you want to initialize monitoring? (y/n) ") {
			return nil
		}

		mon := newMonitoring(cmd)

		cmd.Println("Ensuring directories...")
		err := mon.ensureDirectories()
		if err != nil {
			return err
		}

		cmd.Println("Downloading Grafana dashboards...")
		err = mon.downloadDashboards()
		if err != nil {
			return err
		}

		cmd.Println("Downloading docker compose files...")
		err = mon.downloadDockerComposeFiles()
		if err != nil {
			return err
		}

		cmd.Println("Pulling docker images...")
		err = mon.discoverImages()
		if err != nil {
			return err
		}

		err = mon.dockerCommand([]string{"pull", mon.prometheusImage})
		if err != nil {
			return err
		}
		err = mon.dockerCommand([]string{"pull", mon.grafanaImage})
		if err != nil {
			return err
		}

		cmd.Println(OperationCompleted)
		cmd.Println()
		cmd.Printf("Note: You can change the grafana and prometheus image versions by editing: \n  %s\n\n", path.Join(mon.monitoringDir, dockerComposeYAML))
		return nil
	},
}

type grafanaStatus struct {
	Database string `json:"database"`
	Version  string `json:"version"`
}

// getMonitoringCmd represents the get monitoring command.
var getMonitoringCmd = &cobra.Command{
	Use:   "monitoring",
	Short: "returns monitoring status",
	Long: `The 'get monitoring' gets the monitoring status and ensures
the environment is setup correctly.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		// ensure the base directory
		var (
			mon              = newMonitoring(cmd)
			grafanaOutput    string
			promOutput       string
			err              = mon.validateEnvironment()
			promURL          = fmt.Sprintf("http://localhost:%v/", prometheusPort)
			promHealthURL    = fmt.Sprintf("%s-/healthy", promURL)
			grafanaURL       = fmt.Sprintf("http://localhost:%v/d/coh-main/coherence-dashboard-main", grafanaPort)
			grafanaHealthURL = fmt.Sprintf("%sapi/health", grafanaURL)
		)

		if err != nil {
			return err
		}

		promContent, err := GetURLContents(promHealthURL)
		if err != nil {
			promOutput = err.Error()
		} else {
			promOutput = string(promContent)
		}

		grafanaContent, err := GetURLContents(grafanaHealthURL)
		if err != nil {
			grafanaOutput = err.Error()
		} else {
			var status grafanaStatus
			err = json.Unmarshal(grafanaContent, &status)
			if err != nil {
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
		cmd.Printf("Compose:    %s\n", path.Join(mon.monitoringDir, dockerComposeYAML))
		cmd.Println("Docker")
		return mon.dockerCommand([]string{"ps"})
	},
}

// startMonitoringCmd represents the start monitoring command.
var startMonitoringCmd = &cobra.Command{
	Use:   "monitoring",
	Short: "starts the monitoring stack",
	Long: `The 'start monitoring' starts the monitoring stack, Grafana and 
Prometheus using docker compose.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		// ensure the base directory
		mon := newMonitoring(cmd)

		err := mon.validateEnvironment()

		if err != nil {
			return err
		}

		err = mon.dockerCommand([]string{"compose", "-f", path.Join(mon.monitoringDir, dockerComposeYAML), "up", "-d"})
		if err != nil {
			return err
		}

		grafanaURL := fmt.Sprintf("http://localhost:%v/d/coh-main/coherence-dashboard-main", grafanaPort)
		cmd.Printf("\nOpen the Grafana dashboard at %s, using admin/admin\n\n", grafanaURL)

		return mon.dockerCommand([]string{"ps"})
	},
}

// stopMonitoringCmd represents the stop command.
var stopMonitoringCmd = &cobra.Command{
	Use:   "monitoring",
	Short: "stops the monitoring stack",
	Long: `The 'stop monitoring' stops the monitoring stack, Grafana and Prometheus
using docker compose.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		// ensure the base directory
		mon := newMonitoring(cmd)

		err := mon.validateEnvironment()

		if err != nil {
			return err
		}

		if !confirmOperation(cmd, "Are you sure you want to stop monitoring? (y/n) ") {
			return nil
		}

		err = mon.dockerCommand([]string{"compose", "-f", path.Join(mon.monitoringDir, dockerComposeYAML), "down"})
		if err != nil {
			return err
		}

		return mon.dockerCommand([]string{"ps"})
	},
}

type monitoring struct {
	monitoringDir   string
	dashboardsDir   string
	cmd             *cobra.Command
	grafanaImage    string
	prometheusImage string
}

func newMonitoring(cmd *cobra.Command) *monitoring {
	mon := &monitoring{cmd: cmd}
	mon.monitoringDir = path.Join(cfgDirectory, monitoringDirectory)
	mon.dashboardsDir = path.Join(mon.monitoringDir, dashboardsDirectory)
	return mon
}

func (m *monitoring) String() string {
	return fmt.Sprintf("monitoringDir=%s, bashboardsDir=%s", m.monitoringDir, m.dashboardsDir)
}

func (m *monitoring) ensureDirectories() error {
	// setup monitoring directory
	err := ensureDirectory(m.monitoringDir)
	if err != nil {
		return err
	}

	// setup dashboard directory
	err = ensureDirectory(m.dashboardsDir)
	if err != nil {
		return err
	}

	return nil
}

// validateEnvironment validates that the environment is setup to start monitoring.
func (m *monitoring) validateEnvironment() error {
	if !utils.DirectoryExists(m.monitoringDir) {
		return monitoringNotValid(m.monitoringDir + " does not exist")
	}

	if !utils.DirectoryExists(m.dashboardsDir) {
		return monitoringNotValid(m.dashboardsDir + " does not exist")
	}

	// check each of the dashboard files
	for _, file := range dashboardFiles {
		destPath := filepath.Join(m.dashboardsDir, file)

		if !utils.FileExists(destPath) {
			return monitoringNotValid(destPath + " does not exist, or is not a file")
		}
	}

	// check each of the docker compose files
	for _, file := range dockerComposeFiles {
		destPath := filepath.Join(m.monitoringDir, file)
		if !utils.FileExists(destPath) {
			return monitoringNotValid(destPath + " does not exist, or is not a file")
		}
	}

	return m.discoverImages()
}

// downloadDashboards downloads all Grafana dashboards.
func (m *monitoring) downloadDashboards() error {
	for _, file := range dashboardFiles {
		m.cmd.Println(" - ", file)
		url := fmt.Sprintf("%s/%s", dashboardBaseURL, file)
		response, err := GetURLContents(url)
		if err != nil {
			return fmt.Errorf("error downloading file %s: %w", file, err)
		}

		if err = writeFileContents(m.dashboardsDir, file, response); err != nil {
			return err
		}
	}

	return nil
}

// discoverImages discovers the image names from docker-compose.yaml.
func (m *monitoring) discoverImages() error {
	var (
		promRegex    = regexp.MustCompile(`(?m)^\s*image:\s*(prom/prometheus:[^\s]+)`)
		grafanaRegex = regexp.MustCompile(`(?m)^\s*image:\s*(grafana/grafana:[^\s]+)`)
		composeFile  = path.Join(m.monitoringDir, dockerComposeYAML)
	)

	contents, err := os.ReadFile(composeFile)
	if err != nil {
		return fmt.Errorf("unable to read file: %s %v", composeFile, err)
	}

	promLine := promRegex.FindSubmatch(contents)
	grafanaLine := grafanaRegex.FindSubmatch(contents)

	if promLine != nil {
		m.prometheusImage = string(promLine[1])
	} else {
		return fmt.Errorf("unable to find promethues image in %s", dockerComposeYAML)
	}
	if grafanaLine != nil {
		m.grafanaImage = string(grafanaLine[1])
	} else {
		return fmt.Errorf("unable to find grafana image in %s", dockerComposeYAML)
	}

	return nil
}

// downloadDockerComposeFiles downloads files required by docker compose.
// we specifically don't use docker libraries to start docker to minimize
// dependencies and size of  the cohctl executable.
func (m *monitoring) downloadDockerComposeFiles() error {
	for _, file := range dockerComposeFiles {
		m.cmd.Println(" - ", file)
		url := fmt.Sprintf("%s/%s", configBaseURL, file)
		response, err := GetURLContents(url)
		if err != nil {
			return fmt.Errorf("error downloading file %s: %w", file, err)
		}

		if err = writeFileContents(m.monitoringDir, file, response); err != nil {
			return err
		}
	}

	return nil
}

func (m *monitoring) dockerCommand(args []string) error {
	m.cmd.Printf("Issuing docker %s\n", strings.Join(args, " "))

	return executeHostCommand(m.cmd, "docker", args...)
}

func monitoringNotValid(message string) error {
	return fmt.Errorf("unable to validate monitoring due to %s, please run 'cohctl init monitoring'", message)
}

// writeFileContents writes file contents to the location.
func writeFileContents(base, file string, content []byte) error {
	destPath := filepath.Join(base, file)

	if err := os.WriteFile(destPath, content, 0600); err != nil {
		return fmt.Errorf("error writing file %s: %w", destPath, err)
	}

	return nil
}

func ensureDirectory(directory string) error {
	err := utils.EnsureDirectory(directory)
	if err != nil {
		return fmt.Errorf("unable to create directory: %s, %v", directory, err)
	}

	return nil
}

// executeHostCommand executes a host command.
func executeHostCommand(cmd *cobra.Command, name string, arg ...string) error {
	command := exec.Command(name, arg...)

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

	// Use cobra's output writers
	go streamOutput(stdout, cmd.OutOrStdout())
	go streamOutput(stderr, cmd.ErrOrStderr())

	if err = command.Wait(); err != nil {
		return fmt.Errorf("command finished with error: %w", err)
	}

	return nil
}

func streamOutput(r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		_, _ = fmt.Fprintln(w, scanner.Text())
	}
}

func init() {
	initMonitoringCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	stopMonitoringCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
}
