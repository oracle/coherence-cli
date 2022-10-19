/*
 * Copyright (c) 2021, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package federation

import (
	"fmt"
	"github.com/oracle/coherence-cli/test/test_utils"
	"os"
	"testing"
)

// The entry point for the test suite
func TestMain(m *testing.M) {
	var (
		err      error
		exitCode int
		httpPort = 30000
		restPort = 8080
	)

	context := test_utils.TestContext{ClusterName: "cluster1", HttpPort: httpPort,
		Url: test_utils.GetManagementUrl(httpPort), ExpectedServers: 1, RestUrl: test_utils.GetRestUrl(restPort)}
	test_utils.SetTestContext(&context)

	var fileName = test_utils.GetFilePath("docker-compose-2-members.yaml")

	err = test_utils.StartCoherenceCluster(fileName, context.Url)
	if err != nil {
		fmt.Println(err)
		exitCode = 1
	} else {
		// don't check for balanced federation services as there is only 1 member per cluster
		test_utils.Sleep(10)
		exitCode = m.Run()
	}

	fmt.Printf("Tests completed with return code %d\n", exitCode)
	if exitCode != 0 {
		// collect logs from docker images
		_ = test_utils.CollectDockerLogs()
	}
	_, _ = test_utils.DockerComposeDown(fileName)
	os.Exit(exitCode)
}
