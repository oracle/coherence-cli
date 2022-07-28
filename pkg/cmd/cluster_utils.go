/*
 * Copyright (c) 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"fmt"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"os/exec"
)

// checkCreateRequirements validates that all the necessary requirements are fulfilled
// for creating a cluster. This includes mvn and java executables. Nil is returned to
// indicate everything is ok, otherwise an error is returned
func checkCreateRequirements() error {
	var (
		javaExec = getJavaExec()
		mvnExec  = getMvnExec()
		err      error
	)

	processJava := exec.Command(javaExec, "-v")
	if err = processJava.Start(); err != nil {
		return utils.GetError(fmt.Sprintf("unable to get Java version using %s -v: %v", javaExec, processJava), err)
	}

	processMaven := exec.Command(mvnExec, "-v")
	if err = processMaven.Start(); err != nil {
		return utils.GetError(fmt.Sprintf("unable to get Maven version using %s -v, %v", mvnExec, processMaven), err)
	}

	return nil
}

func getJavaExec() string {
	if isWindows() {
		return "java.exe"
	} else {
		return "java"
	}
}

func getMvnExec() string {
	if isWindows() {
		return "mvn.exe"
	} else {
		return "mvn"
	}
}

// getCoherenceDependencies runs the mvn dependency:get command to download coherence.jar and coherence-json.jar
// which are the minimum requirements to create a cluster with management over rest enabled
func getCoherenceDependencies(cmd *cobra.Command, coherenceVersion string) error {
	var (
		mvnExec = getMvnExec()
		err     error
	)

	if err = runCommand(cmd, mvnExec, getDependencyArgs("coherence", coherenceVersion)); err != nil {
		return nil
	}

	if err = runCommand(cmd, mvnExec, getDependencyArgs("coherence-json", coherenceVersion)); err != nil {
		return nil
	}

	return nil
}

func getDependencyArgs(artefact, version string) []string {
	return []string{"-DgroupId=com.oracle.coherence.ce", "-DartifactId=" + artefact, "-Dversion=" + version, "dependency:get"}
}

func runCommand(cmd *cobra.Command, command string, arguments []string) error {
	process := exec.Command(command, arguments...)
	process.Stdout = cmd.OutOrStdout()
	process.Stdin = cmd.InOrStdin()
	process.Stderr = cmd.ErrOrStderr()
	if err := process.Start(); err != nil {
		return utils.GetError(fmt.Sprintf("unable to start process %s, %v", command, process), err)
	}

	return nil
}
