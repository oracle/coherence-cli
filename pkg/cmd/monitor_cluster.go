/*
 * Copyright (c) 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultLayoutName    = "default"
	defaultLayout        = "members,healthSummary:services,caches:proxies,http-servers:network-stats"
	pressAdditional      = "(press key in [] to expand panel)"
	pressAdditionalReset = "(press space-bar to exit expand)"
	noContent            = "  No Content"
)

var (
	mutex                  sync.Mutex
	lastClusterSummaryInfo clusterSummaryInfo
	emptyStringArray       = make([]string, 0)
	layoutParam            string
	padMaxHeightParam      bool
	monitorCluster         bool
	additionalMonitorMsg   = pressAdditional
	expandedPanel          = ""
	panelCodes             = []rune{'1', '2', '3', '4', '5', '6', '7', '8', '9',
		'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n',
		'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'x', 'y', 'z'}
	lastPanelCode  rune
	initialRefresh = true
	lastDuration   time.Duration
)

var validPanels = []Panel{
	createContentPanel(10, "caches", "Caches", "show caches", cachesContent),
	createContentPanel(8, "departedMembers", "Departed Members", "show departed members", departedMembersContent),
	createContentPanel(5, "elastic-data", "Elastic Data", "show elastic data", elasticDataContent),
	createContentPanel(8, "executors", "Executors", "show Executors", executorsContent),
	createContentPanel(10, "healthSummary", "Health Summary", "show health summary", healthSummaryContent),
	createContentPanel(5, "federation-all", "Federation All", "show all federation details", federationAllContent),
	createContentPanel(5, "federation-dest", "Federation Destinations", "show federation destinations", federationDestinationsContent),
	createContentPanel(5, "federation-origins", "Federation Origins", "show federation origins", federationOriginsContent),
	createContentPanel(8, "http-servers", "HTTP Servers", "show HTTP servers", httpServersContent),
	createContentPanel(8, "http-sessions", "HTTP Sessions", "show HTTP sessions", httpSessionsContent),
	createContentPanel(7, "membersSummary", "Members Summary", "show members summary", membersSummaryContent),
	createContentPanel(5, "machines", "Machines", "show machines", machinesContent),
	createContentPanel(10, "members", "Members", "show members", membersContent),
	createContentPanel(7, "membersShort", "Members (Short)", "show members (short)", membersOnlyContent),
	createContentPanel(8, "network-stats", "Network Stats", "show network stats", networkStatsContent),
	createContentPanel(6, "persistence", "Persistence", "show persistence", persistenceContent),
	createContentPanel(8, "proxies", "Proxy Servers", "show proxy servers", proxiesContent),
	createContentPanel(6, "reporters", "Reporters", "show reporters", reportersContent),
	createContentPanel(10, "services", "Services", "show services", servicesContent),
	createContentPanel(8, "topics", "Topics", "show topics", topicsContent),
	createContentPanel(8, "view-caches", "View Caches", "show view caches", viewCachesContent),
}

var longDescription = `The 'monitor cluster' command displays a text base UI to monitor the overall cluster.
You can specify a layout to show by providing a value for '-l'. Panels can be specified using 'panel1:panel1,panel3'.
Specifying a ':' is the line separator and ',' means panels on the same line. The valid panel types are below:

`

// monitorClusterCmd represents the monitor cluster command
var monitorClusterCmd = &cobra.Command{
	Use:               "cluster connection-name",
	Short:             "monitors the cluster using text based UI",
	Long:              longDescription + fmt.Sprintf("%v", getValidPanelTypes()),
	ValidArgsFunction: completionAllClusters,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, youMustProviderClusterMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			clusterName  = args[0]
			dataFetcher  fetcher.Fetcher
			err          error
			parsedLayout []string
		)

		// set to tru to turn off incompatible color formatting
		monitorCluster = true

		found, _ := GetClusterConnection(clusterName)
		if !found {
			return errors.New(UnableToFindClusterMsg + clusterName)
		}

		dataFetcher, err = GetDataFetcher(clusterName)
		if err != nil {
			return err
		}

		// validate the layout
		if layoutParam == defaultLayoutName {
			layoutParam = defaultLayout
		}
		parsedLayout, err = parseLayout(layoutParam)
		if err != nil {
			return err
		}

		// now validate each of the lines
		err = validatePanels(parsedLayout)
		if err != nil {
			return err
		}

		// retrieve cluster details first so if we are connected
		// to WLS or need authentication, this can be done first
		_, err = dataFetcher.GetClusterDetailsJSON()
		if err != nil {
			return fmt.Errorf("unable to connect to cluster %s: %v", clusterName, err)
		}

		screen, err := tcell.NewScreen()
		if err != nil {
			return err
		}
		if err = screen.Init(); err != nil {
			return err
		}
		defer screen.Fini()

		screen.SetStyle(tcell.StyleDefault)

		// ensure we reset the screen on any panic
		defer func() {
			if r := recover(); r != nil {
				screen.Fini()
				log.Println("Panic: ", r)
			}
		}()

		exit := make(chan struct{})

		// initial update
		err = updateScreen(screen, dataFetcher, parsedLayout, true)
		if err != nil {
			return err
		}

		go func() {
			for {
				select {
				case <-exit:
					return
				case <-time.After(time.Duration(watchDelay) * time.Second):
					err = updateScreen(screen, dataFetcher, parsedLayout, true)
					if err != nil {
						return
					}
				}
			}
		}()

		for {
			ev := screen.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventResize:
				err = updateScreen(screen, dataFetcher, parsedLayout, false)
				screen.Sync()
			case *tcell.EventKey:
				pressedKey := ev.Rune()
				// Exit for 'q', ESC, or CTRL-C
				if ev.Key() == tcell.KeyESC || ev.Key() == tcell.KeyCtrlC {
					close(exit)
					return nil
				}

				if (pressedKey >= '1' && pressedKey <= '9' && pressedKey <= lastPanelCode) ||
					(pressedKey >= 'a' && pressedKey <= 'z' && pressedKey <= lastPanelCode) {
					expandedPanel = string(pressedKey)
					additionalMonitorMsg = pressAdditionalReset
					screen.Clear()
					err = updateScreen(screen, dataFetcher, parsedLayout, false)
					screen.Sync()
				} else if ev.Rune() == ' ' {
					expandedPanel = ""
					additionalMonitorMsg = pressAdditional
					err = updateScreen(screen, dataFetcher, parsedLayout, false)
					screen.Sync()
				}
			}
		}
	},
}

func updateScreen(screen tcell.Screen, dataFetcher fetcher.Fetcher, parsedLayout []string, refresh bool) error {
	var (
		errorList []error
		err       error
		cluster   config.Cluster
		rows      int
	)

	mutex.Lock()
	defer mutex.Unlock()

	w, h := screen.Size()

	if refresh {
		startTime := time.Now()
		if initialRefresh {
			drawText(screen, w-20, 0, w, 0, tcell.StyleDefault, " Retrieving data...")
			screen.Show()
			initialRefresh = false
		}
		lastClusterSummaryInfo, errorList = retrieveClusterSummary(dataFetcher)
		lastDuration = time.Since(startTime)

		if lastDuration > time.Second {
			// if the duration of data retrieval is > 1 second then display the refresh message
			initialRefresh = true
		}

		if len(errorList) > 0 {
			err = utils.GetErrors(errorList)
			_, _ = fmt.Fprint(os.Stderr, err.Error())
			return err
		}
	}

	screen.Clear()

	err = json.Unmarshal(lastClusterSummaryInfo.clusterResult, &cluster)
	if err != nil {
		return err
	}

	drawHeader(screen, w, h, cluster)

	var (
		widths      []int
		panelNumber = 0
	)
	startY := 2

	// loop through each of the layouts and draw each row
	for _, panelRow := range parsedLayout {
		panels := strings.Split(panelRow, ",")
		panelCount := len(panels)

		// from the panel count figure out the width
		if panelCount == 1 {
			widths = []int{w}
		} else {
			widths = getLengths(w, panelCount)
		}

		startX := 0
		maxHeight := 0

		// now draw each of the panels across each row
		for i, panelName := range panels {
			panel := getPanel(panelName)
			if panel == nil {
				return fmt.Errorf("unable to find panel [%v]", panelName)
			}

			var panelCode rune

			if panelNumber > len(panelCodes)-1 {
				return fmt.Errorf("maximum number of panels of %v is reached", len(panelCodes))
			}
			panelCode = panelCodes[panelNumber]

			lastPanelCode = panelCode

			var singlePanel = expandedPanel != ""

			if !singlePanel || (singlePanel && panelCode == rune(expandedPanel[0])) {
				var (
					realStartX = startX
					realStartY = startY
					realWidth  = widths[i]
				)
				if singlePanel {
					realStartX = 0
					realStartY = 2
					realWidth = w
				}
				// now we have the panel then draw it
				rows, err = drawContent(screen, dataFetcher, panel, realStartX, realStartY, realWidth, panelCode)
				if err != nil {
					return err
				}
				startX += widths[i]
				if rows > maxHeight {
					maxHeight = rows
				}
			}
			panelNumber++
		}

		// move to new row
		startY += maxHeight
	}

	screen.Show()

	return nil
}

var membersContent = func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	memberSummary = false
	departedMembers = false
	showMembersOnly = false
	return clusterMembersSummaryInternal(dataFetcher, clusterSummary)
}

var membersOnlyContent = func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	memberSummary = false
	departedMembers = false
	showMembersOnly = true
	return clusterMembersSummaryInternal(dataFetcher, clusterSummary)
}

var departedMembersContent = func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	memberSummary = false
	departedMembers = true
	showMembersOnly = false
	return clusterMembersSummaryInternal(dataFetcher, clusterSummary)
}

var membersSummaryContent = func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	memberSummary = true
	departedMembers = false
	showMembersOnly = false
	return clusterMembersSummaryInternal(dataFetcher, clusterSummary)
}

var machinesContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	return strings.Split(FormatMachines(clusterSummary.machines), "\n"), nil
}

var clusterMembersSummaryInternal = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	var (
		members = config.Members{}
		storage = config.StorageDetails{}
		cluster = config.Cluster{}
	)

	err := json.Unmarshal(clusterSummary.membersResult, &members)
	if err != nil {
		return emptyStringArray, err
	}

	err = json.Unmarshal(clusterSummary.storageData, &storage)
	if err != nil {
		return emptyStringArray, err
	}

	err = json.Unmarshal(lastClusterSummaryInfo.clusterResult, &cluster)
	if err != nil {
		return emptyStringArray, err
	}

	storageMap := utils.GetStorageMap(storage)

	if departedMembers {
		departedList, err1 := decodeDepartedMembers(cluster.MembersDeparted)
		if err1 != nil {
			return emptyStringArray, err1
		}
		return strings.Split(FormatDepartedMembers(departedList), "\n"), nil
	}

	result := FormatMembers(members.Members, true, storageMap, memberSummary, cluster.MembersDepartureCount)

	return strings.Split(result, "\n"), nil
}

var servicesContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	var services = config.ServicesSummaries{}

	err := json.Unmarshal(clusterSummary.servicesResult, &services)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(FormatServices(DeduplicateServices(services, "all")), "\n"), nil
}

var viewCachesContent = func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	value, err := formatViewCachesSummary(clusterSummary.serviceList, dataFetcher)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(value, "\n"), nil
}

var executorsContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	return strings.Split(FormatExecutors(clusterSummary.executors.Executors, true), "\n"), nil
}

var elasticDataContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	edData, err := getElasticDataResult(clusterSummary.flashResult, clusterSummary.ramResult)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(edData, "\n"), nil
}

var persistenceContent = func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	var services = config.ServicesSummaries{}

	err := json.Unmarshal(clusterSummary.servicesResult, &services)
	if err != nil {
		return emptyStringArray, err
	}
	deDuplicatedServices := DeduplicatePersistenceServices(services)

	err = processPersistenceServices(deDuplicatedServices, dataFetcher)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(FormatPersistenceServices(deDuplicatedServices, true), "\n"), nil
}

var reportersContent = func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	var reporters = config.Reporters{}
	if len(clusterSummary.reportersResult) > 0 {
		err := json.Unmarshal(clusterSummary.reportersResult, &reporters)
		if err != nil {
			return emptyStringArray, err
		}

		return strings.Split(FormatReporters(reporters.Reporters), "\n"), nil
	}

	return []string{}, nil
}

var networkStatsContent = func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	var members = config.Members{}

	err := json.Unmarshal(clusterSummary.membersResult, &members)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(FormatNetworkStatistics(members.Members), "\n"), nil
}

var httpSessionsContent = func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	var httpSessions = config.HTTPSessionSummaries{}

	if len(clusterSummary.http) == 0 {
		return []string{}, nil
	}

	err := json.Unmarshal(clusterSummary.http, &httpSessions)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(FormatHTTPSessions(DeduplicateSessions(httpSessions), true), "\n"), nil
}

var federationAllContent = func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	if len(clusterSummary.finalSummariesDestinations) > 0 || len(clusterSummary.finalSummariesOrigins) > 0 {
		var sb strings.Builder
		if len(clusterSummary.finalSummariesDestinations) > 0 {
			sb.WriteString(FormatFederationSummary(clusterSummary.finalSummariesDestinations, destinations))
		}
		sb.WriteString("\n")
		if len(clusterSummary.finalSummariesOrigins) > 0 {
			sb.WriteString(FormatFederationSummary(clusterSummary.finalSummariesOrigins, origins))
		}
		return strings.Split(sb.String(), "\n"), nil
	}

	return []string{}, nil
}

var federationDestinationsContent = func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	if len(clusterSummary.finalSummariesDestinations) > 0 {
		return strings.Split(FormatFederationSummary(clusterSummary.finalSummariesDestinations, destinations), "\n"), nil
	}

	return []string{}, nil
}

var federationOriginsContent = func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	if len(clusterSummary.finalSummariesOrigins) > 0 {
		return strings.Split(FormatFederationSummary(clusterSummary.finalSummariesOrigins, origins), "\n"), nil
	}

	return []string{}, nil
}

var healthSummaryContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	var healthSummaries = config.HealthSummaries{}

	err := json.Unmarshal(clusterSummary.healthResult, &healthSummaries)
	if err != nil {
		return emptyStringArray, err
	}
	healthShort := summariseHealth(filterHealth(healthSummaries))

	return strings.Split(FormatHealthSummary(healthShort), "\n"), nil
}

var proxiesContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	return proxiesContentInternal("tcp", clusterSummary)
}

var httpServersContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	return proxiesContentInternal("http", clusterSummary)
}

var proxiesContentInternal = func(protocol string, clusterSummary clusterSummaryInfo) ([]string, error) {
	var proxiesSummary = config.ProxiesSummary{}

	err := json.Unmarshal(clusterSummary.proxyResults, &proxiesSummary)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(FormatProxyServers(proxiesSummary.Proxies, protocol), "\n"), nil
}

var cachesContent = func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	cachesData, err := formatCachesSummary(clusterSummary.serviceList, dataFetcher)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(cachesData, "\n"), nil
}

var topicsContent = func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	return strings.Split(FormatTopicsSummary(clusterSummary.topicsDetails.Details), "\n"), nil
}

//
// Panel utility functions and types
//

// contentFunction generates content for a panel.
type contentFunction func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error)

type Panel interface {
	GetPanelName() string
	GetMaxHeight() int
	GetTitle() string
	GetContentFunction() contentFunction
	GetDescription() string
}

type panelImpl struct {
	PanelName       string
	MaxHeight       int
	Title           string
	ContentFunction contentFunction
	Description     string
}

func (cs panelImpl) GetPanelName() string {
	return cs.PanelName
}

func (cs panelImpl) GetTitle() string {
	return cs.Title
}

func (cs panelImpl) GetDescription() string {
	return cs.Description
}

func (cs panelImpl) GetMaxHeight() int {
	return cs.MaxHeight
}

func (cs panelImpl) GetContentFunction() contentFunction {
	return cs.ContentFunction
}

// createContentPanel creates a standard content panel.
func createContentPanel(maxHeight int, panelName, title, description string, f contentFunction) Panel {
	return panelImpl{
		MaxHeight:       maxHeight,
		PanelName:       panelName,
		Title:           title,
		ContentFunction: f,
		Description:     description,
	}
}

func parseLayout(layout string) ([]string, error) {
	if layout == "" {
		return emptyStringArray, errors.New("invalid layout")
	}
	s := strings.Split(layout, ":")
	if len(s) == 0 {
		return emptyStringArray, errors.New("invalid layout")
	}

	return s, nil
}

func getPanel(panelName string) Panel {
	for _, panel := range validPanels {
		if panel.GetPanelName() == panelName {
			return panel
		}
	}
	return nil
}

func validatePanels(layout []string) error {
	for _, v := range layout {
		// split by "," for multiple per line
		s := strings.Split(v, ",")

		for _, vv := range s {
			found := false
			for _, l := range validPanels {
				if vv == l.GetPanelName() {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("unable to find panel [%s]", vv)
			}
		}
	}
	return nil
}

// drawContent draws content and returns the height it drew
func drawContent(screen tcell.Screen, dataFetcher fetcher.Fetcher, panel Panel, x, y, w int, code rune) (int, error) {
	h := panel.GetMaxHeight()
	title := panel.GetTitle()

	content, err := panel.GetContentFunction()(dataFetcher, lastClusterSummaryInfo)
	if err != nil {
		return 0, err
	}

	l := len(content)

	if l == 0 || l == 1 && content[0] == "" {
		content = []string{" ", noContent, " "}
		l = len(content)
	}

	if padMaxHeightParam && l < h {
		for i := l; i < h; i++ {
			content = append(content, "")
		}
	}

	if !padMaxHeightParam {
		content = trimBlankContent(content)
	}

	trimmed := false
	var singlePanel = expandedPanel != ""
	if singlePanel {
		// reset the height to max
		h, _ = screen.Size()
		h -= 2
	}

	rows := len(content)
	if !singlePanel && rows > h {
		rows = h
		trimmed = true
	}

	h = rows + 1

	// trim any content > w
	for i := range content {
		line := content[i]
		if len(line) > w {
			content[i] = line[:w-1]
		}
	}

	trimmedText := ""
	if trimmed {
		trimmedText = fmt.Sprintf("%v%s", string(tcell.RuneHLine), "(trimmed)")
	}

	drawBox(screen, x, y, x+w-1, y+h, tcell.StyleDefault, fmt.Sprintf("%s [%v]%s", title, string(code), trimmedText))

	for line := 1; line <= rows; line++ {
		drawText(screen, x+1, y+line, x+w-1, y+h-1, tcell.StyleDefault, content[line-1])
	}

	return rows + 2, nil
}

func trimBlankContent(content []string) []string {
	last := len(content)

	for i := len(content) - 1; i >= 0; i-- {
		if content[i] != "" {
			break
		}
		last = i
	}
	return content[:last]
}

// drawHeader draws the screen header with cluster information.
func drawHeader(screen tcell.Screen, w, h int, cluster config.Cluster) {
	version := strings.Split(cluster.Version, " ")
	title := fmt.Sprintf("Coherence CLI: %s - Monitoring cluster %s (%s) ESC to quit %s. (refresh=%v)",
		time.Now().Format(time.DateTime), cluster.ClusterName, version[0], additionalMonitorMsg, lastDuration)
	drawText(screen, 1, 0, w-1, h-1, tcell.StyleDefault.Reverse(true), title)

}

// drawText draws text on the screen.
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

// drawBox draws a box on the screen and fills it.
func drawBox(s tcell.Screen, x1, y1, x2, y2 int, style tcell.Style, title string) {
	if y2 < y1 {
		y1, y2 = y2, y1
	}
	if x2 < x1 {
		x1, x2 = x2, x1
	}

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

	if y1 != y2 && x1 != x2 {
		s.SetContent(x1, y1, tcell.RuneULCorner, nil, style)
		s.SetContent(x2, y1, tcell.RuneURCorner, nil, style)
		s.SetContent(x1, y2, tcell.RuneLLCorner, nil, style)
		s.SetContent(x2, y2, tcell.RuneLRCorner, nil, style)
	}

	drawText(s, x1+2, y1, x2-1, y2-1, style, title)
}

// getValidPanelTypes returns the list of panels for the --help command.
func getValidPanelTypes() string {
	valid := ""
	for _, p := range validPanels {
		valid = valid + fmt.Sprintf("%-20s: %s\n", p.GetPanelName(), p.GetDescription())
	}

	return valid
}

// getLengths splits up widths into various lengths taking into account the remainder.
func getLengths(width, count int) []int {
	q := width / count
	r := width % count
	lens := make([]int, count)
	for i := 0; i < count; i++ {
		if i < r {
			lens[i] = q + 1
		} else {
			lens[i] = q
		}
	}
	return lens
}

func init() {
	monitorClusterCmd.Flags().StringVarP(&layoutParam, "layout", "l", defaultLayoutName, "layout to use")
	monitorClusterCmd.Flags().BoolVarP(&padMaxHeightParam, "pad", "p", false, "pad to max height")
}
