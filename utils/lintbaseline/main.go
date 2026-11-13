// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

// Command lintbaseline checks a full GolangCI-Lint JSON report against reviewed
// style debt. It never accepts security/correctness findings or new occurrences.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"slices"
	"strings"
)

type signature struct {
	Linter string   `json:"linter"`
	Path   string   `json:"path"`
	Text   string   `json:"text"`
	Source []string `json:"source"`
}

type acceptedIssue struct {
	signature
	Reason string `json:"reason"`
}

type baseline struct {
	Version      int             `json:"version"`
	ToolVersion  string          `json:"tool_version"`
	GoVersion    string          `json:"go_version"`
	ConfigSHA256 string          `json:"config_sha256"`
	Platforms    []string        `json:"platforms"`
	Review       string          `json:"review"`
	Issues       []acceptedIssue `json:"issues"`
}

type issue struct {
	FromLinter  string
	Text        string
	SourceLines []string
	Pos         struct {
		Filename string
		Line     int
		Column   int
	}
}

func (s signature) key() string {
	encoded, _ := json.Marshal(s) // Strings and string slices cannot fail to marshal.
	return string(encoded)
}

func validateBaseline(b baseline, version, platform string, config []byte) error {
	if b.Version != 1 || b.ToolVersion != version || b.GoVersion != runtime.Version() ||
		b.ConfigSHA256 != fmt.Sprintf("%x", sha256.Sum256(config)) ||
		!slices.Contains(b.Platforms, platform) || strings.TrimSpace(b.Review) == "" || b.Issues == nil {
		return fmt.Errorf("lint baseline metadata does not match tool %s, Go %s, config or platform %s; review before updating", version, runtime.Version(), platform)
	}
	for _, accepted := range b.Issues {
		allowed := accepted.Linter == "goconst"
		if accepted.Linter == "revive" {
			rule, _, _ := strings.Cut(accepted.Text, ":")
			switch rule {
			case "var-naming", "dot-imports", "indent-error-flow", "redefines-builtin-id", "unused-parameter", "error-strings":
				allowed = true
			}
		}
		if accepted.Linter == "staticcheck" {
			code, _, _ := strings.Cut(accepted.Text, ":")
			switch code {
			case "QF1003", "QF1004", "QF1012", "S1007", "S1038", "ST1001", "ST1005", "ST1017":
				allowed = true
			}
		}
		if !allowed || accepted.Path == "" || accepted.Text == "" || len(accepted.Source) == 0 || strings.TrimSpace(accepted.Reason) == "" {
			return fmt.Errorf("invalid or non-style baseline entry: %s %s", accepted.Linter, accepted.Path)
		}
	}
	return nil
}

func unexpectedIssues(issues []issue, accepted []acceptedIssue) ([]issue, int) {
	remaining := make(map[string]int)
	for _, entry := range accepted {
		remaining[entry.key()]++
	}
	var unexpected []issue
	for _, entry := range issues {
		key := (signature{entry.FromLinter, entry.Pos.Filename, entry.Text, entry.SourceLines}).key()
		if remaining[key] == 0 {
			unexpected = append(unexpected, entry)
		} else {
			remaining[key]--
		}
	}
	stale := 0
	for _, count := range remaining {
		stale += count
	}
	return unexpected, stale
}

func checkReport(input io.Reader, baselineData, config []byte, version, platform string) error {
	var report map[string]json.RawMessage
	reportDecoder := json.NewDecoder(input)
	if err := reportDecoder.Decode(&report); err != nil {
		return err
	}
	if reportDecoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("lint report must contain exactly one JSON object")
	}
	rawIssues, exists := report["Issues"]
	if !exists || report["Report"] == nil {
		return fmt.Errorf("lint report has no Issues or Report field")
	}
	var issues []issue
	if err := json.Unmarshal(rawIssues, &issues); err != nil {
		return err
	}
	var metadata struct {
		Linters []struct {
			Name    string
			Enabled bool
		}
	}
	if err := json.Unmarshal(report["Report"], &metadata); err != nil || len(metadata.Linters) == 0 {
		return fmt.Errorf("lint report has invalid linter metadata")
	}
	var enabled []string
	for _, linter := range metadata.Linters {
		if linter.Enabled {
			enabled = append(enabled, linter.Name)
		}
	}
	slices.Sort(enabled)
	if strings.Join(enabled, ",") != "errcheck,errname,goconst,gofmt,gosec,govet,ineffassign,revive,staticcheck,typecheck,unused" {
		return fmt.Errorf("lint report enabled linters do not match the reviewed configuration")
	}
	var b baseline
	decoder := json.NewDecoder(strings.NewReader(string(baselineData)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&b); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("lint baseline must contain exactly one JSON object")
	}
	if err := validateBaseline(b, version, platform, config); err != nil {
		return err
	}
	unexpected, stale := unexpectedIssues(issues, b.Issues)
	for _, entry := range unexpected {
		fmt.Fprintf(os.Stderr, "%s:%d:%d: %s (%s)\n", entry.Pos.Filename, entry.Pos.Line, entry.Pos.Column, entry.Text, entry.FromLinter)
	}
	if len(unexpected) != 0 || stale != 0 {
		return fmt.Errorf("%d unreviewed lint issue(s), %d stale baseline entry/entries; fix new issues and remove resolved entries", len(unexpected), stale)
	}
	fmt.Printf("Lint passed: %d reviewed style issue(s), 0 unreviewed issues.\n", len(issues))
	return nil
}

func run() error {
	if len(os.Args) != 3 {
		return fmt.Errorf("usage: lintbaseline TOOL_VERSION GOOS/GOARCH < REPORT (CGO_ENABLED=0)")
	}
	if os.Getenv("CGO_ENABLED") != "0" {
		return fmt.Errorf("lint baseline requires CGO_ENABLED=0")
	}
	// Fixed repository inputs and stdin avoid accepting CLI-controlled file paths.
	baselineData, err := os.ReadFile(".golangci-baseline.json")
	if err != nil {
		return err
	}
	config, err := os.ReadFile(".golangci.yaml")
	if err != nil {
		return err
	}
	return checkReport(os.Stdin, baselineData, config, os.Args[1], os.Args[2])
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
