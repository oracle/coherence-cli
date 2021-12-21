/*
 * Copyright (c) 2021, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package utils

import (
	"encoding/json"
	. "github.com/onsi/gomega"
	"testing"
)

func TestGetUniqueValues(t *testing.T) {
	g := NewGomegaWithT(t)

	g.Expect(len(GetUniqueValues([]string{"A", "A", "B", "C"}))).To(Equal(3))
}

func TestSliceContains(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(SliceContains([]string{"A", "B", "C"}, "A")).To(Equal(true))
	g.Expect(SliceContains([]string{"A", "B", "C"}, "D")).To(Equal(false))
}

func TestGetSliceIndex(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(GetSliceIndex([]string{"A", "B", "C"}, "A")).To(Equal(0))
	g.Expect(GetSliceIndex([]string{"A", "B", "C"}, "D")).To(Equal(-1))
	g.Expect(GetSliceIndex([]string{"A", "B", "C"}, "B")).To(Equal(1))
	g.Expect(GetSliceIndex([]string{"A", "B", "C"}, "C")).To(Equal(2))
}

func TestCombineByteArraysForJSON(t *testing.T) {
	var (
		g      = NewGomegaWithT(t)
		b1     = []byte("abc")
		b2     = []byte("123")
		b3     = []byte("456")
		b4     = make([]byte, 0)
		err    error
		result []byte
	)

	result, err = CombineByteArraysForJSON([][]byte{b1, b2}, []string{"a", "b"})
	g.Expect(err).To(BeNil())
	g.Expect(string(result)).To(Equal("{\"a\":abc,\"b\":123}"))

	result, err = CombineByteArraysForJSON([][]byte{b1, b2, b3}, []string{"a", "b", "c"})
	g.Expect(err).To(BeNil())
	g.Expect(string(result)).To(Equal("{\"a\":abc,\"b\":123,\"c\":456}"))

	result, err = CombineByteArraysForJSON([][]byte{b2, b3, b4}, []string{"a", "b", "c"})
	g.Expect(err).To(BeNil())
	g.Expect(string(result)).To(Equal("{\"a\":123,\"b\":456,\"c\":{}}"))

	result, err = CombineByteArraysForJSON([][]byte{b4, b1}, []string{"a", "b"})
	g.Expect(err).To(BeNil())
	g.Expect(string(result)).To(Equal("{\"a\":{},\"b\":abc}"))
}

func TestJsonPath(t *testing.T) {
	g := NewGomegaWithT(t)

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
func assertJSONParse(g *WithT, jsonString, jsonPath, expected string) {
	jsonData := []byte(jsonString)
	data := getJSON(g, jsonData)
	result, err := ProcessJSONPath(data, jsonPath)
	g.Expect(err).To(BeNil())
	g.Expect(string(result)).To(Equal(expected))
}

// getJSON returns a representation of the Json data as an interface{}
func getJSON(g *WithT, data []byte) interface{} {
	var result interface{}
	err := json.Unmarshal(data, &result)
	g.Expect(err).To(BeNil())
	return result
}

func TestSanitizeSnapshotName(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(SanitizeSnapshotName("abc123")).To(Equal("abc123"))
	g.Expect(SanitizeSnapshotName("abc_123")).To(Equal("abc_123"))
	g.Expect(SanitizeSnapshotName("abc-123")).To(Equal("abc-123"))
	g.Expect(SanitizeSnapshotName("abc123~")).To(Equal("abc123-"))
	g.Expect(SanitizeSnapshotName("abc123 ")).To(Equal("abc123-"))
	g.Expect(SanitizeSnapshotName("!@#$%^")).To(Equal("------"))
	g.Expect(SanitizeSnapshotName("test/tim")).To(Equal("test-tim"))
	g.Expect(SanitizeSnapshotName("test\\tim")).To(Equal("test-tim"))
	g.Expect(SanitizeSnapshotName("test.tim")).To(Equal("test-tim"))
	g.Expect(SanitizeSnapshotName("c:test.tim")).To(Equal("c-test-tim"))
}
