/*
 * Copyright (c) 2021, 2025 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/oracle/coherence-go-client/v2/coherence/discovery"
	"github.com/spf13/cobra"
)

var (
	nsQuery string
)

// nsLookupCmd represents the nslookup command.
var nsLookupCmd = &cobra.Command{
	Use:   "nslookup <host:port>",
	Short: "execute a Coherence Name Service lookup",
	Long: `The 'nslookup' command looks up various Name Service endpoints for a cluster host/port.
The various options to pass via -q option include: Cluster/name, Cluster/info, NameService/string/Cluster/foreign,
NameService/string/management/HTTPManagementURL, NameService/string/management/JMXServiceURL,
NameService/string/metrics/HTTPMetricsURL, NameService/string/$GRPC:GrpcProxy,
NameService/string/health/HTTPHealthURL and NameService/string/Cluster/foreign/<clustername>/NameService/localPort`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			hostPorts []string
			err       error
			ns        *discovery.NSLookup
			count     = len(args)
			result    string
		)

		err = validateTimeout(timeout)
		if err != nil {
			return err
		}

		if count == 0 {
			hostPorts = []string{"localhost"}
		} else {
			hostPorts = args
		}

		for _, address := range hostPorts {
			ns, err = discovery.Open(address, timeout)
			if err != nil {
				err = logErrorAndCheck(cmd, "unable to connect to "+address, err)
				if err != nil {
					return err
				}
				// skip to the next address
				closeSilent(ns)
				continue
			}

			result, err = ns.Lookup(nsQuery)
			if err != nil {
				err = logErrorAndCheck(cmd, "unable to lookup using "+nsQuery, err)
				if err != nil {
					return err
				}
				// skip to the next address
				closeSilent(ns)
				continue
			}

			cmd.Println(result)

			closeSilent(ns)
		}

		return nil
	},
}

func init() {
	nsLookupCmd.Flags().StringVarP(&nsQuery, "query", "q", "",
		"query string to pass to Name Service lookup")
	_ = nsLookupCmd.MarkFlagRequired("query")
	nsLookupCmd.PersistentFlags().BoolVarP(&ignoreErrors, "ignore", "I", false, ignoreErrorsMessage)
	nsLookupCmd.Flags().Int32VarP(&timeout, "timeout", "t", 30, timeoutMessage)
}
