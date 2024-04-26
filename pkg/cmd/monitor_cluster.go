/*
 * Copyright (c) 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"log"
	"os"
	"sync"
	"time"
)

var (
	mutex                  sync.Mutex
	lastClusterSummaryInfo clusterSummaryInfo
	emptyStringArray       = make([]string, 0)
)

var monitorContent = []Panel{
	createContentPanel(50, 6, "Overview", clusterSummary),
	createContentPanel(50, 4, "Members", clusterMembersSummary),
}

// monitorClusterCmd represents the monitor cluster command
var monitorClusterCmd = &cobra.Command{
	Use:               "cluster connection-name",
	Short:             "monitors the cluster using text based UI",
	Long:              `The 'monitor cluster' command displays a text base UI to monitor the overall cluster.`,
	ValidArgsFunction: completionAllClusters,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, youMustProviderClusterMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			clusterName = args[0]
			dataFetcher fetcher.Fetcher
			err         error
		)

		found, _ := GetClusterConnection(clusterName)
		if !found {
			return errors.New(UnableToFindClusterMsg + clusterName)
		}

		dataFetcher, err = GetDataFetcher(clusterName)
		if err != nil {
			return err
		}

		// retrieve cluster details first so if we are connected
		// to WLS or need authentication, this can be done first
		_, err = dataFetcher.GetClusterDetailsJSON()
		if err != nil {
			return fmt.Errorf("unabel to connect to cluster %s: %v", clusterName, err)
		}

		screen, err := tcell.NewScreen()
		if err != nil {
			return err
		}
		if err = screen.Init(); err != nil {
			return err
		}
		defer screen.Fini()

		screen.SetStyle(tcell.StyleDefault)

		// ensure we reset the screen on any panic
		defer func() {
			if r := recover(); r != nil {
				screen.Fini()
				log.Println("Panic: ", r)
			}
		}()

		exit := make(chan struct{})

		// initial update
		err = updateScreen(screen, dataFetcher, true)

		go func() {
			for {
				select {
				case <-exit:
					return
				case <-time.After(time.Duration(watchDelay) * time.Second):
					err = updateScreen(screen, dataFetcher, true) // Function to update the display
					if err != nil {
						return
					}
				}
			}
		}()

		for {
			ev := screen.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventResize:
				err = updateScreen(screen, dataFetcher, false)
				screen.Sync()
			case *tcell.EventKey:
				// Exit for 'q', ESC, or CTRL-C
				if ev.Rune() == 'q' || ev.Key() == tcell.KeyESC || ev.Key() == tcell.KeyCtrlC {
					close(exit)
					return nil
				}
			}
		}
	},
}

func updateScreen(screen tcell.Screen, dataFetcher fetcher.Fetcher, refresh bool) error {
	var (
		errorList []error
		err       error
		cluster   config.Cluster
	)

	mutex.Lock()
	defer mutex.Unlock()

	w, h := screen.Size()

	if refresh {
		lastClusterSummaryInfo, errorList = retrieveClusterSummary(dataFetcher)

		if len(errorList) > 0 {
			err = utils.GetErrors(errorList)
			_, _ = fmt.Fprint(os.Stderr, err.Error())
			return err
		}
	}

	screen.Clear()

	err = json.Unmarshal(lastClusterSummaryInfo.clusterResult, &cluster)
	if err != nil {
		return err
	}

	drawMainPanel(screen, w, h, cluster)

	err = drawContent(screen, lastClusterSummaryInfo, monitorContent)
	if err != nil {
		return err
	}

	//drawBox(screen, w/4, h/4, w/2, h/2, style, "Resizable Box", "Adjusts to screen size")
	//drawBox(screen, 1, 1, 100, 4, style, "Title", "Yellow")

	screen.Show()

	return nil
}

var clusterSummary = func(clusterSummary clusterSummaryInfo) ([]string, error) {
	var cluster config.Cluster

	err := json.Unmarshal(lastClusterSummaryInfo.clusterResult, &cluster)
	if err != nil {
		return emptyStringArray, err
	}
	results := make([]string, 4)
	results[0] = fmt.Sprintf("Cluster Name: %s", cluster.ClusterName)
	results[1] = fmt.Sprintf("Version:      %s", cluster.Version)
	results[2] = fmt.Sprintf("Members:      %v", cluster.ClusterSize)
	results[3] = fmt.Sprintf("Departures:   %v", cluster.MembersDepartureCount)

	return results, nil
}

var clusterMembersSummary = func(clusterSummary clusterSummaryInfo) ([]string, error) {
	var (
		members = config.Members{}
		//storage       = config.StorageDetails{}
	)

	err := json.Unmarshal(clusterSummary.membersResult, &members)
	if err != nil {
		return emptyStringArray, err
	}

	results := make([]string, 1)
	results[0] = fmt.Sprintf("Members: %v", len(members.Members))

	return results, nil
}
