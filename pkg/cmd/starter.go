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
	"os"
	"path/filepath"
)

const (
	framework    = "framework"
	frameworkURL = "https://raw.githubusercontent.com/tmiddlet2666/coherence-playground/main/frameworks"
)

var (
	frameworkTypeParam string
	validFrameworks    = []string{"helidon", "springboot", "micronaut"}

	templateFiles = map[string][]string{
		"helidon": {"pom.xml",
			"src/main/resources/logging.properties",
			"src/main/resources/META-INF/beans.xml",
			"src/main/resources/META-INF/helidon/serial-config.properties",
			"src/main/resources/META-INF/microprofile-config.properties",
			"src/main/java/com/oracle/coherence/demo/frameworks/helidon/Customer.java",
			"src/main/java/com/oracle/coherence/demo/frameworks/helidon/CustomerResource.java"},
		"springboot": {"pom.xml",
			"src/main/resources/application.properties",
			"src/main/java/com/oracle/coherence/demo/frameworks/springboot/Customer.java",
			"src/main/java/com/oracle/coherence/demo/frameworks/springboot/controller/DemoController.java",
			"src/main/java/com/oracle/coherence/demo/frameworks/springboot/DemoApplication.java"},
		"micronaut": {"pom.xml",
			"src/main/resources/logback.xml",
			"src/main/resources/application.yml",
			"src/main/java/com/oracle/coherence/demo/frameworks/micronaut/Application.java",
			"src/main/java/com/oracle/coherence/demo/frameworks/micronaut/Customer.java",
			"src/main/java/com/oracle/coherence/demo/frameworks/micronaut/rest/ApplicationController.java"},
	}

	frameworkVersions = map[string]string{
		"helidon":    "4.1.6",
		"springboot": "spring-boot-starter 3.4.1, coherence-spring 4.3.0",
		"micronaut":  "micronaut-parent: 4.7.3, micronaut-coherence: 5.0.4",
	}
	frameWorkInstructions = map[string]string{
		"helidon": `
To run the Helidon starter you must have JDK21+ and maven 3.8.5+.
Change to the newly created directory and run the following to build:

    mvn clean install

To run single server:
    java -jar target/helidon.jar

To run additional server:
    java -Dmain.class=com.tangosol.net.Coherence -Dcoherence.management.http=none -Dserver.port=-1 -jar target/helidon.jar
`,
		"springboot": `
To run Spring Boot starter you must have JDK21+ and maven 3.8.5+.
Change to the newly created directory and run the following to build:
    mvn clean install

To run single server:
    java -jar target/springboot-1.0-SNAPSHOT.jar

To run additional server:
    java -Dserver.port=-1 -Dloader.main=com.tangosol.net.Coherence -Dcoherence.management.http=none -jar target/springboot-1.0-SNAPSHOT.jar
`,
		"micronaut": `
To run Micronaut starter you must have JDK21+ and maven 3.8.5+.
Change to the newly created directory and run the following to build:
    mvn clean install

To run single server:
    java -jar target/micronaut-1.0-SNAPSHOT-shaded.jar

To run additional server:
    java -Dmicronaut.main.class=com.tangosol.net.Coherence -Dcoherence.management.http=none -Dmicronaut.server.port=-1 -jar target/micronaut-1.0-SNAPSHOT-shaded.jar
`,
	}

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
will be created off the current directory with the same name as the project name.`,
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

		frameworkInfo := frameworkVersions[frameworkTypeParam]

		absolutePath, err := filepath.Abs(projectName)
		if err != nil {
			return fmt.Errorf("unable to get absolute path for directory %s: %w", projectName, err)
		}

		cmd.Println("\nCreate Starter Project")
		cmd.Printf("Project Name:       %s\n", projectName)
		cmd.Printf("Framework Type:     %s\n", frameworkTypeParam)
		cmd.Printf("Framework Versions: %s\n", frameworkInfo)
		cmd.Printf("Project Path        %s\n\n", absolutePath)

		// confirm the operation
		if !confirmOperation(cmd, "Are you sure you want to create the starter project above? (y/n) ") {
			return nil
		}

		return initProject(cmd, frameworkTypeParam, projectName, absolutePath)
	},
}

func initProject(cmd *cobra.Command, frameworkType string, projectName string, absolutePath string) error {
	var (
		fileList []string
		exists   bool
		err      error
	)

	if fileList, exists = templateFiles[frameworkType]; !exists {
		return fmt.Errorf("unable to find files for framework %s", frameworkType)
	}

	// create the directory
	if err = utils.EnsureDirectory(projectName); err != nil {
		return err
	}

	err = saveFiles(cmd, fileList, projectName, frameworkType)
	if err != nil {
		return err
	}

	// output instructions for the framework
	cmd.Printf("\nYour %s project has been saved to %s\n", frameworkType, absolutePath)

	// get framework instructions
	instructions := frameWorkInstructions[frameworkType]

	cmd.Printf("\nPlease see the file %s/readme.txt for instructions\n\n", projectName)
	return writeContentToFile(projectName, "readme.txt", instructions+commonInstructions)
}

func saveFiles(cmd *cobra.Command, fileList []string, baseDir string, frameworkType string) error {
	cmd.Println("Downloading template for", frameworkType)

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
