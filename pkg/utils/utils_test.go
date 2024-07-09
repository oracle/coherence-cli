/*
 * Copyright (c) 2021, 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package utils

import (
	"encoding/json"
	"fmt"
	"github.com/onsi/gomega"
	"github.com/oracle/coherence-cli/pkg/config"
	"os"
	"path/filepath"
	"testing"
)

func TestGetUniqueValues(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	g.Expect(len(GetUniqueValues([]string{"A", "A", "B", "C"}))).To(gomega.Equal(3))
}

func TestSliceContains(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	g.Expect(SliceContains([]string{"A", "B", "C"}, "A")).To(gomega.Equal(true))
	g.Expect(SliceContains([]string{"A", "B", "C"}, "D")).To(gomega.Equal(false))
}

func TestGetSliceIndex(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	g.Expect(GetSliceIndex([]string{"A", "B", "C"}, "A")).To(gomega.Equal(0))
	g.Expect(GetSliceIndex([]string{"A", "B", "C"}, "D")).To(gomega.Equal(-1))
	g.Expect(GetSliceIndex([]string{"A", "B", "C"}, "B")).To(gomega.Equal(1))
	g.Expect(GetSliceIndex([]string{"A", "B", "C"}, "C")).To(gomega.Equal(2))
}

func TestCombineByteArraysForJSON(t *testing.T) {
	var (
		g      = gomega.NewGomegaWithT(t)
		b1     = []byte("abc")
		b2     = []byte("123")
		b3     = []byte("456")
		b4     = make([]byte, 0)
		err    error
		result []byte
	)

	result, err = CombineByteArraysForJSON([][]byte{b1, b2}, []string{"a", "b"})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(string(result)).To(gomega.Equal("{\"a\":abc,\"b\":123}"))

	result, err = CombineByteArraysForJSON([][]byte{b1, b2, b3}, []string{"a", "b", "c"})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(string(result)).To(gomega.Equal("{\"a\":abc,\"b\":123,\"c\":456}"))

	result, err = CombineByteArraysForJSON([][]byte{b2, b3, b4}, []string{"a", "b", "c"})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(string(result)).To(gomega.Equal("{\"a\":123,\"b\":456,\"c\":{}}"))

	result, err = CombineByteArraysForJSON([][]byte{b4, b1}, []string{"a", "b"})
	g.Expect(err).To(gomega.BeNil())
	g.Expect(string(result)).To(gomega.Equal("{\"a\":{},\"b\":abc}"))
}

func TestJsonPath(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	jsonString1 := `{  
  "event": {  
    "name": "agent",  
    "data": {  
      "name": "James Bond"  
    }  
  }  
}`

	jsonString2 := `{   
	"customers": [  
		{
		"id": 1,
		"name": "Tim",
		"address": "123 James Street, Perth",
		"balance": 1000
		},		
		{
		"id": 2,
		"name": "John",
		"address": "1233 William Stress, West Perth",
		"balance": 10
		}
   		]
	}`

	assertJSONParse(g, jsonString1, "$.event.data.name", "[\"James Bond\"]")

	assertJSONParse(g, jsonString2, "$.customers[0].name", "[\"Tim\"]")

	assertJSONParse(g, jsonString2, "$.customers[?(@.balance <= 10)].name", "[\"John\"]")
}

// assertJSONParse asserts that the jsonpath expression works
func assertJSONParse(g *gomega.WithT, jsonString, jsonPath, expected string) {
	jsonData := []byte(jsonString)
	data := getJSON(g, jsonData)
	result, err := ProcessJSONPath(data, jsonPath)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(string(result)).To(gomega.Equal(expected))
}

// getJSON returns a representation of the Json data as an interface{}
func getJSON(g *gomega.WithT, data []byte) interface{} {
	var result interface{}
	err := json.Unmarshal(data, &result)
	g.Expect(err).To(gomega.BeNil())
	return result
}

func TestSanitizeSnapshotName(t *testing.T) {
	var (
		g        = gomega.NewGomegaWithT(t)
		expected = "test-tim"
	)
	g.Expect(SanitizeSnapshotName("abc123")).To(gomega.Equal("abc123"))
	g.Expect(SanitizeSnapshotName("abc_123")).To(gomega.Equal("abc_123"))
	g.Expect(SanitizeSnapshotName("abc-123")).To(gomega.Equal("abc-123"))
	g.Expect(SanitizeSnapshotName("abc123~")).To(gomega.Equal("abc123-"))
	g.Expect(SanitizeSnapshotName("abc123 ")).To(gomega.Equal("abc123-"))
	g.Expect(SanitizeSnapshotName("!@#$%^")).To(gomega.Equal("------"))
	g.Expect(SanitizeSnapshotName("test/tim")).To(gomega.Equal(expected))
	g.Expect(SanitizeSnapshotName("test\\tim")).To(gomega.Equal(expected))
	g.Expect(SanitizeSnapshotName("test.tim")).To(gomega.Equal(expected))
	g.Expect(SanitizeSnapshotName("c:test.tim")).To(gomega.Equal("c-test-tim"))
}

func TestGetStorageMap(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	testCase1 := config.StorageDetails{Details: []config.StorageDetail{
		{NodeID: "1", OwnedPartitionsPrimary: 1},
		{NodeID: "2", OwnedPartitionsPrimary: 2},
		{NodeID: "3", OwnedPartitionsPrimary: 0},
	}}

	result := GetStorageMap(testCase1)
	g.Expect(len(result)).To(gomega.Equal(3))
	g.Expect(result[1]).To(gomega.Equal(true))
	g.Expect(result[2]).To(gomega.Equal(true))
	g.Expect(result[3]).To(gomega.Equal(false))

	g.Expect(IsStorageEnabled(1, result)).To(gomega.Equal(true))
	g.Expect(IsStorageEnabled(2, result)).To(gomega.Equal(true))
	g.Expect(IsStorageEnabled(3, result)).To(gomega.Equal(false))

	// test that a single storage count > 0 should make the node storage enabled
	testCase2 := config.StorageDetails{Details: []config.StorageDetail{
		{NodeID: "1", OwnedPartitionsPrimary: 1},
		{NodeID: "1", OwnedPartitionsPrimary: 0},
		{NodeID: "2", OwnedPartitionsPrimary: 0},
	}}

	result = GetStorageMap(testCase2)
	g.Expect(len(result)).To(gomega.Equal(2))
	g.Expect(result[1]).To(gomega.Equal(true))
	g.Expect(result[2]).To(gomega.Equal(false))

	// test that a single storage count > 0 should make the node storage enabled when it is second
	testCase3 := config.StorageDetails{Details: []config.StorageDetail{
		{NodeID: "1", OwnedPartitionsPrimary: 0},
		{NodeID: "1", OwnedPartitionsPrimary: 1},
		{NodeID: "2", OwnedPartitionsPrimary: 0},
	}}

	result = GetStorageMap(testCase3)
	g.Expect(len(result)).To(gomega.Equal(2))
	g.Expect(result[1]).To(gomega.Equal(true))
	g.Expect(result[2]).To(gomega.Equal(false))

	// test that a single storage count > 0 should make the node storage enabled when it is second
	testCase4 := config.StorageDetails{Details: []config.StorageDetail{
		{NodeID: "1", OwnedPartitionsPrimary: 0},
		{NodeID: "1", OwnedPartitionsPrimary: 1},
		{NodeID: "2", OwnedPartitionsPrimary: 0},
		{NodeID: "2", OwnedPartitionsPrimary: 1},
		{NodeID: "3", OwnedPartitionsPrimary: -1},
	}}

	result = GetStorageMap(testCase4)
	g.Expect(len(result)).To(gomega.Equal(3))
	g.Expect(result[1]).To(gomega.Equal(true))
	g.Expect(result[2]).To(gomega.Equal(true))
	g.Expect(result[3]).To(gomega.Equal(false))
}

func TestPortValidation(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	g.Expect(ValidatePort(100)).Should(gomega.Equal(ErrPort))
	g.Expect(ValidatePort(1023)).Should(gomega.Equal(ErrPort))
	g.Expect(ValidatePort(65536)).Should(gomega.Equal(ErrPort))
	g.Expect(ValidatePort(-1)).Should(gomega.Equal(ErrPort))
	g.Expect(ValidatePort(1024)).ShouldNot(gomega.HaveOccurred())
	g.Expect(ValidatePort(65535)).ShouldNot(gomega.HaveOccurred())
	g.Expect(ValidatePort(12345)).ShouldNot(gomega.HaveOccurred())
}

func TestGetStartupDelayInMillis(t *testing.T) {
	var (
		g      = gomega.NewGomegaWithT(t)
		result int64
		err    error
	)

	result, err = GetStartupDelayInMillis("0ms")
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	g.Expect(result).Should(gomega.Equal(int64(0)))

	result, err = GetStartupDelayInMillis("123")
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	g.Expect(result).Should(gomega.Equal(int64(123)))

	result, err = GetStartupDelayInMillis("10ms")
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	g.Expect(result).Should(gomega.Equal(int64(10)))

	result, err = GetStartupDelayInMillis("1s")
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	g.Expect(result).Should(gomega.Equal(int64(1000)))

	result, err = GetStartupDelayInMillis("23s")
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	g.Expect(result).Should(gomega.Equal(int64(23000)))

	_, err = GetStartupDelayInMillis("233123123123123123s")
	g.Expect(err).To(gomega.HaveOccurred())

	_, err = GetStartupDelayInMillis("23xs")
	g.Expect(err).To(gomega.HaveOccurred())

	_, err = GetStartupDelayInMillis("s")
	g.Expect(err).To(gomega.HaveOccurred())

	_, err = GetStartupDelayInMillis("ms")
	g.Expect(err).To(gomega.HaveOccurred())
}

func TestNoWritableHomeDir(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	dir, err := os.MkdirTemp("", "")
	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

	defer os.RemoveAll(dir)

	// change the directory to be not readable
	err = os.Chmod(dir, 0555)

	g.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

	// try and ensure a file
	err = EnsureDirectory(filepath.Join(dir, "my-file"))
	g.Expect(err).Should(gomega.HaveOccurred())

	// required
	fmt.Println("")
}
