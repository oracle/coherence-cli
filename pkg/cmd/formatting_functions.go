/*
 * Copyright (c) 2023, 2024 Oracle and/or its affiliates.
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

const stopped = "stopped"

// statusHAFormatter formats a column value and makes it Red if contains ENDANGERED.
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

// errorFormatter formats a column value which represents an error or number that needs to be highlighted.
var errorFormatter = func(s string) string {
	v, err := getInt64Value(s)
	if err != nil || v == 0 {
		return s
	}

	if v > 20 {
		return red(s)
	}
	return yellow(s)
}

// endangeredPartitionsFormatter formats a column value which represents the number of endangered partitions.
var endangeredPartitionsFormatter = func(s string) string {
	v, err := getInt64Value(s)
	if err != nil || v == 0 || v == -1 {
		return s
	}

	return red(s)
}

// vulnerablePartitionsFormatter formats a column value which represents the number of vulnerable or unbalanced partitions.
var vulnerablePartitionsFormatter = func(s string) string {
	v, err := getInt64Value(s)
	if err != nil || v == 0 || v == -1 {
		return s
	}

	return yellow(s)
}

// packetFormatter formats a column value which represents packages where higher numbers need to be highlighted.
var packetFormatter = func(s string) string {
	v, err := getInt64Value(s)
	if err != nil {
		return s
	}

	if v == 0 {
		return s
	}

	if v > 10 {
		return red(s)
	}
	return yellow(s)
}

// healthFormatter formats a column value when false will be displayed in red.
var healthFormatter = func(s string) string {
	if s == stringFalse {
		return red(s)
	}
	return s
}

// reporterFormatter formats a column value when "Stopped" will be displayed in red.
var reporterFormatter = func(s string) string {
	if strings.Contains(strings.ToLower(s), stopped) {
		return red(s)
	}
	return s
}

// trueBoolFormatter formats a column value when true will be displayed in red.
var trueBoolFormatter = func(s string) string {
	if s == stringTrue {
		return red(s)
	}
	return s
}

// yesBoolFormatter formats a column value when yes will be displayed in red.
var yesBoolFormatter = func(s string) string {
	if strings.Contains(s, "yes") {
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

// healthSummaryFormatter formats a column value for federation state.
var federationStateFormatter = func(s string) string {
	lowerCase := strings.ToLower(s)
	if strings.Contains(lowerCase, "error") {
		return red(s)
	}
	if strings.Contains(lowerCase, "paused") || strings.Contains(lowerCase, stopped) || strings.Contains(lowerCase, "connect_wait") {
		return yellow(s)
	}

	return s
}

// networkStatsFormatter formats a column value representing publisher or receiver rates.
var networkStatsFormatter = func(s string) string {
	floatValue, err := strconv.ParseFloat(trimPercent(s), 32)
	if err != nil {
		return s
	}

	if floatValue > 0.95 {
		return s
	}
	if floatValue >= 0.9 {
		return yellow(s)
	}
	return red(s)
}

func getInt64Value(s string) (int64, error) {
	return strconv.ParseInt(strings.ReplaceAll(strings.TrimSpace(s), ",", ""), 10, 64)
}

func trimPercent(s string) string {
	return strings.TrimSpace(strings.Replace(s, "%", "", 1))
}
