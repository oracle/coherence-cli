/*
 * Copyright (c) 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const providePanelName = "you must provide a panel name"

var panelLayout string

// addPanelCmd represents the add profile command.
var addPanelCmd = &cobra.Command{
	Use:   "panel panel-name",
	Short: "add a panel and layout for displaying in monitor clusters.",
	Long: `The 'add panel' command adds a panel to the list of panels that can be displayed
byt the 'monitor clusters' command.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, providePanelName)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			panelName = args[0]
			err       error
			panels    = Config.Panels
		)

		// validate panel name
		if err = validateProfileName(panelName); err != nil {
			return err
		}

		if len(panelLayout) == 0 {
			return errors.New("you must provide a value for the panel")
		}

		if getPanelLayout(panelName) != "" {
			return fmt.Errorf("the panel '%s' already exists", panelName)
		}

		if panelAlreadyExists(panelName) {
			return fmt.Errorf("the panel '%s' already exists in the list of panels in monitor cluster", panelName)
		}

		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to add the panel %s with layout of [%s]? (y/n) ", panelName, panelLayout)) {
			return nil
		}

		panels = append(panels, Panel{Name: panelName, Layout: panelLayout})

		viper.Set(panelsKey, panels)
		err = WriteConfig()
		if err != nil {
			return err
		}
		cmd.Printf("panel %s was added with layout [%s]\n", panelName, panelLayout)
		return nil
	},
}

// removePanelCmd represents the remove panel command.
var removePanelCmd = &cobra.Command{
	Use:               "panel panel-name",
	Short:             "remove a panel from the list of panels",
	Long:              `The 'remove panel' command removes a panel from the list of panels.`,
	ValidArgsFunction: completionAllPanels,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, providePanelName)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			panelName = args[0]
			err       error
			panels    = Config.Panels
		)

		if getPanelLayout(panelName) == "" {
			return fmt.Errorf("a panel with the name %s does not exist", panelName)
		}

		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to remove the panel %s? (y/n) ", panelName)) {
			return nil
		}

		newPanels := make([]Panel, 0)

		// loop though the list of panels
		for _, v := range panels {
			if v.Name != panelName {
				newPanels = append(newPanels, v)
			}
		}

		viper.Set(panelsKey, newPanels)
		err = WriteConfig()
		if err != nil {
			return err
		}
		cmd.Printf("panel %s was removed\n", panelName)
		return nil
	},
}

// panelAlreadyExists returns true if the panel exists in either the default panel or
// in the list of panels
func panelAlreadyExists(panelName string) bool {
	// also check for other panels
	if validatePanels([]string{panelName}) == nil {
		return true
	}

	// validate that the panel is not in the list of known default panels in the monitor cluster command
	if _, ok := defaultMap[panelName]; ok {
		return true
	}

	return false
}

// getProfilesCmd represents the get profiles command.
var getPanelsCmd = &cobra.Command{
	Use:   "panels",
	Short: "display the panels that have been created",
	Long:  `The 'get panels' displays the panels that have been created.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.Println(FormatPanels(Config.Panels))
		return nil
	},
}

func init() {
	addPanelCmd.Flags().StringVarP(&panelLayout, "layout", "l", "", "panel layout")
	_ = addPanelCmd.MarkFlagRequired("layout")
	addPanelCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	removePanelCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
}

// getPanelLayout returns the layout for a given panel or "" if the panel doesn't exist.
func getPanelLayout(panelName string) string {
	for _, v := range Config.Panels {
		if v.Name == panelName {
			return v.Layout
		}
	}

	return ""
}
