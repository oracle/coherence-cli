/*
 * Copyright (c) 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package proc

type ProcessExecution interface {
	CreateProcess(command ...string) ExecutionRequest
	SetLogfile()
}

type ExecutionRequest struct {
	Command    string
	Arguments  []string
	LogFile    string
	ReturnCode int
}

func StartProcess(command ...string, logFileName) {

}