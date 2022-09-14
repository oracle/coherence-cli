/*
 * Copyright (c) 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"regexp"
)

const provideProfileName = "you must provide a profile name"

var (
	isValid      = regexp.MustCompile(`^[A-Za-z\d-]+$`).MatchString
	profileValue string
)

// setProfileCmd represents the set profile command
var setProfileCmd = &cobra.Command{
	Use:   "profile profile-name",
	Short: "set a profile value for creating and starting clusters",
	Long: `The 'set profile' command sets a profile value for creating and starting clusters.
Profiles can be specified using the '-P' option when creating and starting clusters. They
contain property values to be set prior to the final class and must be surrounded by quotes 
and be space delimited. If you set a profile that exists, it will be overwritten.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideProfileName)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			profileName = args[0]
			err         error
			found       = false
			profiles    = Config.Profiles
		)

		// validate profile name
		if err = validateProfileName(profileName); err != nil {
			return err
		}

		if len(profileValue) == 0 {
			return errors.New("you must provide a value for the profile")
		}

		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to set the profile %s to a value of [%s]? (y/n) ", profileName, profileValue)) {
			return nil
		}

		// loop though the list of profiles
		for i, v := range profiles {
			if v.Name == profileName {
				// update
				profiles[i].Value = profileValue
				found = true
				break
			}
		}

		if !found {
			// must be new so append
			profiles = append(profiles, ProfileValue{Name: profileName, Value: profileValue})
		}

		viper.Set(profilesKey, profiles)
		err = WriteConfig()
		if err != nil {
			return err
		}
		cmd.Printf("profile %s was set to value [%s]\n", profileName, profileValue)
		return nil
	},
}

// removeProfileCmd represents the remove profile command
var removeProfileCmd = &cobra.Command{
	Use:   "profile profile-name",
	Short: "removes a profile value from the list of profiles",
	Long:  `The 'remove profile' command removes a profile value from the list of profiles.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, provideProfileName)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			profileName = args[0]
			err         error
			profiles    = Config.Profiles
		)

		if getProfileValue(profileName) == "" {
			return fmt.Errorf("a profile with the name %s does not exist", profileName)
		}

		// confirm the operation
		if !confirmOperation(cmd, fmt.Sprintf("Are you sure you want to remove the profile %s? (y/n) ", profileName)) {
			return nil
		}

		newProfiles := make([]ProfileValue, 0)

		// loop though the list of profiles
		for _, v := range profiles {
			if v.Name != profileName {
				newProfiles = append(newProfiles, v)
			}
		}

		viper.Set(profilesKey, newProfiles)
		err = WriteConfig()
		if err != nil {
			return err
		}
		cmd.Printf("profile %s was removed\n", profileName)
		return nil
	},
}

// getProfilesCmd represents the get profiles command
var getProfilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "displays the profiles that have been created",
	Long:  `The 'get profiles' displays the profiles that have been created.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println(FormatProfiles(Config.Profiles))
		return nil
	},
}

func init() {
	setProfileCmd.Flags().StringVarP(&profileValue, "value", "v", "", "profile value to set")
	_ = setProfileCmd.MarkFlagRequired("value")
	setProfileCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)

	removeProfileCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
}

func validateProfileName(profileName string) error {
	if !isValid(profileName) {
		return fmt.Errorf("profile name %s must only contain letters, numbers and '", profileName)
	}

	return nil
}

// getProfileValue returns the value for a given profile or "" if the profile doesn't exist
func getProfileValue(profileName string) string {
	for _, v := range Config.Profiles {
		if v.Name == profileName {
			return v.Value
		}
	}

	return ""
}
