/*
 * Copyright (c) 2024, 2025 Oracle and/or its affiliates.
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
	"github.com/spf13/viper"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultLayoutName    = "default"
	pressAdditional      = "(press key in [] to toggle expand, ? = help)"
	pressAdditionalReset = "(press key in [] or ESC to exit expand)"
	noContent            = "  No Content"
	errorContent         = "Unable to retrieve data"
	unableToFindPanel    = "unable to find panel [%v], use --help or --show-panels to see all options"
	serviceNameToken     = "%SERVICE"
	cacheNameToken       = "%CACHE"
	topicNameToken       = "%TOPIC"
	subscriberToken      = "%SUBSCRIBER"
	setDefaultStyleMsg   = "Default style is now set to "
	getDefaultStyleMsg   = "Default style is: "
)

var (
	defaultMap = map[string]string{
		"default":            "summary-cluster,summary-members,machines:services,caches:proxies,http-servers:health-summary,persistence:federation-all,elastic-data:network-stats",
		"default-service":    "services:service-members:service-distributions",
		"default-cache":      "caches,cache-indexes:cache-access:cache-storage:cache-stores:cache-partitions",
		"default-topic":      "topics:topic-members:subscribers:subscriber-groups",
		"default-subscriber": "topics:subscribers:subscriber-channels",
		"default-federation": "federation-all:services:caches:elastic-data",
		"default-members":    "members:machines,departed-members:network-stats",
	}
	errSelectService       = errors.New("you must provide a service name via -S option")
	errSelectCache         = errors.New("you must provide a cache using the -C option")
	errSelectTopic         = errors.New("you must provide a topic using the -T option")
	errSelectSubscriber    = errors.New("you must provide a subscriber using the -B option")
	mutex                  sync.Mutex
	lastClusterSummaryInfo clusterSummaryInfo
	emptyStringArray       = make([]string, 0)
	layoutParam            string
	setMaxHeight           int
	originalMaxHeight      int
	padMaxHeightParam      = true
	colorStyleParam        string
	previewStylesParam     bool
	showAllPanels          bool
	ignoreRESTErrors       bool
	disablePadding         bool
	monitorCluster         bool
	additionalMonitorMsg   = pressAdditional
	expandedPanel          = ""
	panelCodes             = []rune{'1', '2', '3', '4', '5', '6', '7', '8', '9',
		'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n',
		'o', 'q', 'r', 's', 't', 'u', 'v', 'x', 'y', 'z'}
	lastPanelCode  rune
	initialRefresh = true
	lastDuration   time.Duration
	inHelp         = false
	selectedCache  string
	selectedTopic  string
	allBaseData    []string
	heightAdjust   int
	noContentArray = []string{"  ", noContent, " "}
	drawnPositions map[rune]position

	// color styles
	boxStyle   = tcell.StyleDefault
	titleStyle = tcell.StyleDefault
	textStyle  = tcell.StyleDefault
)

type StyleConfig struct {
	TextStyle  tcell.Style
	TitleStyle tcell.Style
	BoxStyle   tcell.Style
}

var styleConfigsMap = map[string]StyleConfig{
	"default": {
		TextStyle:  tcell.StyleDefault,
		TitleStyle: tcell.StyleDefault,
		BoxStyle:   tcell.StyleDefault,
	},
	"vintage": {
		TextStyle:  tcell.StyleDefault.Foreground(tcell.ColorWheat),
		TitleStyle: tcell.StyleDefault.Foreground(tcell.ColorCoral),
		BoxStyle:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
	},
	"oceanic": {
		TextStyle:  tcell.StyleDefault.Foreground(tcell.ColorLightCyan),
		TitleStyle: tcell.StyleDefault.Foreground(tcell.ColorTurquoise),
		BoxStyle:   tcell.StyleDefault.Foreground(tcell.ColorNavy),
	},
	"tty": {
		TextStyle:  tcell.StyleDefault.Foreground(tcell.ColorGreen),
		TitleStyle: tcell.StyleDefault.Foreground(tcell.ColorWhite),
		BoxStyle:   tcell.StyleDefault.Foreground(tcell.ColorGray),
	},
	"monokai": {
		TextStyle:  tcell.StyleDefault.Foreground(tcell.ColorDarkKhaki),
		TitleStyle: tcell.StyleDefault.Foreground(tcell.ColorOrangeRed),
		BoxStyle:   tcell.StyleDefault.Foreground(tcell.ColorSlateGray),
	},
	"ice": {
		TextStyle:  tcell.StyleDefault.Foreground(tcell.ColorLightSteelBlue),
		TitleStyle: tcell.StyleDefault.Foreground(tcell.ColorCoral),
		BoxStyle:   tcell.StyleDefault.Foreground(tcell.ColorSlateGray),
	},
	"high-contrast": {
		TextStyle:  tcell.StyleDefault.Foreground(tcell.ColorWhite),
		TitleStyle: tcell.StyleDefault.Foreground(tcell.ColorRed),
		BoxStyle:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
	},
	"light-mode": {
		TextStyle:  tcell.StyleDefault.Foreground(tcell.ColorBlack),
		TitleStyle: tcell.StyleDefault.Foreground(tcell.ColorDarkBlue),
		BoxStyle:   tcell.StyleDefault.Foreground(tcell.ColorDarkGray),
	},
	"solarized-dark": {
		TextStyle:  tcell.StyleDefault.Foreground(tcell.ColorLightSlateGray),
		TitleStyle: tcell.StyleDefault.Foreground(tcell.ColorGoldenrod),
		BoxStyle:   tcell.StyleDefault.Foreground(tcell.ColorTeal),
	},
	"sunset": {
		TextStyle:  tcell.StyleDefault.Foreground(tcell.ColorOrange),
		TitleStyle: tcell.StyleDefault.Foreground(tcell.ColorTomato),
		BoxStyle:   tcell.StyleDefault.Foreground(tcell.ColorDarkRed),
	},
	"midnight": {
		TextStyle:  tcell.StyleDefault.Foreground(tcell.ColorLightSlateGray),
		TitleStyle: tcell.StyleDefault.Foreground(tcell.ColorMediumPurple),
		BoxStyle:   tcell.StyleDefault.Foreground(tcell.ColorIndigo),
	},
	"dracula": {
		TextStyle:  tcell.StyleDefault.Foreground(tcell.ColorMediumOrchid),
		TitleStyle: tcell.StyleDefault.Foreground(tcell.ColorDeepPink),
		BoxStyle:   tcell.StyleDefault.Foreground(tcell.ColorSlateBlue),
	},
	"clay": {
		TextStyle:  tcell.StyleDefault.Foreground(tcell.ColorSandyBrown),
		TitleStyle: tcell.StyleDefault.Foreground(tcell.ColorPeru),
		BoxStyle:   tcell.StyleDefault.Foreground(tcell.ColorMaroon),
	},
	"fog": {
		TextStyle:  tcell.StyleDefault.Foreground(tcell.ColorGainsboro),
		TitleStyle: tcell.StyleDefault.Foreground(tcell.ColorSlateGray),
		BoxStyle:   tcell.StyleDefault.Foreground(tcell.ColorDarkSlateGray),
	},
}

var validPanels = []panelImpl{
	createContentPanel(7, "caches", "Caches", "show caches", cachesContent, cachesPanelData, servicesPanelData),
	createContentPanel(7, "cache-access", "Cache Access (%SERVICE/%CACHE)", "show cache access", cacheAccessContent, cachesPanelData, servicesPanelData),
	createContentPanel(7, "cache-indexes", "Cache Indexes (%SERVICE/%CACHE)", "show cache indexes", cacheIndexesContent, cachesPanelData, servicesPanelData),
	createContentPanel(7, "cache-storage", "Cache Storage (%SERVICE/%CACHE)", "show cache storage", cacheStorageContent, cachesPanelData, servicesPanelData),
	createContentPanel(7, "cache-stores", "Cache Stores (%SERVICE/%CACHE)", "show cache stores", cacheStoresContent),
	createContentPanel(7, "cache-partitions", "Cache Partitions (%SERVICE/%CACHE)", "show cache partitions", cachePartitionContent, cachesPanelData, servicesPanelData),
	createContentPanel(7, "departed-members", "Departed Members", "show departed members", departedMembersContent, memberPanelData, storagePanelData),
	createContentPanel(4, "elastic-data", "Elastic Data", "show elastic data", elasticDataContent, elasticDataPanelData),
	createContentPanel(7, "executors", "Executors", "show Executors", executorsContent, executorsPanelData),
	createContentPanel(7, "health-summary", "Health Summary", "show health summary", healthSummaryContent, healthPanelData),
	createContentPanel(4, "federation-all", "Federation All", "show all federation details", federationAllContent, federationPanelData),
	createContentPanel(3, "federation-dest", "Federation Destinations", "show federation destinations", federationDestinationsContent, federationPanelData),
	createContentPanel(3, "federation-origins", "Federation Origins", "show federation origins", federationOriginsContent, federationPanelData),
	createContentPanel(7, "http-servers", "HTTP Servers", "show HTTP servers", httpServersContent, proxiesPanelData),
	createContentPanel(7, "http-sessions", "HTTP Sessions", "show HTTP sessions", httpSessionsContent, httpSessionsPanelData),
	createContentPanel(7, "machines", "Machines", "show machines", machinesContent, memberPanelData, storagePanelData),
	createContentPanel(7, "members", "Members", "show members", membersContent, memberPanelData, storagePanelData),
	createContentPanel(7, "members-short", "Members (Short)", "show members (short)", membersOnlyContent, memberPanelData, storagePanelData),
	createContentPanel(7, "network-stats", "Network Stats", "show network stats", networkStatsContent, memberPanelData, storagePanelData),
	createContentPanel(7, "persistence", "Persistence", "show persistence", persistenceContent, servicesPanelData),
	createContentPanel(7, "proxies", "Proxy Servers", "show proxy servers", proxiesContent, proxiesPanelData),
	createContentPanel(7, "proxy-connections", "Proxy Connections (%SERVICE)", "show proxy connections", proxyConnectionsContent, proxiesPanelData),
	createContentPanel(7, "reporters", "Reporters", "show reporters", reportersContent, reportersPanelData),
	createContentPanel(7, "services", "Services", "show services", servicesContent, servicesPanelData),
	createContentPanel(7, "service-members", "Service Members (%SERVICE)", "show service members", serviceMembersContent, servicesPanelData),
	createContentPanel(7, "service-distributions", "Service Distributions (%SERVICE)", "show service distributions", serviceDistributionsContent, servicesPanelData),
	createContentPanel(7, "service-ownership", "Service Ownership (%SERVICE)", "show service ownership", serviceOwnershipContent, servicesPanelData),
	createContentPanel(7, "service-storage", "Service Storage", "show service storage", serviceStorageContent, servicesPanelData),
	createContentPanel(7, "topic-members", "Topic Members (%SERVICE/%TOPIC)", "show topic members", topicMembersContent, topicsPanelData),
	createContentPanel(7, "subscribers", "Topic Subscribers (%SERVICE/%TOPIC)", "show topic subscribers", topicSubscribersContent, topicsPanelData),
	createContentPanel(7, "subscriber-channels", "Subscriber Channels (%SERVICE/%TOPIC/%SUBSCRIBER)", "show topic subscriber channels", topicSubscriberChannelsContent, topicsPanelData),
	createContentPanel(7, "subscriber-groups", "Subscriber Channels (%SERVICE/%TOPIC)", "show subscriber groups", topicSubscriberGroupsContent, topicsPanelData),
	createContentPanel(7, "topics", "Topics", "show topics", topicsContent, topicsPanelData),
	createContentPanel(7, "view-caches", "View Caches", "show view caches", viewCachesContent, servicesPanelData),
	// cluster summary panels below
	createContentPanel(7, "summary-cluster", "Cluster Information", "show cluster information", summaryClusterContent, memberPanelData, storagePanelData),
	createContentPanel(7, "summary-members", "Members Summary", "show members summary", membersContent, memberPanelData, storagePanelData),
	createContentPanel(7, "summary-caches", "Caches Summary", "show caches summary", cachesContent, cachesPanelData, servicesPanelData),
}

type position struct {
	x      int
	y      int
	width  int
	height int
}

// monitorClusterCmd represents the monitor cluster command
var monitorClusterCmd = &cobra.Command{
	Use:   "cluster connection-name",
	Short: "monitors the cluster using text based UI",
	Long: `The 'monitor cluster' command displays a text based UI to monitor the overall cluster.
You can specify a layout to show by providing a value for '-l'. Panels can be specified using 'panel1:panel1,panel3'.
Specifying a ':' is the line separator and ',' means panels on the same line. If you don't specify one the 'default' layout is used.
There are a number of layouts available: 'default-service', 'default-cache', 'default-topic' and 'default-subscriber' which 
require you to specify cache, service, topic or subscriber.
Use --show-panels to show all available panels.`,
	ValidArgsFunction: completionAllClusters,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 && (!showAllPanels && !previewStylesParam) {
			displayErrorAndExit(cmd, youMustProviderConnectionMessage)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			clusterName  string
			dataFetcher  fetcher.Fetcher
			err          error
			parsedLayout []string
		)

		if showAllPanels {
			cmd.Println(getValidPanelTypes())
			return nil
		}

		if previewStylesParam {
			return previewAllStyles()
		}

		clusterName = args[0]

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

		if err = setColorStyle(); err != nil {
			return err
		}

		if disablePadding {
			padMaxHeightParam = false
		}

		// check for custom panel layouts added by "add panel"
		if layoutParam != "" {
			if l := getPanelLayout(layoutParam); l != "" {
				layoutParam = l

			}
		}

		// check for default layouts
		if l, ok := defaultMap[layoutParam]; ok {
			layoutParam = l
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

		allBaseData = getAllBaseData(parsedLayout)

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
				screen.Clear()
				screen.Show()
				screen.Fini()
				log.Println("Panic: ", r)
			}
		}()

		exit := make(chan struct{})

		originalMaxHeight = setMaxHeight

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
					if !inHelp {
						err = updateScreen(screen, dataFetcher, parsedLayout, true)
						if err != nil {
							screen.Clear()
							screen.Show()
							screen.Fini()
							panic(err)
						}
					}
				}
			}
		}()

		for {
			ev := screen.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventResize:
				if err := refresh(screen, dataFetcher, parsedLayout, false); err != nil {
					panic(err)
				}
				screen.Sync()
			case *tcell.EventKey:
				pressedKey := ev.Rune()
				// Exit for 'q', ESC, or CTRL-C
				if (ev.Key() == tcell.KeyESC && expandedPanel == "") || ev.Key() == tcell.KeyCtrlC {
					close(exit)
					return nil
				}
				if pressedKey == '?' {
					showHelp(screen)
					if err := refresh(screen, dataFetcher, parsedLayout, true); err != nil {
						panic(err)
					}
				} else if pressedKey == 'p' {
					padMaxHeightParam = !padMaxHeightParam
					if err := refresh(screen, dataFetcher, parsedLayout, false); err != nil {
						panic(err)
					}
				} else if pressedKey == '+' {
					increaseMaxHeight()
					if err := refresh(screen, dataFetcher, parsedLayout, false); err != nil {
						panic(err)
					}
				} else if pressedKey == '-' {
					decreaseMaxHeight()
					if err := refresh(screen, dataFetcher, parsedLayout, false); err != nil {
						panic(err)
					}
				} else if pressedKey == '0' {
					resetMaxHeight()
					if err := refresh(screen, dataFetcher, parsedLayout, false); err != nil {
						panic(err)
					}
				} else if ((pressedKey >= '1' && pressedKey <= '9' && pressedKey <= lastPanelCode) || ev.Key() == tcell.KeyESC) ||
					(pressedKey >= 'a' && pressedKey <= 'z' && pressedKey <= lastPanelCode) {
					updateExpanded(pressedKey, screen, dataFetcher, parsedLayout)
				}
			}
		}
	},
}

// setDefaultStyleCmd represents the set default-style or command.
var setDefaultStyleCmd = &cobra.Command{
	Use:   "default-style style",
	Short: "set default style for monitor clusters command",
	Long:  `The 'set default-style' command sets the default style for monitor clusters command.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		value := args[0]

		_, err := getStyleValue(value)
		if err != nil {
			return err
		}

		viper.Set(defaultStyleKey, value)
		err = WriteConfig()
		if err != nil {
			return err
		}
		cmd.Println(setDefaultStyleMsg + value)
		return nil
	},
}

// getDefaultStyleCmd represents the get default-style command.
var getDefaultStyleCmd = &cobra.Command{
	Use:   "default-style",
	Short: "display the current default style for monitor clusters",
	Long:  `The 'get default-style' command displays the current style for monitor clusters.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.Printf("%s%v\n", getDefaultStyleMsg, Config.DefaultStyle)
		return nil
	},
}

func previewAllStyles() error {
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
			screen.Clear()
			screen.Show()
			screen.Fini()
			log.Println("Panic: ", r)
		}
	}()

	sortedStyles := getAllSortedStyles()

	y := 1
	x := 2
	row := 1
	for _, styleKey := range sortedStyles {
		if style, ok := styleConfigsMap[styleKey]; ok {
			titleStyle = style.TitleStyle
			drawBox(screen, x, y, x+40, y+4, style.BoxStyle, fmt.Sprintf("Example style: [%s]", styleKey))
			drawText(screen, x+1, y+1, x+39, y+1, style.TextStyle, "NODE ID  ADDRESS     PORT   PROCESS")
			drawText(screen, x+1, y+2, x+39, y+2, style.TextStyle, "      1  /127.0.0.1  58086    97529")
			drawText(screen, x+1, y+3, x+39, y+3, style.TextStyle, "      2  /127.0.0.1  58044    66466")
		}

		if row%2 == 0 {
			// move to next line
			y += 5
			x = 2
		} else {
			// move across
			x += 45
		}
		row++
	}

	drawText(screen, 2, y+1, 40, y+40, tcell.StyleDefault, "Press any key to continue")

	screen.Show()

	// Wait for any key press
	for {
		ev := screen.PollEvent()
		if _, ok := ev.(*tcell.EventKey); ok {
			break
		}
	}

	return nil
}

func setColorStyle() error {
	// use default style if none specified
	if colorStyleParam == "" {
		colorStyleParam = Config.DefaultStyle
	}
	style, err := getStyleValue(colorStyleParam)
	if err != nil {
		return err
	}

	textStyle = style.TextStyle
	titleStyle = style.TitleStyle
	boxStyle = style.BoxStyle

	return nil
}

func getStyleValue(styleValue string) (StyleConfig, error) {
	style, ok := styleConfigsMap[styleValue]
	if !ok {
		return StyleConfig{}, fmt.Errorf("invalid color style %s, valid values are %v", colorStyleParam, getAllSortedStyles())
	}

	return style, nil
}

func getAllSortedStyles() []string {
	sortedList := make([]string, 0, len(styleConfigsMap))
	for k := range styleConfigsMap {
		sortedList = append(sortedList, k)
	}
	sort.Strings(sortedList)

	return sortedList
}

func updateExpanded(pressedKey rune, screen tcell.Screen, dataFetcher fetcher.Fetcher, parsedLayout []string) {
	if expandedPanel != "" {
		expandedPanel = ""
		additionalMonitorMsg = pressAdditional
	} else {
		expandedPanel = string(pressedKey)
		additionalMonitorMsg = pressAdditionalReset
	}
	if err := refresh(screen, dataFetcher, parsedLayout, false); err != nil {
		panic(err)
	}
}

func increaseMaxHeight() {
	for i := range validPanels {
		validPanels[i].MaxHeight++
	}
	heightAdjust++
	if setMaxHeight != 0 {
		setMaxHeight++
	}
}

func decreaseMaxHeight() {
	if heightAdjust > 0 {
		for i := range validPanels {
			if validPanels[i].MaxHeight > validPanels[i].OriginalMaxHeight {
				validPanels[i].MaxHeight--
			}
		}
		heightAdjust--
		if setMaxHeight != 0 {
			setMaxHeight--
		}
	}
}

func resetMaxHeight() {
	for i := range validPanels {
		validPanels[i].MaxHeight = validPanels[i].OriginalMaxHeight
	}
	heightAdjust = 0
	setMaxHeight = originalMaxHeight
}

func refresh(screen tcell.Screen, dataFetcher fetcher.Fetcher, parsedLayout []string, refresh bool) error {
	err := updateScreen(screen, dataFetcher, parsedLayout, refresh)
	if err != nil {
		screen.Sync()
	}

	return err
}

func showHelp(screen tcell.Screen) {
	help := []string{
		"",
		"  Monitor Cluster CLI Help ",
		"",
		"  - 'p' to toggle panel row padding",
		"  - '+' to increase max height of all panels",
		"  - '-' to decrease max height of all panels",
		"  - '0' to reset max height of all panels",
		"  - Key in [] to expand that panel",
		"  - ESC / CTRL-C to exit monitoring",
		"  ",
		"  Press any key to exit help.",
	}

	inHelp = true
	defer func() { inHelp = false }()
	lenHelp := len(help)

	w, h := screen.Size()
	x := w/2 - 25
	y := h/2 - lenHelp

	drawBox(screen, x, y, x+53, y+lenHelp+2, boxStyle, "Help")

	for line := 1; line <= lenHelp; line++ {
		drawText(screen, x+1, y+line, x+w-1, y+h-1, textStyle, help[line-1])
	}
	screen.Show()
	_ = screen.PollEvent()
}

func updateScreen(screen tcell.Screen, dataFetcher fetcher.Fetcher, parsedLayout []string, refresh bool) error {
	var (
		errorList []error
		err       error
		rows      int
		cluster   config.Cluster
	)

	mutex.Lock()
	defer mutex.Unlock()

	w, h := screen.Size()

	if refresh {
		startTime := time.Now()
		if initialRefresh {
			drawText(screen, w-20, 0, w, 0, textStyle, " Retrieving data...")
			screen.Show()
			initialRefresh = false
		}
		lastClusterSummaryInfo, errorList = retrieveClusterSummary(dataFetcher, allBaseData...)
		lastDuration = time.Since(startTime)

		if lastDuration > time.Second {
			// if the duration of data retrieval is > 1 second then display the refresh message
			initialRefresh = true
		}

		if len(errorList) > 0 && !ignoreRESTErrors {
			err = utils.GetErrors(errorList)
			return err
		}
	}

	screen.Clear()

	err = json.Unmarshal(lastClusterSummaryInfo.clusterResult, &cluster)
	if err != nil && !ignoreRESTErrors {
		return err
	}

	drawHeader(screen, w, h, cluster, dataFetcher)

	// re-create the list of drawn positions
	drawnPositions = make(map[rune]position, 0)

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
				return fmt.Errorf(unableToFindPanel, panelName)
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
				rows, err = drawContent(screen, dataFetcher, *panel, realStartX, realStartY, realWidth, panelCode)
				if err != nil {
					return err
				}
				startX += widths[i]
				if rows > maxHeight {
					maxHeight = rows
				}

				// save the position of the panel
				drawnPositions[panelCode] = position{x: realStartX, y: realStartY, width: realWidth, height: maxHeight}
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

var serviceMembersContent = func(dataFetcher fetcher.Fetcher, _ clusterSummaryInfo) ([]string, error) {
	var membersDetails = config.ServiceMemberDetails{}
	if serviceName == "" {
		return emptyStringArray, errSelectService
	}

	membersResult, err := dataFetcher.GetServiceMembersDetailsJSON(serviceName)
	if err != nil {
		return emptyStringArray, err
	}

	err = json.Unmarshal(membersResult, &membersDetails)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(FormatServiceMembers(membersDetails.Services), "\n"), nil
}

var serviceDistributionsContent = func(dataFetcher fetcher.Fetcher, _ clusterSummaryInfo) ([]string, error) {
	var (
		distributionsData []byte
		distributions     config.Distributions
	)

	if serviceName == "" {
		return emptyStringArray, errSelectService
	}

	servicesResult, err := GetDistributedServices(dataFetcher)
	if err != nil {
		return emptyStringArray, err
	}

	if !utils.SliceContains(servicesResult, serviceName) {
		return emptyStringArray, fmt.Errorf(unableToFindService, serviceName)
	}

	distributionsData, err = dataFetcher.GetScheduledDistributionsJSON(serviceName)
	if err != nil {
		return emptyStringArray, err
	}

	if len(distributionsData) != 0 {
		err = json.Unmarshal(distributionsData, &distributions)
		if err != nil {
			return emptyStringArray, err
		}
	} else {
		distributions.ScheduledDistributions = noDistributionsData
	}

	return strings.Split(distributions.ScheduledDistributions, "\n"), nil
}

var serviceOwnershipContent = func(dataFetcher fetcher.Fetcher, _ clusterSummaryInfo) ([]string, error) {
	var (
		membersResult  []byte
		memberNodeID   string
		membersDetails = config.ServiceMemberDetails{}
	)

	if serviceName == "" {
		return emptyStringArray, errSelectService
	}

	servicesResult, err := GetDistributedServices(dataFetcher)
	if err != nil {
		return emptyStringArray, err
	}

	if !utils.SliceContains(servicesResult, serviceName) {
		return emptyStringArray, fmt.Errorf(unableToFindService, serviceName)
	}

	// find storage member node
	membersResult, err = dataFetcher.GetServiceMembersDetailsJSON(serviceName)
	if err != nil {
		return emptyStringArray, err
	}

	if len(membersResult) != 0 {
		err = json.Unmarshal(membersResult, &membersDetails)
		if err != nil {
			return emptyStringArray, utils.GetError("unable to unmarshall members ownership", err)
		}

		for _, v := range membersDetails.Services {
			memberNodeID = v.NodeID
			break
		}

		var ownershipData []byte

		ownershipData, err = dataFetcher.GetServiceOwnershipJSON(serviceName, memberNodeID)
		if err != nil {
			return emptyStringArray, err
		}

		result, err := getOwnershipData(dataFetcher, ownershipData)
		if err != nil {
			return emptyStringArray, err
		}

		return strings.Split(FormatPartitionOwnership(result), "\n"), nil
	}

	return noContentArray, nil
}

var serviceStorageContent = func(dataFetcher fetcher.Fetcher, _ clusterSummaryInfo) ([]string, error) {
	storageSummary, err := getServiceStorageDetails(dataFetcher)

	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(FormatServicesStorage(storageSummary), "\n"), nil
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

	if contentIsEmpty(clusterSummary.servicesResult) {
		return noContentArray, nil
	}

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

var reportersContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	var reporters = config.Reporters{}
	if !contentIsEmpty(clusterSummary.reportersResult) {
		err := json.Unmarshal(clusterSummary.reportersResult, &reporters)
		if err != nil {
			return emptyStringArray, err
		}

		return strings.Split(FormatReporters(reporters.Reporters), "\n"), nil
	}

	return []string{}, nil
}

var networkStatsContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	var members = config.Members{}

	err := json.Unmarshal(clusterSummary.membersResult, &members)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(FormatNetworkStatistics(members.Members), "\n"), nil
}

var httpSessionsContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
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

var federationAllContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
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

	return noContentArray, nil
}

var federationDestinationsContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	if len(clusterSummary.finalSummariesDestinations) > 0 {
		return strings.Split(FormatFederationSummary(clusterSummary.finalSummariesDestinations, destinations), "\n"), nil
	}

	return noContentArray, nil
}

var federationOriginsContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	if len(clusterSummary.finalSummariesOrigins) > 0 {
		return strings.Split(FormatFederationSummary(clusterSummary.finalSummariesOrigins, origins), "\n"), nil
	}

	return noContentArray, nil
}

var healthSummaryContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	var healthSummaries = config.HealthSummaries{}

	if contentIsEmpty(clusterSummary.healthResult) {
		return noContentArray, nil
	}

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

	if contentIsEmpty(clusterSummary.proxyResults) {
		return noContentArray, nil
	}

	err := json.Unmarshal(clusterSummary.proxyResults, &proxiesSummary)
	if err != nil {
		return emptyStringArray, err
	}
	finalResult, err := summarizeProxies(proxiesSummary.Proxies, protocol)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(FormatProxyServersSummary(finalResult, protocol), "\n"), nil
}

var proxyConnectionsContent = func(dataFetcher fetcher.Fetcher, _ clusterSummaryInfo) ([]string, error) {
	if serviceName == "" {
		return emptyStringArray, errSelectService
	}

	connectionDetailsFinal, err := getProxyConnections(dataFetcher, serviceName)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(FormatProxyConnections(connectionDetailsFinal), "\n"), nil
}

var cacheAccessContent = func(dataFetcher fetcher.Fetcher, _ clusterSummaryInfo) ([]string, error) {
	return getCacheContent(dataFetcher, "access")
}

var cacheIndexesContent = func(dataFetcher fetcher.Fetcher, _ clusterSummaryInfo) ([]string, error) {
	return getCacheContent(dataFetcher, "index")
}

var cacheStorageContent = func(dataFetcher fetcher.Fetcher, _ clusterSummaryInfo) ([]string, error) {
	return getCacheContent(dataFetcher, "storage")
}

var cacheStoresContent = func(dataFetcher fetcher.Fetcher, _ clusterSummaryInfo) ([]string, error) {
	var (
		cacheStoreResult  []byte
		cacheStoreDetails = config.CacheStoreDetails{}
		err               error
	)

	if selectedCache == "" {
		return emptyStringArray, errSelectCache
	}

	if serviceName, err = findServiceForCacheOrTopic(dataFetcher, selectedCache, "cache"); err != nil {
		return emptyStringArray, err
	}

	cacheStoreResult, err = dataFetcher.GetCacheMembers(serviceName, selectedCache)
	if err != nil {
		return emptyStringArray, err
	}

	if contentIsEmpty(cacheStoreResult) {
		return noContentArray, nil
	}

	if err = json.Unmarshal(cacheStoreResult, &cacheStoreDetails); err != nil {
		return emptyStringArray, err
	}

	finalDetails := ensureTierBack(cacheStoreDetails.Details)

	return strings.Split(FormatCacheStoreDetails(finalDetails, selectedCache, serviceName, false), "\n"), nil
}

var cachePartitionContent = func(dataFetcher fetcher.Fetcher, _ clusterSummaryInfo) ([]string, error) {
	return getCacheContent(dataFetcher, partitionDisplayType)
}

func getCacheContent(dataFetcher fetcher.Fetcher, displayType string) ([]string, error) {
	var (
		cacheResult           []byte
		cacheDetails          = config.CacheDetails{}
		cachePartitionDetails = config.CachePartitionDetails{}
		err                   error
		sb                    strings.Builder
	)

	if serviceName, err = findServiceForCacheOrTopic(dataFetcher, selectedCache, "cache"); err != nil {
		return emptyStringArray, err
	}

	if selectedCache == "" {
		return emptyStringArray, errSelectCache
	}

	if displayType == partitionDisplayType {
		cacheResult, err = dataFetcher.GetCachePartitions(serviceName, selectedCache)
	} else {
		cacheResult, err = dataFetcher.GetCacheMembers(serviceName, selectedCache)
	}
	if err != nil {
		return emptyStringArray, err
	}

	if contentIsEmpty(cacheResult) {
		return emptyStringArray, fmt.Errorf(cannotFindCache, selectedCache, serviceName)
	}

	if displayType == partitionDisplayType {
		err = json.Unmarshal(cacheResult, &cachePartitionDetails)
	} else {
		err = json.Unmarshal(cacheResult, &cacheDetails)
	}
	if err != nil {
		return emptyStringArray, err
	}

	if displayType == access {
		sb.WriteString(FormatCacheDetailsSizeAndAccess(cacheDetails.Details))
	} else if displayType == "index" {
		sb.WriteString(FormatCacheIndexDetails(cacheDetails.Details))
	} else if displayType == storage {
		sb.WriteString(FormatCacheDetailsStorage(cacheDetails.Details))
	} else if displayType == partitionDisplayType {
		sb.WriteString(FormatCachePartitions(cachePartitionDetails.Details, cacheSummary))
	}
	return strings.Split(sb.String(), "\n"), nil
}

var cachesContent = func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	cachesData, err := formatCachesSummary(clusterSummary.serviceList, dataFetcher)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(cachesData, "\n"), nil
}

var topicsContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	return strings.Split(FormatTopicsSummary(clusterSummary.topicsDetails.Details), "\n"), nil
}

var topicMembersContent = func(dataFetcher fetcher.Fetcher, _ clusterSummaryInfo) ([]string, error) {
	var (
		err                 error
		selectedDetails     config.TopicDetails
		topicsMemberDetails []config.TopicsMemberDetail
	)

	if selectedTopic == "" {
		return emptyStringArray, errSelectTopic
	}

	selectedDetails, err = getSelectedDetails(dataFetcher)
	if err != nil {
		return emptyStringArray, err
	}

	topicsMemberDetails, err = getTopicsMembers(dataFetcher, selectedDetails)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(FormatTopicsMembers(topicsMemberDetails), "\n"), nil
}

var topicSubscribersContent = func(dataFetcher fetcher.Fetcher, _ clusterSummaryInfo) ([]string, error) {
	var (
		err                     error
		selectedDetails         config.TopicDetails
		topicsSubscriberDetails []config.TopicsSubscriberDetail
	)

	if selectedTopic == "" {
		return emptyStringArray, errSelectTopic
	}

	selectedDetails, err = getSelectedDetails(dataFetcher)
	if err != nil {
		return emptyStringArray, err
	}

	topicsSubscriberDetails, err = getTopicsSubscribers(dataFetcher, selectedDetails)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(FormatTopicsSubscribers(topicsSubscriberDetails), "\n"), nil
}

var topicSubscriberGroupsContent = func(dataFetcher fetcher.Fetcher, _ clusterSummaryInfo) ([]string, error) {
	var (
		err                    error
		selectedDetails        config.TopicDetails
		subscriberGroupDetails []config.TopicsSubscriberGroupDetail
	)

	if selectedTopic == "" {
		return emptyStringArray, errSelectTopic
	}

	selectedDetails, err = getSelectedDetails(dataFetcher)
	if err != nil {
		return emptyStringArray, err
	}

	subscriberGroupDetails, err = getTopicsSubscriberGroups(dataFetcher, selectedDetails)
	if err != nil {
		return emptyStringArray, err
	}

	return strings.Split(FormatTopicsSubscriberGroups(subscriberGroupDetails), "\n"), nil
}

var topicSubscriberChannelsContent = func(dataFetcher fetcher.Fetcher, _ clusterSummaryInfo) ([]string, error) {
	var (
		err                     error
		selectedDetails         config.TopicDetails
		topicsSubscriberDetails []config.TopicsSubscriberDetail
	)

	if subscriber <= 0 {
		return emptyStringArray, errSelectSubscriber
	}

	selectedDetails, err = getSelectedDetails(dataFetcher)
	if err != nil {
		return emptyStringArray, err
	}

	topicsSubscriberDetails, err = getTopicsSubscribers(dataFetcher, selectedDetails)
	if err != nil {
		return emptyStringArray, err
	}

	nodeIndex := getSubscriberNodeIndex(topicsSubscriberDetails)

	if nodeIndex == -1 {
		return emptyStringArray, fmt.Errorf(subscriberMessage, 0, selectedTopic, serviceName, subscriber)
	}

	return strings.Split(FormatSubscriberChannelStats(generateSubscriberChannelStats(topicsSubscriberDetails[nodeIndex].Channels)), "\n"), nil
}

func getSelectedDetails(dataFetcher fetcher.Fetcher) (config.TopicDetails, error) {
	var (
		err             error
		selectedDetails config.TopicDetails
	)

	if serviceName, err = findServiceForCacheOrTopic(dataFetcher, selectedTopic, "topic"); err != nil {
		return selectedDetails, err
	}

	return getTopicsDetails(dataFetcher, serviceName, selectedTopic)
}

// Cluster Summary

var summaryClusterContent = func(_ fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error) {
	var cluster config.Cluster
	err := json.Unmarshal(clusterSummary.clusterResult, &cluster)
	if err != nil && !ignoreRESTErrors {
		return emptyStringArray, err
	}
	summary := fmt.Sprintf("Cluster Name:  %s\nVersion:       %v\nLicense Mode:  %v\nMembers:       %v\nDeparted:      %v\nMachines:      %v",
		cluster.ClusterName, cluster.Version, cluster.LicenseMode, cluster.ClusterSize,
		cluster.MembersDepartureCount, len(clusterSummary.machines))

	return strings.Split(summary, "\n"), nil
}

//
// Panel utility functions and types
//

// contentFunction generates content for a panel.
type contentFunction func(dataFetcher fetcher.Fetcher, clusterSummary clusterSummaryInfo) ([]string, error)

type panelImpl struct {
	PanelName         string
	OriginalMaxHeight int
	MaxHeight         int
	Title             string
	ContentFunction   contentFunction
	Description       string
	BaseData          []string
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
func createContentPanel(maxHeight int, panelName, title, description string, f contentFunction, baseData ...string) panelImpl {
	return panelImpl{
		MaxHeight:         maxHeight,
		OriginalMaxHeight: maxHeight,
		PanelName:         panelName,
		Title:             title,
		ContentFunction:   f,
		Description:       description,
		BaseData:          baseData,
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

func getPanel(panelName string) *panelImpl {
	for _, panel := range validPanels {
		if panel.GetPanelName() == panelName {
			return &panel
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
				return fmt.Errorf(unableToFindPanel, vv)
			}
		}
	}
	return nil
}

func getAllBaseData(layout []string) []string {
	allPanelData := make([]string, 0)

	for _, v := range layout {
		// split by "," for multiple per line
		s := strings.Split(v, ",")

		for _, vv := range s {
			panel := getPanel(vv)
			for _, b := range panel.BaseData {
				if !utils.SliceContains(allPanelData, b) {
					allPanelData = append(allPanelData, b)
				}
			}
		}
	}
	return allPanelData
}

// drawContent draws content and returns the height it drew
func drawContent(screen tcell.Screen, dataFetcher fetcher.Fetcher, panel panelImpl, x, y, w int, code rune) (int, error) {
	h := panel.GetMaxHeight()
	title := panel.GetTitle()

	if setMaxHeight != 0 {
		h = setMaxHeight
	}

	content, err := panel.GetContentFunction()(dataFetcher, lastClusterSummaryInfo)
	if err != nil {
		if ignoreRESTErrors {
			content = noContentArray
		} else {
			return 0, err
		}
	}

	l := len(content)

	if (l == 0 || l == 1) && content[0] == "" {
		content = noContentArray
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
		if len(line) >= w {
			content[i] = line[:w-2]
		}
	}

	trimmedText := ""
	if trimmed {
		trimmedText = fmt.Sprintf("%v%s", string(tcell.RuneHLine), "(trimmed)")
	}

	drawBox(screen, x, y, x+w-1, y+h, boxStyle, fmt.Sprintf("%s[%v]%s", parseTitle(title), string(code), trimmedText))

	for line := 1; line <= rows; line++ {
		drawText(screen, x+1, y+line, x+w-1, y+h-1, textStyle, content[line-1])
	}

	return rows + 2, nil
}

func parseTitle(title string) string {
	s := strings.ReplaceAll(title, topicNameToken, selectedTopic)
	s = strings.ReplaceAll(s, cacheNameToken, selectedCache)
	s = strings.ReplaceAll(s, serviceNameToken, serviceName)
	return strings.ReplaceAll(s, subscriberToken, fmt.Sprintf("%v", subscriber))
}

// trimBlankContent trims blank content at the end of the row.
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
func drawHeader(screen tcell.Screen, w, h int, cluster config.Cluster, dataFetcher fetcher.Fetcher) {
	var (
		title   string
		padding = " "
		height  = "0"
	)
	if cluster.ClusterName == "" && ignoreRESTErrors {
		title = errorContent + " from " + dataFetcher.GetURL()
	} else {
		version := strings.Split(cluster.Version, " ")
		if padMaxHeightParam {
			padding = "P"
		}
		if heightAdjust <= 0 {
			height = " 0 "
		} else if heightAdjust > 0 {
			height = fmt.Sprintf("+%v ", heightAdjust)
		}
		title = fmt.Sprintf("Coherence CLI: %s - %s (%s) ESC to quit %s. %s%s(%v)",
			time.Now().Format(time.DateTime), cluster.ClusterName, version[0], additionalMonitorMsg, padding, height, lastDuration)
		titleLen := len(title)
		if titleLen < w-2 {
			title = fmt.Sprintf("%s%-*s", title, w-titleLen-2, " ")
		}
	}
	drawText(screen, 1, 0, w-1, h-1, textStyle.Reverse(true), title)
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

	drawText(s, x1+2, y1, x2-1, y2-1, titleStyle, title)
}

// getValidPanelTypes returns the list of panels for the --help command.
func getValidPanelTypes() string {
	var sb strings.Builder
	sb.WriteString("Default panels\n--------------\n")
	for k, v := range defaultMap {
		sb.WriteString(fmt.Sprintf("%-22s: %s\n", k, v))
	}

	sb.WriteString("\nIndividual panels\n-----------------\n")
	for _, p := range validPanels {
		sb.WriteString(fmt.Sprintf("%-22s: %s\n", p.GetPanelName(), p.GetDescription()))
	}

	return sb.String()
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

func contentIsEmpty(data []byte) bool {
	return len(data) == 0 || string(data) == "{}"
}

func init() {
	monitorClusterCmd.Flags().StringVarP(&layoutParam, "layout", "l", defaultLayoutName, "layout to use")
	monitorClusterCmd.Flags().BoolVarP(&showAllPanels, "show-panels", "", false, "show all available panels")
	monitorClusterCmd.Flags().BoolVarP(&ignoreRESTErrors, "ignore-errors", "I", false, "ignore errors after initial refresh")
	monitorClusterCmd.Flags().BoolVarP(&disablePadding, "disable-padding", "D", false, "disable padding of panels by default")
	monitorClusterCmd.Flags().StringVarP(&serviceName, serviceNameOption, "S", "", serviceNameDescription)
	monitorClusterCmd.Flags().StringVarP(&selectedCache, "cache-name", "C", "", "cache name")
	monitorClusterCmd.Flags().StringVarP(&selectedTopic, "topic-name", "T", "", "topic name")
	monitorClusterCmd.Flags().StringVarP(&colorStyleParam, "style", "", "", "color style")
	monitorClusterCmd.Flags().Int64VarP(&subscriber, "subscriber-id", "B", 0, "subscriber")
	monitorClusterCmd.Flags().IntVarP(&setMaxHeight, "max-height", "M", 0, "override max height for all panels")
	monitorClusterCmd.Flags().BoolVarP(&ignoreSpecialCaches, "ignore-special", "", false, ignoreCachesDescription)
	monitorClusterCmd.Flags().BoolVarP(&previewStylesParam, "preview-styles", "", false, "preview all the styles")
}
