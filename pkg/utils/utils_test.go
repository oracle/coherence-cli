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

const (
	Ownership71 = `There are currently no pending or scheduled distributions for this service.
<br/>*** Member:  1 total=5 (primary=3, backup=2)<br/>Primary[]#003: 000, 001, 002
<br/>Backup[1]#002: 003, 004<br/><br/>*** Member:  2 total=5 (primary=2, backup=3)
<br/>Primary[]#002: 005, 006<br/>Backup[1]#003: 000, 001, 002<br/><br/>*** Member:  3 total=4
(primary=2, backup=2)<br/>Primary[]#002: 003, 004<br/>Backup[1]#002: 005, 006<br/><br/>
*** Orphans:<br/>Primary[]#000<br/>Backup[1]#000<br/>`

	Ownership192 = `There are currently no pending or scheduled distributions for this service.
<br/>*** Member:  1 total=9 (primary=3, backup=6)<br/>Primary[]#003: 000, 008, 012
<br/>Backup[1]#003: 013, 015, 017<br/>Backup[2]#003: 002, 004, 007<br/><br/>
*** Member:  2 total=9 (primary=3, backup=6)<br/>Primary[]#003: 005, 009, 013
<br/>Backup[1]#002: 006, 008<br/>Backup[2]#004: 010, 012, 015, 017<br/><br/>
*** Member:  3 total=9 (primary=3, backup=6)<br/>Primary[]#003: 001, 002, 004
<br/>Backup[1]#006: 000, 003, 005, 010, 011, 016<br/>Backup[2]#000<br/><br/>
*** Member:  4 total=9 (primary=3, backup=6)<br/>Primary[]#003: 006, 010, 014
<br/>Backup[1]#001: 018<br/>Backup[2]#005: 000, 003, 005, 008, 013<br/><br/>
*** Member:  5 total=10 (primary=3, backup=7)<br/>Primary[]#003: 003, 007, 011
<br/>Backup[1]#003: 009, 012, 014<br/>Backup[2]#004: 001, 006, 016, 018<br/><br/>
*** Member:  6 total=11 (primary=4, backup=7)<br/>Primary[]#004: 015, 016, 017, 018
<br/>Backup[1]#004: 001, 002, 004, 007<br/>Backup[2]#003: 009, 011, 014<br/><br/>
*** Orphans:<br/>Primary[]#000<br/>Backup[1]#000<br/>Backup[2]#000<br/>`

	Ownership71Safe = `There are currently no pending or scheduled distributions for this service.
<br/>*** Member:  1 total=5 (primary=3, backup=2)<br/>Primary[]#003:+000,+001,+002
<br/>Backup[1]#002:+003,+004<br/><br/>*** Member:  2 total=5 (primary=2, backup=3)
<br/>Primary[]#002:+005,+006<br/>Backup[1]#003:+000,+001,+002<br/><br/>*** Member:  3 total=4
(primary=2, backup=2)<br/>Primary[]#002:+003,+004<br/>Backup[1]#002:+005,+006<br/><br/>
*** Orphans:<br/>Primary[]#000<br/>Backup[1]#000<br/>`
)

func TestParsePartitions(t *testing.T) {
	var (
		g = gomega.NewGomegaWithT(t)
	)
	partitions := extractPartitions("Backup[1]#008: ")
	g.Expect(len(partitions)).To(gomega.Equal(0))

	partitions = extractPartitions("Backup[1]#000")
	g.Expect(len(partitions)).To(gomega.Equal(0))

	partitions = extractPartitions("Backup[1]#008: 333, 444, 5555")
	g.Expect(len(partitions)).To(gomega.Equal(3))

	partitions = extractPartitions("Primary[]#006: 031, 032, 033, 034, 035, 036")
	g.Expect(len(partitions)).To(gomega.Equal(6))

	partitions = extractPartitions("Primary[]#006:+031,+032,+033,+034,+035,+036")
	g.Expect(len(partitions)).To(gomega.Equal(6))
}

func TestExtractBackup(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	g.Expect(extractBackup("Rubbish")).To(gomega.Equal(-1))
	g.Expect(extractBackup("Backup[1]#008:")).To(gomega.Equal(1))
	g.Expect(extractBackup("Backup[1]#008: 333, 444, 5555")).To(gomega.Equal(1))
	g.Expect(extractBackup("Backup[2]#008: 333, 444, 5555")).To(gomega.Equal(2))
	g.Expect(extractBackup("Backup[2]#008:+333,+444,+5555")).To(gomega.Equal(2))
}

func TestRemovePrefix(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	g.Expect(removePrefix("Rubbish")).To(gomega.Equal(""))
	g.Expect(removePrefix("Backup[1]#008: ")).To(gomega.Equal(""))
	g.Expect(removePrefix("Backup[1]#000")).To(gomega.Equal(""))
	g.Expect(removePrefix("Backup[1]#008: 333, 444, 5555")).To(gomega.Equal("333, 444, 5555"))
	g.Expect(removePrefix("Backup[2]#008: 333, 444")).To(gomega.Equal("333, 444"))
	g.Expect(removePrefix("Primary[]#006: 031, 032, 033, 034, 035, 036")).To(gomega.Equal("031, 032, 033, 034, 035, 036"))
}

func TestFormatPartitions(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	g.Expect(FormatPartitions([]int{0, 1, 3, 4, 5, 10})).To(gomega.Equal("0..1, 3..5, 10"))
	g.Expect(FormatPartitions([]int{0, 1, 2, 3, 4, 5})).To(gomega.Equal("0..5"))
	g.Expect(FormatPartitions([]int{0, 3, 5, 7})).To(gomega.Equal("0, 3, 5, 7"))
	g.Expect(FormatPartitions([]int{10, 9, 8, 22, 21})).To(gomega.Equal("8..10, 21..22"))
}

func Test7Partitions1Backup(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Parse the partition ownership from Ownership_7_1
	mapOwnership, err := ParsePartitionOwnership(encodeOwnership(Ownership71))
	g.Expect(err).ToNot(gomega.HaveOccurred())

	for _, v := range mapOwnership {
		g.Expect(v.PartitionMap).To(gomega.Not(gomega.BeNil()))
	}

	// Parse the partition ownership from Ownership_7_1
	mapOwnership, err = ParsePartitionOwnership(encodeOwnership(Ownership71Safe))
	g.Expect(err).ToNot(gomega.HaveOccurred())

	for _, v := range mapOwnership {
		g.Expect(v.PartitionMap).To(gomega.Not(gomega.BeNil()))
	}

	// Assert that the map size is correct
	g.Expect(len(mapOwnership)).To(gomega.Equal(4))
}

func Test19Partitions2Backup(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Parse the partition ownership from Ownership_19_2
	mapOwnership, err := ParsePartitionOwnership(encodeOwnership(Ownership192))
	g.Expect(err).ToNot(gomega.HaveOccurred(), "Expected no error during parsing")

	// Print the map for visualization (optional)
	for k, v := range mapOwnership {
		fmt.Printf("k=%d, v=%+v\n", k, v)
	}

	// Assert that the map size is correct
	g.Expect(len(mapOwnership)).To(gomega.Equal(7), "Expected map size to be 7")
}

func encodeOwnership(sText string) string {
	return fmt.Sprintf("{\"ownership\":\"%s\"}", sText)
}
