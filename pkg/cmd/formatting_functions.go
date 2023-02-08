/*
 * Copyright (c) 2023 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"github.com/fatih/color"
	"strconv"
	"strings"
)

var (
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgHiYellow).SprintFunc()
)

// statusHAFormatter formats a column value and makes it Red if if contains ENDANGERED
var statusHAFormatter = func(s string) string {
	if strings.Contains(s, endangered) {
		return red(s)
	}

	return s
}

// hitRateFormatter formats a column value which represents a cache hit rate.
var hitRateFormatter = func(s string) string {
	floatValue, err := strconv.ParseFloat(trimPercent(s), 32)
	if err != nil {
		return s
	}
	if floatValue > 75 {
		return s
	}
	if floatValue > 50 {
		return yellow(s)
	}

	return red(s)
}

// machineMemoryFormatting formats a column value which represents machine percent memory used.
var machineMemoryFormatting = func(s string) string {
	floatValue, err := strconv.ParseFloat(trimPercent(s), 32)
	if err != nil {
		return s
	}
	if floatValue > 25 {
		return s
	}
	if floatValue > 15 {
		return yellow(s)
	}

	return red(s)
}

// errorFormatter formats a column value which represents and error or number that needs to be highlighted.
var errorFormatter = func(s string) string {
	v, err := getInt64Value(s)
	if err != nil {
		return s
	}

	if v == 0 {
		return s
	}

	if v > 20 {
		return red(s)
	}
	return yellow(s)
}

// healthFormatter formats a column value when false will be displayed in red.
var healthFormatter = func(s string) string {
	if s == "false" {
		return red(s)
	}
	return s
}

// healthSummaryFormatter formats a column value for a health summary.
var healthSummaryFormatter = func(s string) string {
	if !strings.Contains(s, "/") {
		return s
	}
	// string contains something like "0/4"
	result := strings.Split(s, "/")
	if len(result) != 2 {
		return s
	}

	value1, err := getInt64Value(result[0])

	if err != nil {
		return s
	}

	if value1 == 0 {
		return red(s)
	}
	return yellow(s)
}

func getInt64Value(s string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

func trimPercent(s string) string {
	return strings.TrimSpace(strings.Replace(s, "%", "", 1))
}
