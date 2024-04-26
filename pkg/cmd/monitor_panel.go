/*
 * Copyright (c) 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/oracle/coherence-cli/pkg/config"
	"strings"
	"time"
)

// contentFunction generates content for a panel.
type contentFunction func(clusterSummary clusterSummaryInfo) ([]string, error)

type PanelType string

type Grid int

const (
	Content PanelType = "content"
	Row     PanelType = "row"

	grid12 Grid = 12
	grid6  Grid = 6
	grid4  Grid = 4
	grid3  Grid = 3
)

type Panel interface {
	GetWidth() int
	GetHeight() int
	GetTitle() string
	GetContentFunction() contentFunction
	GetGridWidth() Grid
	GetPanels() []Panel
}

type panelImpl struct {
	Width           int
	Height          int
	Title           string
	Type            PanelType
	Panels          []Panel
	GridWidth       Grid
	ContentFunction contentFunction
}

func (cs panelImpl) GetWidth() int {
	return cs.Width
}

func (cs panelImpl) GetTitle() string {
	return cs.Title
}

func (cs panelImpl) GetHeight() int {
	return cs.Height
}

func (cs panelImpl) GetPanels() []Panel {
	return cs.Panels
}

func (cs panelImpl) GetGridWidth() Grid {
	return cs.GridWidth
}

func (cs panelImpl) GetContentFunction() contentFunction {
	return cs.ContentFunction
}

// createContentPanel creates a standard content panel.
func createContentPanel(w, h int, title string, f contentFunction, grid ...Grid) Panel {
	gw := grid12
	if len(grid) == 1 {
		gw = grid[0]
	}
	return panelImpl{Width: w, Height: h, Title: title, ContentFunction: f, Type: Content, GridWidth: gw}
}

func drawContent(screen tcell.Screen, c clusterSummaryInfo, monitorContent []Panel) error {
	var (
		x = 1
		y = 2
	)

	// loop through and display one panel per line for the moment
	for _, v := range monitorContent {
		w := v.GetWidth()
		h := v.GetHeight()
		title := v.GetTitle()

		content, err := v.GetContentFunction()(c)
		if err != nil {
			return err
		}

		rows := len(content)

		drawBox(screen, x, y, x+w, y+h, tcell.StyleDefault, title)

		for line := 1; line <= rows; line++ {
			drawText(screen, x+1, y+line, x+w-1, y+h-1, tcell.StyleDefault, content[line-1])
		}

		y += h + 1
	}
	return nil
}

func drawMainPanel(screen tcell.Screen, w, h int, cluster config.Cluster) {
	version := strings.Split(cluster.Version, " ")
	title := fmt.Sprintf("%s - Cluster: %s (%s) - %s: ESC to quit",
		time.Now().Format(time.DateTime), cluster.ClusterName, version[0], cluster.LicenseMode)

	drawBox(screen, 0, 0, w-1, h-1, tcell.StyleDefault, title)
}

func drawText(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, text string) {
	row := y1
	col := x1
	for _, r := range text {
		s.SetContent(col, row, r, nil, style)
		col++
		if col >= x2 {
			row++
			col = x1
		}
		if row > y2 {
			break
		}
	}
}

func drawBox(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, title string) {
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	if x2 < x1 {
		x1, x2 = x2, x1
	}

	// Fill background
	for row := y1; row <= y2; row++ {
		for col := x1; col <= x2; col++ {
			s.SetContent(col, row, ' ', nil, style)
		}
	}

	// Draw borders
	for col := x1; col <= x2; col++ {
		s.SetContent(col, y1, tcell.RuneHLine, nil, style)
		s.SetContent(col, y2, tcell.RuneHLine, nil, style)
	}
	for row := y1 + 1; row < y2; row++ {
		s.SetContent(x1, row, tcell.RuneVLine, nil, style)
		s.SetContent(x2, row, tcell.RuneVLine, nil, style)
	}

	// Only draw corners if necessary
	if y1 != y2 && x1 != x2 {
		s.SetContent(x1, y1, tcell.RuneULCorner, nil, style)
		s.SetContent(x2, y1, tcell.RuneURCorner, nil, style)
		s.SetContent(x1, y2, tcell.RuneLLCorner, nil, style)
		s.SetContent(x2, y2, tcell.RuneLRCorner, nil, style)
	}

	// title
	drawText(s, x1+2, y1, x2-1, y2-1, style, title)
}
