//go:build darwin || linux

/*
 * Copyright (c) 2022, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"os/exec"
	"syscall"
)

// setForkProcess set the process to be forked for linux
func setForkProcess(process *exec.Cmd) {
	process.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
}

//func handleCTRLC() {
//	chanSignal := make(chan os.Signal, 1)
//	chanDone := make(chan struct{})
//	signal.Notify(chanSignal, os.Interrupt)
//	go func() {
//		<-chanSignal
//		fmt.Println("CTRL-C Received")
//		close(chanDone)
//	}()
//	<-chanDone
//}
