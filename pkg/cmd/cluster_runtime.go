/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var javaVersionPattern = regexp.MustCompile(`(?m)^(?:openjdk|java) (?:version )?"?([0-9]+)(?:\.([0-9]+))?(?:\.([0-9]+))?`)

func parseJavaVersion(output string) (int, int, error) {
	match := javaVersionPattern.FindStringSubmatch(output)
	if match == nil {
		return 0, 0, fmt.Errorf("unrecognized Java version output from java -version")
	}
	major, _ := strconv.Atoi(match[1])
	if major == 1 {
		major, _ = strconv.Atoi(match[2])
	}
	patch, _ := strconv.Atoi(match[3])
	return major, patch, nil
}

func queryJavaVersion() (int, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, javaExec, "-version").CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("unable to complete java -version (check the Java executable on PATH): %w", err)
	}
	return parseJavaVersion(string(output))
}

// These are known runtime floors, not a claim that all newer JDKs are certified.
// Unknown version families are left to their vendor's compatibility policy.
func validateCoherenceJava(version string, major, patch int) error {
	minimum := 0
	switch {
	case strings.HasPrefix(version, "22.06") || (strings.HasPrefix(version, "14.1.1.2206") || strings.HasPrefix(version, "14.1.1-2206")):
		minimum = 11
	case strings.HasPrefix(version, "14.1.2") || strings.HasPrefix(version, "15.1.1"):
		minimum = 17
	default:
		parts := strings.Split(version, ".")
		if len(parts) > 1 {
			year, _ := strconv.Atoi(parts[0])
			month, _ := strconv.Atoi(parts[1])
			if year >= 23 && year <= 25 || year == 22 && month >= 9 {
				minimum = 17
			}
		}
	}
	if minimum > 0 && major < minimum {
		return fmt.Errorf("coherence %s requires Java %d or later; java -version returned Java %d", version, minimum, major)
	}
	if (strings.HasPrefix(version, "14.1.1.2206") || strings.HasPrefix(version, "14.1.1-2206")) && major == 11 && patch < 15 {
		return fmt.Errorf("coherence %s requires Java 11.0.15 or later", version)
	}
	return nil
}

func checkManagedRuntime(version string, requiredMajor int) error {
	major, patch, err := queryJavaVersion()
	if err != nil {
		return err
	}
	if major < requiredMajor {
		return fmt.Errorf("split launcher options require Java %d or later; java -version returned Java %d", requiredMajor, major)
	}
	return validateCoherenceJava(version, major, patch)
}
