/*
 * Copyright (c) 2021, 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	setIgnoreCertsMsg   = "Value is now set to "
	setIgnoreCertsError = "you can only specify 'true' or 'false'"
)

var (
	getIgnoreCertsMsg = "Current setting: "
)

// getIgnoreCertsCmd represents the get ignore-certs.
var getIgnoreCertsCmd = &cobra.Command{
	Use:   ignoreCertsFlag,
	Short: "display the saved connection's certificate-validation setting",
	Long: `The 'get ignore-certs' command displays the certificate-validation setting for
the saved connection selected by -c/--connection, or the current context.
For HTTPS, false means certificates are verified and true allows invalid certificates
only for that connection's HTTPS origin. HTTP and other connections report n/a.
Use 'get config -v' to review all saved choices. There is no global setting.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		connection, err := selectedCertificateConnection()
		if err != nil {
			return err
		}
		cmd.Println(getIgnoreCertsMsg + connectionCertificateSetting(connection))
		return nil
	},
}

// setIgnoreInvalidCertsCmd represents the set ignore-certs command.
var setIgnoreCertsCmd = &cobra.Command{
	Use:   "ignore-certs {true|false}",
	Short: "set the saved HTTPS connection's certificate-validation setting",
	Long: `The 'set ignore-certs' command changes only the saved HTTPS connection selected
by -c/--connection, or the current context. False verifies certificates; true allows
invalid certificates only for that connection's HTTPS origin. Use true only when you
are sure of the identity of the target server. HTTP rejects both values. This command
requires an existing saved connection; an initial exception must use 'add cluster
... --ignore-certs'. Prefer COHERENCE_TLS_CERTS_PATH with a valid matching certificate.
Downloads, discovery and other-origin destinations remain verified. Stderr warns once
per connection/origin per execution on actual use; the log records every bypass attempt.`,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("you must provide either true or false")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		value := args[0]
		var setting bool
		if value == stringTrue {
			setting = true
		} else if value == stringFalse {
			setting = false
		} else {
			return errors.New(setIgnoreCertsError)
		}

		connection, err := selectedCertificateConnection()
		if err != nil {
			return err
		}
		if !fetcher.IsValidHTTPSURL(connection.ConnectionURL) {
			return fmt.Errorf("connection %q must have a valid HTTPS URL to set ignore-certs", connection.Name)
		}
		if err = saveConnectionCertificateChoice(connection, setting); err != nil {
			return fmt.Errorf("unable to write config: %w", err)
		}
		cmd.Println(setIgnoreCertsMsg + value)
		return nil
	},
}

// Keep other saved entries and unknown/null extensions intact, as in migration.
// Reuse the existing staged field writer rather than rewriting the typed list.
func saveConnectionCertificateChoice(connection ClusterConnection, setting bool) error {
	return saveConnectionCertificateChoiceWithOps(connection, setting, defaultConfigMigrationOps())
}

func saveConnectionCertificateChoiceWithOps(connection ClusterConnection, setting bool, ops configMigrationOps) error {
	live := viper.GetViper()
	var entries []any
	err := updateConfigFile(live.ConfigFileUsed(), clusterKey, func(data map[string]any) error {
		key, err := migrationKey(data, clusterKey)
		if err != nil {
			return err
		}
		entries, _ = data[key].([]any)
		for _, value := range entries {
			entry, ok := value.(map[string]any)
			if !ok {
				continue
			}
			nameKey, keyErr := migrationKey(entry, "name")
			if keyErr != nil {
				return keyErr
			}
			if entry[nameKey] != connection.Name {
				continue
			}
			urlKey, keyErr := migrationKey(entry, "connectionurl")
			if keyErr != nil {
				return keyErr
			}
			if entry[urlKey] != connection.ConnectionURL {
				break // Do not apply a choice to an externally changed target.
			}
			choiceKey, keyErr := migrationKey(entry, ignoreCertsContextKey)
			if keyErr != nil {
				return keyErr
			}
			entry[choiceKey] = setting
			return nil
		}
		return fmt.Errorf("saved connection %q changed; reload the configuration and retry", connection.Name)
	}, ops)
	if err != nil {
		return err
	}
	live.Set(clusterKey, entries)
	for i := range Config.Clusters {
		if Config.Clusters[i].Name == connection.Name {
			Config.Clusters[i].IgnoreInvalidCerts = &setting
			break
		}
	}
	return nil
}

// Selection here deliberately never invokes discovery or constructs a fetcher.
func selectedCertificateConnection() (ClusterConnection, error) {
	name, err := GetConnectionNameFromContextOrArg()
	if err != nil {
		return ClusterConnection{}, err
	}
	if strings.Contains(name, ":") {
		return ClusterConnection{}, errors.New("ignore-certs requires a saved connection name; host:port discovery is not supported")
	}
	found, connection := GetClusterConnection(name)
	if !found {
		return ClusterConnection{}, errors.New(UnableToFindClusterMsg + name)
	}
	return connection, nil
}

func connectionCertificateSetting(connection ClusterConnection) string {
	if !fetcher.IsValidHTTPSURL(connection.ConnectionURL) {
		return na
	}
	return strconv.FormatBool(connection.IgnoreInvalidCerts != nil && *connection.IgnoreInvalidCerts)
}
