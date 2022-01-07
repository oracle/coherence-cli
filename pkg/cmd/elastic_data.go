/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"encoding/json"
	"errors"
	"github.com/oracle/coherence-cli/pkg/config"
	"github.com/oracle/coherence-cli/pkg/constants"
	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/oracle/coherence-cli/pkg/utils"
	"github.com/spf13/cobra"
	"strings"
	"sync"
	"time"
)

var (
	ElasticDataMessage = "name must be FlashJournalRM or RamJournalRM"
	errorInvalidType   = errors.New(ElasticDataMessage)
)

// getElasticDataCmd represents the get elastic-data command
var getElasticDataCmd = &cobra.Command{
	Use:   "elastic-data",
	Short: "display elastic data information for a cluster",
	Long: `The 'get elastic-data' command displays the Flash Journal and RAM
Journal details for the cluster.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			dataFetcher fetcher.Fetcher
			finalResult string
			connection  string
			err         error
			wg          sync.WaitGroup
			errorSink   ErrorSink
			ramResult   []byte
			flashResult []byte
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		for {
			if watchEnabled {
				cmd.Println("\n" + time.Now().String())
			}

			errorSink = createErrorSink()
			wg.Add(2)

			go func() {
				defer wg.Done()
				var err1 error
				ramResult, err1 = dataFetcher.GetElasticDataDetails("ram")
				if err1 != nil {
					errorSink.AppendError(err1)
				}
			}()

			go func() {
				defer wg.Done()
				var err1 error
				flashResult, err1 = dataFetcher.GetElasticDataDetails("flash")
				if err1 != nil {
					errorSink.AppendError(err1)
				}
			}()

			wg.Wait()
			errorList := errorSink.GetErrors()
			if len(errorList) > 0 {
				return utils.GetErrors(errorList)
			}

			if strings.Contains(OutputFormat, constants.JSONPATH) || OutputFormat == constants.JSON {
				finalResult, err := utils.CombineByteArraysForJSON([][]byte{flashResult, ramResult},
					[]string{constants.FlashJournal, constants.RAMJournal})
				if err != nil {
					return err
				}
				if strings.Contains(OutputFormat, constants.JSONPATH) {
					result, err := utils.GetJSONPathResults(finalResult, OutputFormat)
					if err != nil {
						return err
					}
					cmd.Println(result)
				} else {
					cmd.Println(string(finalResult))
				}
			} else {
				cmd.Println(FormatCurrentCluster(connection))
				finalResult, err = getElasticDataResult(flashResult, ramResult)
				if err != nil {
					return err
				}
				if finalResult == "" {
					return nil
				}
				cmd.Println(finalResult)
			}

			// check to see if we should exit if we are not watching
			if !watchEnabled {
				break
			}
			// we are watching services so sleep and then repeat until CTRL-C
			time.Sleep(time.Duration(watchDelay) * time.Second)
		}

		return nil
	},
}

// describeElasticDataCmd represents the describe elastic-data command
var describeElasticDataCmd = &cobra.Command{
	Use:   "elastic-data {FlashJournalRM|RamJournalRM}",
	Short: "describe a flash our ram journal",
	Long: `The 'describe elastic-data' command shows information related to a specific journal type.
The allowable values are RamJournalRM or FlashJournalRM.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			displayErrorAndExit(cmd, "you must provide either FlashJournalRM or RamJournalRM")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			queryType   string
			result      []byte
			err         error
			header      string
			dataFetcher fetcher.Fetcher
			edValues    = config.ElasticDataValues{}
			connection  string
			journalType = args[0]
		)

		connection, dataFetcher, err = GetConnectionAndDataFetcher()
		if err != nil {
			return err
		}

		if journalType == "RamJournalRM" {
			queryType = "ram"
			header = "RAM JOURNAL DETAILS"
		} else if journalType == "FlashJournalRM" {
			queryType = "flash"
			header = "FLASH JOURNAL DETAILS"
		} else {
			return errorInvalidType
		}

		result, err = dataFetcher.GetElasticDataDetails(queryType)
		if err != nil {
			return err
		}

		if strings.Contains(OutputFormat, constants.JSONPATH) {
			jsonPathResult, err := utils.GetJSONPathResults(result, OutputFormat)
			if err != nil {
				return err
			}
			cmd.Println(jsonPathResult)
			return nil
		} else if OutputFormat == constants.JSON {
			cmd.Println(string(result))
		} else {
			if len(result) == 0 {
				return nil
			}

			err = json.Unmarshal(result, &edValues)
			if err != nil {
				return utils.GetError("unable to elastic data details", err)
			}
			cmd.Println(FormatCurrentCluster(connection))

			cmd.Println(header)
			headerLen := len(header)
			underline := make([]byte, headerLen)
			for i := 0; i < headerLen; i++ {
				underline[i] = '-'
			}
			cmd.Println(string(underline))

			cmd.Println(FormatElasticData(edValues.ElasticData, false))
		}

		return nil
	},
}

// getElasticDataResult returns elastic data results
func getElasticDataResult(flashResult, ramResult []byte) (string, error) {
	var (
		flashValues = config.ElasticDataValues{}
		ramValues   = config.ElasticDataValues{}
		err         error
	)

	if len(flashResult) > 0 {
		err = json.Unmarshal(flashResult, &flashValues)
		if err != nil {
			return "", utils.GetError("unable to flash details", err)
		}
	}
	if len(ramResult) > 0 {
		err = json.Unmarshal(ramResult, &ramValues)
		if err != nil {
			return "", utils.GetError("unable to ram details", err)
		}
	}

	// get combined results
	combinedFlash := combineElasticData(flashValues)
	combinedRAM := combineElasticData(ramValues)

	// only display if details exist
	var (
		lenRAM   = len(combinedRAM)
		lenFlash = len(combinedFlash)
	)
	if lenRAM == 1 || lenFlash == 1 {
		finalEd := make([]config.ElasticData, 0)
		if lenFlash == 1 {
			finalEd = append(finalEd, combinedFlash[0])
		}
		if lenRAM == 1 {
			finalEd = append(finalEd, combinedRAM[0])
		}
		return FormatElasticData(finalEd, true), nil
	}
	return "", nil
}

// combineElasticData combines all the elastic data details together to give a summary
func combineElasticData(elasticData config.ElasticDataValues) []config.ElasticData {
	var finalData = make([]config.ElasticData, 0)

	for _, value := range elasticData.ElasticData {
		if len(finalData) == 0 {
			// no entries so add it anyway
			finalData = append(finalData, value)
		} else {
			var foundIndex = -1
			for i, v := range finalData {
				if v.Type == value.Type {
					foundIndex = i
					break
				}
			}

			if foundIndex >= 0 {
				// update the existing elastic Data
				finalData[foundIndex].FileCount += value.FileCount
				finalData[foundIndex].MaxJournalFilesNumber += value.MaxJournalFilesNumber
				finalData[foundIndex].TotalDataSize += value.TotalDataSize
				finalData[foundIndex].CompactionCount += value.CompactionCount
				finalData[foundIndex].ExhaustiveCompactionCount += value.ExhaustiveCompactionCount
				if value.HighestLoadFactor > finalData[foundIndex].HighestLoadFactor {
					finalData[foundIndex].HighestLoadFactor = value.HighestLoadFactor
				}
			} else {
				finalData = append(finalData, value)
			}
		}
	}

	return finalData
}
