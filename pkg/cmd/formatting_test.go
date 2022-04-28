/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"fmt"
	. "github.com/onsi/gomega"
	"testing"
)

func TestCreateCamelCaseLabel(t *testing.T) {
	g := NewGomegaWithT(t)

	g.Expect(CreateCamelCaseLabel("unicastListener")).To(Equal("Unicast Listener"))
	g.Expect(CreateCamelCaseLabel("maxMemoryMB")).To(Equal("Max Memory MB"))
	g.Expect(CreateCamelCaseLabel("nodeId")).To(Equal("Node Id"))
	g.Expect(CreateCamelCaseLabel("UID")).To(Equal("UID"))
	g.Expect(CreateCamelCaseLabel("UUID")).To(Equal("UUID"))
	g.Expect(CreateCamelCaseLabel("multicastTTL")).To(Equal("Multicast TTL"))
	g.Expect(CreateCamelCaseLabel("statusHA")).To(Equal("Status HA"))
	g.Expect(CreateCamelCaseLabel("")).To(Equal(""))
}

func TestFormattingLatency(t *testing.T) {
	g := NewGomegaWithT(t)

	g.Expect(formatLatency(123.333)).To(Equal("123.333ms"))
	g.Expect(formatLatency(1)).To(Equal("1.000ms"))
	g.Expect(formatLatency0(123)).To(Equal("123ms"))
	g.Expect(formatMbps(123.2)).To(Equal("123.2Mbps"))
}

func TestFormatting(t *testing.T) {

	var (
		g        = NewGomegaWithT(t)
		mb int64 = 1024 * 1024
	)

	g.Expect(formatBytesOnly(123)).To(Equal("123"))
	g.Expect(formatBytesOnly(0)).To(Equal("0"))
	g.Expect(formatKBOnly(0)).To(Equal("0 KB"))
	g.Expect(formatKBOnly(1024)).To(Equal("1 KB"))
	g.Expect(formatKBOnly(1000)).To(Equal("0 KB"))
	g.Expect(formatKBOnly(1025)).To(Equal("1 KB"))
	g.Expect(formatKBOnly(13000)).To(Equal("12 KB"))
	g.Expect(formatMBOnly(0)).To(Equal("0 MB"))
	g.Expect(formatMBOnly(10 * mb)).To(Equal("10 MB"))
	g.Expect(formatMBOnly(10*mb - 100)).To(Equal("9 MB"))

	g.Expect(formatGBOnly(0)).To(Equal("0.0 GB"))
	g.Expect(formatGBOnly(123 * mb)).To(Equal("0.1 GB"))
	g.Expect(formatGBOnly(12344 * mb)).To(Equal("12.1 GB"))

}

func TestGetMaxColumnLengths(t *testing.T) {
	g := NewGomegaWithT(t)

	g.Expect(len(getMaxColumnLengths([]string{}))).To(Equal(0))
	values := getMaxColumnLengths([]string{"A" + sep + "B", "B" + sep + "CDD"})
	g.Expect(len(values)).To(Equal(2))
	g.Expect(values[0]).To(Equal(1))
	g.Expect(values[1]).To(Equal(3))
}

func TestMax(t *testing.T) {
	g := NewGomegaWithT(t)

	g.Expect(max(123, 124)).To(Equal(int64(124)))
	g.Expect(max(-1, 124)).To(Equal(int64(124)))
}

func TestFormattingPercent(t *testing.T) {
	g := NewGomegaWithT(t)

	g.Expect(formatPercent(.5)).To(Equal("50.00%"))
	g.Expect(formatPercent(-1)).To(Equal("n/a"))
}

func TestFormattingAllStringsWithAlignment(t *testing.T) {
	g := NewGomegaWithT(t)

	values := make([]string, 3)

	values[0] = getColumns("ONE", "TWO", "THREE")
	values[1] = getColumns("string", "123,200", "100MB")
	values[2] = getColumns("string", "123", "10MB")

	var alignment1 = []string{L, R, R}
	var alignment2 = []string{L} // test incorrect length which will turn it off

	result := formatLinesAllStringsWithAlignment(alignment1, values)
	g.Expect(result).To(Equal(`ONE         TWO  THREE
string  123,200  100MB
string      123   10MB
`))

	result = formatLinesAllStringsWithAlignment(alignment2, values)
	fmt.Println(result)
	g.Expect(result).To(Equal(`ONE     TWO      THREE
string  123,200  100MB
string  123      10MB 
`))
}

func TestGetColumns(t *testing.T) {
	g := NewGomegaWithT(t)
	g.Expect(getColumns("A", "B")).To(Equal("A" + sep + "B"))
	g.Expect(getColumns("A")).To(Equal("A"))
	g.Expect(getColumns("A", "B", "C")).To(Equal("A" + sep + "B" + sep + "C"))
	g.Expect(getColumns("A", "B", "C", "D")).To(Equal("A" + sep + "B" + sep + "C" + sep + "D"))
	g.Expect(getColumns()).To(Equal(""))
}

// TestFormattingAllStringsWithAlignmentMax1 tests truncated 1st column
func TestFormattingAllStringsWithAlignmentMax1(t *testing.T) {
	g := NewGomegaWithT(t)

	values := make([]string, 3)

	values[0] = getColumns("ONE", "TWO", "THREE")
	values[1] = getColumns("abcdefghijh", "123,200", "100MB")
	values[2] = getColumns("string", "123", "10MB")

	var alignment1 = []string{L, R, R}

	result := formatLinesAllStringsWithAlignmentMax(alignment1, values, 10)
	fmt.Println(result)
	g.Expect(result).To(Equal(`ONE             TWO  THREE
abcdefg...  123,200  100MB
string          123   10MB
`))
}

// TestFormattingAllStringsWithAlignmentMax2 tests all columns < max
func TestFormattingAllStringsWithAlignmentMax2(t *testing.T) {
	g := NewGomegaWithT(t)

	values := make([]string, 3)

	values[0] = getColumns("ONE", "TWO", "THREE")
	values[1] = getColumns("123", "123,200", "100MB")
	values[2] = getColumns("string", "123", "10MB")

	var alignment1 = []string{L, R, R}

	result := formatLinesAllStringsWithAlignmentMax(alignment1, values, 10)
	fmt.Println(result)
	g.Expect(result).To(Equal(`ONE         TWO  THREE
123     123,200  100MB
string      123   10MB
`))
}

// TestFormattingAllStringsWithAlignmentMax3 tests all columns truncates
func TestFormattingAllStringsWithAlignmentMax3(t *testing.T) {
	g := NewGomegaWithT(t)

	values := make([]string, 3)

	values[0] = getColumns("ONE", "TWO", "THREE")
	values[1] = getColumns("1this is really long", "1this must be event longer", "1wow how long is this string")
	values[2] = getColumns("2this is really long", "2this must be event longer", "2wow how long is this string")

	var alignment1 = []string{L, L, L}

	result := formatLinesAllStringsWithAlignmentMax(alignment1, values, 10)
	fmt.Println(result)
	g.Expect(result).To(Equal(`ONE         TWO         THREE     
1this i...  1this m...  1wow ho...
2this i...  2this m...  2wow ho...
`))
}
