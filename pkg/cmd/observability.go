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
	"strings"
)

const (
	observabilityDirectory = "observability"
	dashboardsDirectory    = "dashboards"
	dashboardBaseURL       = "https://raw.githubusercontent.com/oracle/coherence-operator/refs/heads/main/dashboards/grafana"
	configBaseURL          = "https://raw.githubusercontent.com/oracle/coherence-cli/refs/heads/observability/observability"
	//configBaseURL          = "https://raw.githubusercontent.com/oracle/coherence-cli/refs/heads/main/observability"
	grafanaPort    = 3000
	prometheusPort = 9090
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
		"docker-compose.yaml",
		"prometheus.yaml",
	}
)

// initObservabilityCmd represents the init observability command.
var initObservabilityCmd = &cobra.Command{
	Use:   "observability",
	Short: "initializes local observability for Coherence",
	Long: `The 'init observability' initializes local observability for Coherence. 
This involves downloading Grafana and Prometheus docker images and downloading
docker compose files and related dashboards to ~/.cohctl/observability directory.
Use the 'start observability' command to start local Grafana and Prometheus to 
monitor local Coherence clusters.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		// confirm the operation
		cmd.Println("This command will:")
		cmd.Printf("1. Create a directory %s/%s\n", cfgDirectory, observabilityDirectory)
		cmd.Println("2. Download Grafana dashboards")
		cmd.Println("2. Download docker compose files")
		cmd.Println("3. Pull Grafana and Prometheus images")
		cmd.Println()
		if !confirmOperation(cmd, "Are you sure you want to initialize observability? (y/n) ") {
			return nil
		}

		// ensure the base directory
		obs := newObservability(cmd)

		cmd.Println("Ensuring directories...")
		err := obs.ensureDirectories()
		if err != nil {
			return err
		}

		cmd.Println("Downloading Grafana dashboards...")
		err = obs.downloadDashboards()
		if err != nil {
			return err
		}

		cmd.Println("Downloading docker compose files...")
		err = obs.downloadDockerComposeFiles()
		if err != nil {
			return err
		}

		cmd.Println("Pulling docker images...")

		err = obs.dockerCommand([]string{"pull", obs.prometheusImage})
		if err != nil {
			return err
		}
		err = obs.dockerCommand([]string{"pull", obs.grafanaImage})
		if err != nil {
			return err
		}

		cmd.Println(OperationCompleted)
		return nil
	},
}

type grafanaStatus struct {
	Database string `json:"database"`
	Version  string `json:"version"`
}

// getObservabilityCmd represents the get observability command.
var getObservabilityCmd = &cobra.Command{
	Use:   "observability",
	Short: "returns observability status",
	Long: `The 'get observability' gets the observability status and ensures
the environment is setup`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		// ensure the base directory
		var (
			obs              = newObservability(cmd)
			grafanaOutput    string
			promOutput       string
			err              = obs.validateEnvironment()
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

		cmd.Println("Observability status")
		cmd.Printf("Grafana:    %s\n", grafanaURL)
		cmd.Printf("  Status:   %s\n", grafanaOutput)
		cmd.Printf("Prometheus: %s\n", promURL)
		cmd.Printf("  Status:   %s\n", promOutput)
		cmd.Println("Docker")
		return obs.dockerCommand([]string{"ps"})
	},
}

// startObservabilityCmd represents the start observability command.
var startObservabilityCmd = &cobra.Command{
	Use:   "observability",
	Short: "starts the observability stack",
	Long: `The 'start observability' starts the observability stack, Grafana and 
Prometheus using docker compose`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		// ensure the base directory
		obs := newObservability(cmd)

		err := obs.validateEnvironment()

		if err != nil {
			return err
		}

		err = obs.dockerCommand([]string{"compose", "-f", path.Join(obs.observabilityDir, "docker-compose.yaml"), "up", "-d"})
		if err != nil {
			return err
		}

		return obs.dockerCommand([]string{"ps"})
	},
}

// stopObservabilityCmd represents the stop command.
var stopObservabilityCmd = &cobra.Command{
	Use:   "observability",
	Short: "stops the observability stack",
	Long: `The 'stop observability' stops the observability stack, Grafana and Prometheus
using docker compose`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		// ensure the base directory
		obs := newObservability(cmd)

		err := obs.validateEnvironment()

		if err != nil {
			return err
		}

		if !confirmOperation(cmd, "Are you sure you want to stop observability? (y/n) ") {
			return nil
		}

		err = obs.dockerCommand([]string{"compose", "-f", path.Join(obs.observabilityDir, "docker-compose.yaml"), "down"})
		if err != nil {
			return err
		}

		return obs.dockerCommand([]string{"ps"})
	},
}

type observability struct {
	observabilityDir string
	dashboardsDir    string
	cmd              *cobra.Command
	grafanaImage     string
	prometheusImage  string
}

func newObservability(cmd *cobra.Command) *observability {
	obs := &observability{cmd: cmd}
	obs.observabilityDir = path.Join(cfgDirectory, observabilityDirectory)
	obs.dashboardsDir = path.Join(obs.observabilityDir, dashboardsDirectory)
	obs.grafanaImage = "grafana/grafana:11.6.2"
	obs.prometheusImage = "prom/prometheus:v2.53.4"
	return obs
}

func (o *observability) String() string {
	return fmt.Sprintf("observabilityDir=%s, bashboardsDir=%s", o.observabilityDir, o.dashboardsDir)
}

func (o *observability) ensureDirectories() error {
	// setup observability directory
	err := ensureDirectory(o.observabilityDir)
	if err != nil {
		return err
	}

	// setup dashboard directory
	err = ensureDirectory(o.dashboardsDir)
	if err != nil {
		return err
	}

	return nil
}

// validateEnvironment validates that the environment is setup to start observability.
func (o *observability) validateEnvironment() error {
	if !utils.DirectoryExists(o.observabilityDir) {
		return observabilityNotValid(o.observabilityDir + " does not exist")
	}

	if !utils.DirectoryExists(o.dashboardsDir) {
		return observabilityNotValid(o.dashboardsDir + " does not exist")
	}

	// check each of the dashboard files
	for _, file := range dashboardFiles {
		destPath := filepath.Join(o.dashboardsDir, file)

		if !utils.FileExists(destPath) {
			return observabilityNotValid(destPath + " does not exist, or is not a file")
		}
	}

	// check each of the docker compose files
	for _, file := range dockerComposeFiles {
		destPath := filepath.Join(o.observabilityDir, file)
		if !utils.FileExists(destPath) {
			return observabilityNotValid(destPath + " does not exist, or is not a file")
		}
	}

	return nil
}

// downloadDashboards downloads all Grafana dashboards.
func (o *observability) downloadDashboards() error {
	for _, file := range dashboardFiles {
		o.cmd.Println(" - ", file)
		url := fmt.Sprintf("%s/%s", dashboardBaseURL, file)
		response, err := GetURLContents(url)
		if err != nil {
			return fmt.Errorf("error downloading file %s: %w", file, err)
		}

		if err = writeFileContents(o.dashboardsDir, file, response); err != nil {
			return err
		}
	}

	return nil
}

// downloadDockerComposeFiles downloads files required by docker compose.
// we specifically don't use docker libraries to start docker to minimize
// dependencies and size of  the cohctl executable.
func (o *observability) downloadDockerComposeFiles() error {
	for _, file := range dockerComposeFiles {
		o.cmd.Println(" - ", file)
		url := fmt.Sprintf("%s/%s", configBaseURL, file)
		response, err := GetURLContents(url)
		if err != nil {
			return fmt.Errorf("error downloading file %s: %w", file, err)
		}

		if err = writeFileContents(o.observabilityDir, file, response); err != nil {
			return err
		}
	}

	return nil
}

func (o *observability) dockerCommand(args []string) error {
	o.cmd.Printf("Issuing docker %s\n", strings.Join(args, " "))

	return executeHostCommand(o.cmd, "docker", args...)
}

func observabilityNotValid(message string) error {
	return fmt.Errorf("unable to validate observability due to %s, please run 'cohctl init observability'", message)
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
	initObservabilityCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
	stopObservabilityCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
}
