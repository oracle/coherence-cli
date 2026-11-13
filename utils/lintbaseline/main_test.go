// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"
)

const (
	testConstantLinter = "goconst"
	testExistingPath   = "example.go"
	testExistingSource = "source"
	testPriorReason    = "pre-existing"
	testVersion        = "test-version"
	testPlatform       = "darwin/arm64"
	testConfig         = "reviewed configuration"
	testSecurityLinter = "gosec"
	testLiteralText    = "literal"
)

func testBaseline() baseline {
	return baseline{Version: 1, ToolVersion: testVersion, GoVersion: runtime.Version(),
		ConfigSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(testConfig))),
		Platforms:    []string{testPlatform}, Review: "review.md", Issues: []acceptedIssue{}}
}

func TestBaselineAcceptsOnlyReviewedOccurrences(t *testing.T) {
	entry := issue{FromLinter: testConstantLinter, Text: "repeated literal", SourceLines: []string{`value := "example"`}}
	entry.Pos.Filename = "pkg/example.go"
	accepted := acceptedIssue{signature{entry.FromLinter, entry.Pos.Filename, entry.Text, entry.SourceLines}, "reviewed existing style debt"}
	if got, stale := unexpectedIssues([]issue{entry}, []acceptedIssue{accepted}); len(got) != 0 || stale != 0 {
		t.Fatalf("reviewed issue was rejected: %v, stale %d", got, stale)
	}
	if _, stale := unexpectedIssues(nil, []acceptedIssue{accepted}); stale != 1 {
		t.Fatal("resolved findings must be removed from the baseline")
	}
	entry.Pos.Line = 200 // Moving reviewed code must not require a new acceptance.
	if got, stale := unexpectedIssues([]issue{entry}, []acceptedIssue{accepted}); len(got) != 0 || stale != 0 {
		t.Fatal("line-number-only movement changed the signature")
	}
	if got, _ := unexpectedIssues([]issue{entry, entry}, []acceptedIssue{accepted}); len(got) != 1 {
		t.Fatalf("one additional identical occurrence must fail: %v", got)
	}
	if _, stale := unexpectedIssues([]issue{entry}, []acceptedIssue{accepted, accepted}); stale != 1 {
		t.Fatal("one missing duplicate must fail")
	}
	for name, change := range map[string]func(*issue){
		"text":   func(i *issue) { i.Text += " changed count" },
		"source": func(i *issue) { i.SourceLines = []string{"different source"} },
		"path":   func(i *issue) { i.Pos.Filename = "pkg/new.go" },
		"linter": func(i *issue) { i.FromLinter = testSecurityLinter },
	} {
		t.Run(name, func(t *testing.T) {
			changed := entry
			change(&changed)
			if got, stale := unexpectedIssues([]issue{changed}, []acceptedIssue{accepted}); len(got) != 1 || stale != 1 {
				t.Fatalf("new finding was accepted: %v, stale %d", got, stale)
			}
		})
	}
}

func TestBaselineRejectsSecurityAndCorrectnessDebt(t *testing.T) {
	b := testBaseline()
	for _, linter := range []string{testSecurityLinter, "errcheck", "errname", "govet", "ineffassign", "unused", "gofmt"} {
		b.Issues = []acceptedIssue{{signature{linter, testExistingPath, "finding", []string{testExistingSource}}, testPriorReason}}
		if err := validateBaseline(b, testVersion, testPlatform, []byte(testConfig)); err == nil {
			t.Fatalf("prohibited linter accepted: %s", linter)
		}
	}
	for _, text := range []string{"G703: taint", "G115: overflow", "SA1000: invalid regexp", "S1000: unreviewed simplification", "unreachable-code: unreachable branch"} {
		for _, linter := range []string{testSecurityLinter, "staticcheck", "revive"} {
			b.Issues = []acceptedIssue{{signature{linter, testExistingPath, text, []string{testExistingSource}}, testPriorReason}}
			if err := validateBaseline(b, testVersion, testPlatform, []byte(testConfig)); err == nil {
				t.Fatalf("unsafe/unreviewed rule accepted: %s %s", linter, text)
			}
		}
	}
	for _, entry := range []acceptedIssue{
		{signature{testConstantLinter, testExistingPath, testLiteralText, []string{testExistingSource}}, ""},
		{signature{testConstantLinter, testExistingPath, testLiteralText, nil}, "missing source"},
		{signature{testConstantLinter, "", testLiteralText, []string{testExistingSource}}, testPriorReason},
	} {
		b.Issues = []acceptedIssue{entry}
		if err := validateBaseline(b, testVersion, testPlatform, []byte(testConfig)); err == nil {
			t.Fatalf("incomplete acceptance allowed: %v", entry)
		}
	}
}

