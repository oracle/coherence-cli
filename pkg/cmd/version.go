/*
 * Copyright (c) 2021, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"
)

var checkForUpdates bool

const stableURL = "https://oracle.github.io/coherence-cli/stable.txt"
const updateURL = "https://github.com/oracle/coherence-cli/releases"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long: `The 'get version' command displays version and build details for the Coherence-CLI.
Use the '-u' option to check for updates.`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		var (
			err           error
			latestVersion string
		)

		cmd.Printf("Coherence Command Line Interface\nCLI Version:  %s\nDate:         %s\n"+
			"Commit:       %s\nOS:           %s\nArchitecture: %s\n",
			Version, Date, Commit, runtime.GOOS, runtime.GOARCH)

		if checkForUpdates {
			cmd.Println("\nChecking for updates...")
			latestVersion, err = getLatestVersion()
			if err != nil {
				cmd.Printf("Error: unable to check for updates: %s\n", err.Error())
				cmd.Println("If you are behind a Proxy Server then set the HTTP_PROXY environment variable")
				cmd.Println("E.g. HTTP_PROXY=http://proxy-host:proxy-port/")
			} else {
				latestVersion = strings.ReplaceAll(latestVersion, "\n", "")
				// we now have a value for the latest version
				if isVersionUpdateAvailable(Version, latestVersion) {
					cmd.Printf("A newer version of cohctl (%s) is available.\nPlease visit the following URL to update:\n%s\n",
						latestVersion, updateURL)
				} else {
					cmd.Println("you are on the latest version")
				}
			}
		}
	},
}

// isVersionUpdate returns true if there is a new stable version available
func isVersionUpdateAvailable(haveVersion, stableVersion string) bool {
	return haveVersion != stableVersion
}

func init() {
	versionCmd.Flags().BoolVarP(&checkForUpdates, "check-updates", "u", false,
		"If true, will check for updates")
}

func getLatestVersion() (string, error) {
	var (
		err       error
		req       *http.Request
		resp      *http.Response
		body      []byte
		buffer    bytes.Buffer
		URL       = url.URL{}
		httpProxy = os.Getenv("HTTP_PROXY")
	)
	cookies, _ := cookiejar.New(nil)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false, MinVersion: tls.VersionTLS12},
	}

	if httpProxy != "" {
		proxy, err := URL.Parse(httpProxy)
		if err != nil {
			return "", errors.New("unable to parse HTTP_PROXY environment variable")
		}
		tr.Proxy = http.ProxyURL(proxy)
	}

	client := &http.Client{Transport: tr,
		Timeout: time.Duration(fetcher.RequestTimeout) * time.Second,
		Jar:     cookies,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	req, err = http.NewRequest("GET", stableURL, bytes.NewBuffer(constants.EmptyByte))
	if err != nil {
		return "", err
	}

	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unable to issue GET to %s: response=%s",
			stableURL, resp.Status)
	}

	_, err = io.Copy(&buffer, resp.Body)
	if err != nil {
		return "", err
	}

	body = buffer.Bytes()
	return string(body), nil
}
