/*
 * Copyright (c) 2025 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"fmt"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

const (
	framework         = "framework"
	frameworkURL      = "https://raw.githubusercontent.com/oracle/coherence-cli/refs/heads/main/templates"
	frameworkTypesURL = frameworkURL + "/templates.yaml"
)

// FrameworkTemplate contains the contents read from the coherence-cli repository.
type FrameworkTemplate struct {
	Name             string   `yaml:"name"`
	FrameworkVersion string   `yaml:"frameworkVersion"`
	Instructions     string   `yaml:"instructions"`
	Files            []string `yaml:"files"`
}

type Templates struct {
	Templates []FrameworkTemplate `yaml:"templates"`
}

var (
	frameworkTypeParam string
	validFrameworks    = []string{"helidon", "springboot", "micronaut"}

	commonInstructions = `
Add a customer:
    curl -X POST -H "Content-Type: application/json" -d '{"id": 1, "name": "Tim", "balance": 1000}' http://localhost:8080/api/customers

Get a customer:
    curl -s http://localhost:8080/api/customers/1

Get all customers:
    curl -s http://localhost:8080/api/customers

Delete a customer:
    curl -X DELETE http://localhost:8080/api/customers/1
`
)

// createStarterCmd represents the create starter command.
var createStarterCmd = &cobra.Command{
	Use:   "starter project-name",
	Short: "creates a starter project for Coherence",
	Long: `The 'create starter' command creates a starter Maven project to use Coherence 
with various frameworks including Helidon, Spring Boot and Micronaut. A directory
will be created off the current directory with the same name as the project name.
NOTE: This is an experimental feature only and the projects created are not fully
completed applications. They are a demo/example of how to do basic integration with
each of the frameworks.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide a project name")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			projectName = args[0]
			err         error
		)

		// the project name will be used as the directory off the current directory, validate it
		if utils.SanitizeSnapshotName(projectName) != projectName {
			return fmt.Errorf("invalid project name: %s", projectName)
		}

		// validate the framework
		if !utils.SliceContains(validFrameworks, frameworkTypeParam) {
			return fmt.Errorf("framework must be one of %v", validFrameworks)
		}

		// check the directory does not exist
		if utils.DirectoryExists(projectName) {
			return fmt.Errorf("the directory %s already exists", projectName)
		}

		cmd.Printf("checking for availability of template %v...\n", frameworkTypeParam)

		// load the template files
		templates, err := loadTemplateFiles()
		if err != nil {
			return err
		}

		// get the template for the framework
		template := getTemplate(templates, frameworkTypeParam)

		if template == nil {
			return fmt.Errorf("unable to find files for framework %s", frameworkTypeParam)
		}

		absolutePath, err := filepath.Abs(projectName)
		if err != nil {
			return fmt.Errorf("unable to get absolute path for directory %s: %w", projectName, err)
		}

		cmd.Println("\nCreate Starter Project")
		cmd.Printf("Project Name:       %s\n", projectName)
		cmd.Printf("Framework Type:     %s\n", frameworkTypeParam)
		cmd.Printf("Framework Versions: %s\n", template.FrameworkVersion)
		cmd.Printf("Project Path        %s\n\n", absolutePath)

		// confirm the operation
		if !confirmOperation(cmd, "Are you sure you want to create the starter project above? (y/n) ") {
			return nil
		}

		return initProject(cmd, template, projectName, absolutePath)
	},
}

func initProject(cmd *cobra.Command, template *FrameworkTemplate, projectName string, absolutePath string) error {
	var err error

	// create the directory
	if err = utils.EnsureDirectory(projectName); err != nil {
		return err
	}

	err = saveFiles(template.Files, projectName, template.Name)
	if err != nil {
		return err
	}

	// output instructions for the framework
	cmd.Printf("\nYour %s template project has been saved to %s\n", template.Name, absolutePath)

	// get framework instructions
	instructions := template.Instructions

	cmd.Printf("\nPlease see the file %s/readme.txt for instructions\n\n", projectName)
	return writeContentToFile(projectName, "readme.txt", instructions+commonInstructions)
}

func loadTemplateFiles() (*Templates, error) {
	var templates Templates

	response, err := GetURLContents(frameworkTypesURL)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(response, &templates)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal %v, %v", frameworkTypesURL, err)
	}

	return &templates, nil
}

func getTemplate(templates *Templates, framework string) *FrameworkTemplate {
	for _, template := range templates.Templates {
		if template.Name == framework {
			return &template
		}
	}
	return nil
}

func saveFiles(fileList []string, baseDir string, frameworkType string) error {
	for _, file := range fileList {
		url := fmt.Sprintf("%s/%s/%s", frameworkURL, frameworkType, file)

		// Get the file contents
		response, err := GetURLContents(url)
		if err != nil {
			return fmt.Errorf("error downloading file %s: %w", file, err)
		}

		// Construct the full destination path
		destPath := filepath.Join(baseDir, file)

		destDir := filepath.Dir(destPath)
		if err = os.MkdirAll(destDir, os.ModePerm); err != nil {
			return fmt.Errorf("error creating directory %s: %w", destDir, err)
		}

		// Write the file contents to the destination
		if err = os.WriteFile(destPath, response, 0600); err != nil {
			return fmt.Errorf("error writing file %s: %w", destPath, err)
		}
	}

	return nil
}

func writeContentToFile(baseDir, fileName, content string) error {
	filePath := filepath.Join(baseDir, fileName)

	// Write the content to the file
	err := os.WriteFile(filePath, []byte(content), 0600)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filePath, err)
	}

	return nil
}

func init() {
	createStarterCmd.Flags().StringVarP(&frameworkTypeParam, framework, "f", "", "the framework to create for: helidon, springboot or micronaut")
	_ = createStarterCmd.MarkFlagRequired(framework)
	createStarterCmd.Flags().BoolVarP(&automaticallyConfirm, "yes", "y", false, confirmOptionMessage)
}
