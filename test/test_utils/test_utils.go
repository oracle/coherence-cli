/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package test_utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/onsi/gomega"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestContext is a context to pass to tests
type TestContext struct {
	ClusterName     string
	HttpPort        int
	Url             string
	RestUrl         string
	ExpectedServers int
	Username        string
	Password        string
}

const failedToExecute = "Failed to execute 'cliCmd.Execute()'."
const errMessage = "Error msg "

var (
	currentTestContext *TestContext
	emptyByte          = make([]byte, 0)
)

// SetTestContext sets the current context
func SetTestContext(context *TestContext) {
	currentTestContext = context
}

// GetTestContext gets the current context
func GetTestContext() *TestContext {
	return currentTestContext
}

// CreateTempDirectory creates a temporary directory
func CreateTempDirectory(pattern string) string {
	dir, err := ioutil.TempDir("", pattern)
	if err != nil {
		fmt.Println("Unable to create temporary directory " + err.Error())
	}
	defer os.RemoveAll(dir)

	return dir
}

// FileExistsInDirectory returns true if a file exists in a directory
func FileExistsInDirectory(dir string, file string) bool {
	files, err := ioutil.ReadDir(dir)

	if err != nil {
		return false
	}

	for _, f := range files {
		if f.Name() == file {
			return true
		}
	}
	return false
}

// CreateNewConfigYaml creates a full path of a new directory and config
func CreateNewConfigYaml(name string) (string, error) {
	dir := CreateTempDirectory("temp")
	err := os.Mkdir(dir, 0755)
	if err != nil {
		return "", errors.New("Unable to create directory: " + err.Error())
	}

	return filepath.Join(dir, name), nil
}

// EnsureCommandContains executes a command and checks that it contains the output expected
func EnsureCommandContains(g *WithT, t *testing.T, command *cobra.Command, expected string, args ...string) {
	_, output, err := ExecuteCommand(t, command, args...)
	t.Log("Actual Output=[" + output + "], expected to contain=[" + expected + "]")
	if err != nil {
		t.Fatal(failedToExecute, errMessage, err)
	}
	g.Expect(strings.Contains(output, expected)).To(Equal(true))
}

// EnsureCommandNotContains executes a command and checks that it does not contain the output expected
func EnsureCommandNotContains(g *WithT, t *testing.T, command *cobra.Command, expected string, args ...string) {
	_, output, err := ExecuteCommand(t, command, args...)
	t.Log("Actual Output=[" + output + "], expected NOT to contain=[" + expected + "]")
	if err != nil {
		t.Fatal(failedToExecute, errMessage, err)
	}
	g.Expect(strings.Contains(output, expected)).To(Equal(false))
}

// EnsureCommandContainsAll executes a command and checks that it contains all the comma
// separated values in expectedCSV
func EnsureCommandContainsAll(g *WithT, t *testing.T, command *cobra.Command, expectedCSV string, args ...string) {
	_, output, err := ExecuteCommand(t, command, args...)
	t.Log("Actual Output=[" + output + "], expected to contain=[" + expectedCSV + "]")
	if err != nil {
		t.Fatal(failedToExecute, errMessage, err)
	}
	for _, value := range strings.Split(expectedCSV, ",") {
		g.Expect(strings.Contains(output, value)).To(Equal(true))
	}
}

// GetCommandOutput returns the output from a command
func GetCommandOutput(t *testing.T, command *cobra.Command, args ...string) string {
	_, output, err := ExecuteCommand(t, command, args...)
	if err != nil {
		t.Fatal(failedToExecute, errMessage, err)
	}
	return output
}

// EnsureCommandErrorContains executes a command and checks that the error contains expected output
func EnsureCommandErrorContains(g *WithT, t *testing.T, command *cobra.Command, expected string, args ...string) {
	_, output, err := ExecuteCommand(t, command, args...)
	g.Expect(err).NotTo(BeNil())
	errString := err.Error()
	t.Log("Error=[" + errString + "], output=[" + output + "]")
	g.Expect(strings.Contains(errString, expected)).To(Equal(true))
}

// EnsureCommandOutputEquals executes a command and checks that it equals the output expected
func EnsureCommandOutputEquals(g *WithT, t *testing.T, command *cobra.Command, expected string, args ...string) {
	_, output, err := ExecuteCommand(t, command, args...)
	t.Log("Actual Output=[" + output + "], expected=[" + expected + "]")
	if err != nil {
		t.Fatal(failedToExecute, errMessage, err)
	}
	g.Expect(output == expected).To(Equal(true))
}