func TestBaselineMetadata(t *testing.T) {
	for name, change := range map[string]func(*baseline){
		"schema":   func(b *baseline) { b.Version++ },
		"tool":     func(b *baseline) { b.ToolVersion += "changed" },
		"go":       func(b *baseline) { b.GoVersion += "changed" },
		"config":   func(b *baseline) { b.ConfigSHA256 = "wrong" },
		"platform": func(b *baseline) { b.Platforms = []string{"unreviewed/architecture"} },
		"review":   func(b *baseline) { b.Review = " " },
		"issues":   func(b *baseline) { b.Issues = nil },
	} {
		t.Run(name, func(t *testing.T) {
			b := testBaseline()
			change(&b)
			if err := validateBaseline(b, testVersion, testPlatform, []byte(testConfig)); err == nil {
				t.Fatal("wrong metadata accepted")
			}
		})
	}
}

func testReport(t *testing.T, issues []issue) string {
	t.Helper()
	linters := make([]map[string]any, 0)
	for _, name := range strings.Split("errcheck,errname,goconst,gofmt,gosec,govet,ineffassign,revive,staticcheck,typecheck,unused", ",") {
		linters = append(linters, map[string]any{"Name": name, "Enabled": true})
	}
	data, err := json.Marshal(map[string]any{"Issues": issues, "Report": map[string]any{"Linters": linters}})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestReportErrorsFailClosed(t *testing.T) {
	b, err := json.Marshal(testBaseline())
	if err != nil {
		t.Fatal(err)
	}
	valid := testReport(t, nil)
	if err = checkReport(strings.NewReader(valid), b, []byte(testConfig), testVersion, testPlatform); err != nil {
		t.Fatal(err)
	}
	for _, contents := range []string{`{`, `{}`, `null`, `{"Issues":"invalid","Report":{}}`, `{"Issues":null}`, `{"Issues":null,"Report":null}`, valid + `{}`, valid + `garbage`, strings.ReplaceAll(valid, `"Enabled":true`, `"Enabled":false`)} {
		if err = checkReport(strings.NewReader(contents), b, []byte(testConfig), testVersion, testPlatform); err == nil {
			t.Fatalf("invalid report accepted: %s", contents)
		}
	}
	for _, contents := range []string{`{`, `{}`, `null`, `{"unknown":true}`, string(b) + `{}`, strings.Replace(string(b), `"version":1`, `"version":2`, 1)} {
		if err = checkReport(strings.NewReader(valid), []byte(contents), []byte(testConfig), testVersion, testPlatform); err == nil {
			t.Fatalf("invalid baseline accepted: %s", contents)
		}
	}
}

func TestReportEnforcesUnexpectedAndStaleFindings(t *testing.T) {
	entry := issue{FromLinter: testConstantLinter, Text: testLiteralText, SourceLines: []string{testExistingSource}}
	entry.Pos.Filename = testExistingPath
	b := testBaseline()
	b.Issues = []acceptedIssue{{signature{entry.FromLinter, entry.Pos.Filename, entry.Text, entry.SourceLines}, testPriorReason}}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	for name, entries := range map[string][]issue{"accepted": {entry}, "stale": nil, "duplicate": {entry, entry}, "new": {{FromLinter: testSecurityLinter, Text: "G703"}}} {
		t.Run(name, func(t *testing.T) {
			err := checkReport(strings.NewReader(testReport(t, entries)), data, []byte(testConfig), testVersion, testPlatform)
			if (err == nil) != (name == "accepted") {
				t.Fatalf("unexpected outcome: %v", err)
			}
		})
	}
}