// ExecuteCommand executes a given command with the arguments provided
func ExecuteCommand(t *testing.T, root *cobra.Command, args ...string) (c *cobra.Command, output string, err error) {
	t.Log("Executing with args " + strings.Join(args, " "))
	var bufferResults = new(bytes.Buffer)
	root.SetOut(bufferResults)
	root.SetErr(bufferResults)
	root.SetArgs(args)

	c, err = root.ExecuteC()

	return c, bufferResults.String(), err
}

// GetFilePath returns the file path of a file
func GetFilePath(fileName string) string {
	_, c, _, _ := runtime.Caller(0)
	dir := filepath.Dir(c)
	return dir + string(os.PathSeparator) + fileName
}

// StartCoherenceCluster starts a Coherence cluster
func StartCoherenceCluster(fileName, url string) error {
	output, err := DockerComposeUp(fileName)
	if err != nil {
		return errors.New(output + ": " + err.Error())
	} else {
		// wait for ready
		if err = WaitForHttpReady(url, 120); err != nil {
			return errors.New("Unable to start cluster: " + err.Error())
		}
	}
	return nil
}

// DockerComposeUp runs docker-compose up on a given file
func DockerComposeUp(composeFile string) (string, error) {
	fmt.Println("Issuing docker-compose up with file " + composeFile)

	output, err := ExecuteHostCommand("docker-compose", "-f", composeFile, "--env-file", "../../test_utils/.env", "up", "-d")

	if err != nil {
		fmt.Println(output)
		return "", err
	}
	fmt.Println(output)

	return output, err
}

// CollectDockerLogs collects docker logs
func CollectDockerLogs() error {
	var (
		output    string
		err       error
		logs      string
		file      *os.File
		directory = GetFilePath("../../build/_output/test-logs/")
	)
	output, err = ExecuteHostCommand("docker", "ps", "-q")
	if err != nil {
		return err
	}

	for _, container := range strings.Split(output, "\n") {
		if container == "" {
			continue
		}

		logs, err = ExecuteHostCommand("docker", "logs", container)
		if err != nil {
			return err
		}

		//write to build output directory
		fileName := filepath.Join(directory, container+".logs")

		fmt.Println("Dumping logs for " + container + " to " + fileName)

		file, err = os.Create(fileName)
		if err != nil {
			return err
		}
		_, err = file.WriteString(logs)
		if err != nil {
			return err
		}

		_ = file.Close()
	}

	return nil
}

// DockerComposeDown runs docker-compose down on a given file
func DockerComposeDown(composeFile string) (string, error) {
	fmt.Println("Issuing docker-compose down with file " + composeFile)
	// sleep as sometimes docker compose networks are not completely stopped
	Sleep(5)

	output, err := ExecuteHostCommand("docker-compose", "-f", composeFile, "down")

	if err != nil {
		fmt.Println(output)
		return "", err
	}
	return output, err
}

// StartDockerImage starts a coherence image using docker
func StartDockerImage(t *testing.T, image string, name string, httpPort int, clusterName string, delete bool) (string, error) {
	t.Log(fmt.Sprintf("Starting docker image %s with image name %s, httpPort %d and clusterName %s",
		image, name, httpPort, clusterName))
	if delete {
		_, _ = StopDockerImage(name)
	}

	var ports = fmt.Sprintf("%d:%d", httpPort, httpPort)
	output, err := ExecuteHostCommand("docker", "run", "-d", "-e", "COHERENCE_CLUSTER="+clusterName,
		"--rm", "--name", name, "-p", ports, image)

	if err != nil {
		t.Log("Error starting image: " + err.Error())
		t.Log(output)
		t.Fatal("Unable to start image " + image)
	}

	// once we get here the container name is in output
	output = strings.ReplaceAll(output, "\n", "")
	t.Log("Started container " + output)

	return output, WaitForHttpReady(GetManagementUrl(httpPort), 120)
}

// GetManagementUrl returns the management URL given a management port
func GetManagementUrl(httpPort int) string {
	return fmt.Sprintf("http://localhost:%d/management/coherence/cluster", httpPort)
}

// GetRestUrl returns the REST URL
func GetRestUrl(restPort int) string {
	return fmt.Sprintf("http://localhost:%d", restPort)
}

// IssueGetRequest issues a HTTP GET request using the URL
func IssueGetRequest(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return emptyByte, err
	}

	if resp.StatusCode != 200 {
		return emptyByte, errors.New("Did not receive a 200 response code: " + resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return emptyByte, err
	}

	return body, nil
}

// IssuePostRequest issues a HTTP POST request using the URL
func IssuePostRequest(url string) ([]byte, error) {
	resp, err := issueRequest("POST", url, emptyByte)

	if err != nil {
		return emptyByte, err
	}

	if resp.StatusCode != 200 {
		return emptyByte, errors.New("Did not receive a 200 response code: " + resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return emptyByte, err
	}

	return body, nil
}

func issueRequest(requestType, url string, data []byte) (*http.Response, error) {
	var (
		err error
		req *http.Request
	)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}
	cookies, _ := cookiejar.New(nil)
	client := &http.Client{Transport: tr,
		Timeout: time.Duration(120) * time.Second,
		Jar:     cookies,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	req, err = http.NewRequest(requestType, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

// WaitForHttpReady waits for the HTTP endpoint to be ready
func WaitForHttpReady(url string, timeout int) error {
	var duration = 0
	for duration < timeout {
		_, err := IssueGetRequest(url)
		if err != nil {
			// unable to connect, so wait 5 seconds
			fmt.Println("Waiting to connect to " + url + ", sleeping 5")
			Sleep(5)
			duration += 5
		} else {
			fmt.Println("HTTP endpoint ready")
			return nil
		}
	}

	return fmt.Errorf("Unable to connect to url %s after %d seconds\n", url, timeout)
}

// WaitForHttpBalancedServices waits for all services to be balanced
func WaitForHttpBalancedServices(url string, timeout int) error {
	var duration = 0
	fmt.Println("Waiting for services to be balanced...")
	for duration < timeout {
		content, err := IssueGetRequest(url)
		if err != nil {
			// unable to connect, so wait 5 seconds
			fmt.Println("Waiting for services " + url + ", sleeping 5")
			Sleep(5)
			duration += 5
		} else {
			var contentString = string(content)
			if contentString == "OK" {
				fmt.Println("All services balanced")
				return nil
			}
			fmt.Println(contentString)
			Sleep(5)
			duration += 5
		}
	}

	return fmt.Errorf("Unable to connect to url %s after %d seconds\n", url, timeout)
}

// WaitForIdlePersistence waits for idle persistence coordinator which means the last operation has completed
func WaitForIdlePersistence(timeout int, dataFetcher fetcher.Fetcher, serviceName string) error {
	var (
		duration    = 0
		coordinator = config.PersistenceCoordinator{}
		coordData   []byte
		err         error
		status      string
	)
	for duration < timeout {
		coordData, err = dataFetcher.GetPersistenceCoordinator(serviceName)
		if err != nil {
			return err
		}

		err = json.Unmarshal(coordData, &coordinator)
		if err != nil {
			return err
		}

		idle := coordinator.Idle
		status = coordinator.OperationStatus

		if idle {
			return nil
		}
		// not idle
		fmt.Printf("Current status for service %s is %s, sleeping 5\n", status, serviceName)
		Sleep(5)
		duration += 5
	}

	return fmt.Errorf("Unable to get to idle for service %s after %d seconds. Last status %s",
		serviceName, timeout, status)
}

// Sleep will sleep for a duration of seconds
func Sleep(seconds int) {
	time.Sleep(time.Duration(seconds) * time.Second)
}

// StopDockerImage stops a docker image with the given name
func StopDockerImage(name string) (string, error) {
	return ExecuteHostCommand("docker", "stop", name)
}

// ExecuteHostCommand executes a host command
func ExecuteHostCommand(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	stdout, err := cmd.CombinedOutput()

	var stringStdOut = string(stdout)

	if err != nil {
		return stringStdOut, err
	}

	return stringStdOut, nil
}

// CleanupConfigFileAfterTest cleans up a config file after a test
func CleanupConfigFileAfterTest(t *testing.T, file string) {
	t.Cleanup(func() {
		_ = os.Remove(file)
	})
}

// CleanupDirectoryAfterTest cleans up a directory after a test
func CleanupDirectoryAfterTest(t *testing.T, dir string) {
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
}
