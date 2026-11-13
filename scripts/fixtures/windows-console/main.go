// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

//go:build windows

// Native console controller and Java-shaped process fixture. No Coherence claim.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) == 3 && os.Args[1] == "--control" {
		if os.Args[2] != requestName {
			panic("fixed private request name required")
		}
		if err := control(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	// exec.Command("mvn", ...) resolves mvn.exe but retains "mvn" as argv[0].
	name := filepath.Base(os.Args[0])
	if strings.EqualFold(name, "mvn") || strings.EqualFold(name, "mvn.exe") {
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "-version" {
		fmt.Fprintln(os.Stderr, `openjdk version "17.0.21"`)
		return
	}
	root, err := openExecutableTaskRoot()
	if err != nil {
		panic(err)
	}
	defer root.Close()
	memberDirectory := os.Getenv("COHCTL_WINDOWS_MEMBER_DIR")
	pid := os.Getpid()
	if err := writeFixtureMarker(root, memberDirectory, fmt.Sprintf("child-%d", pid), []byte(fmt.Sprint(pid))); err != nil {
		panic(err)
	}
	port := "-1"
	cluster := ""
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-Dcoherence.management.http.port=") {
			port = strings.TrimPrefix(arg, "-Dcoherence.management.http.port=")
		}
		if strings.HasPrefix(arg, "-Dcoherence.cluster=") {
			cluster = strings.TrimPrefix(arg, "-Dcoherence.cluster=")
		}
	}
	if port == "-1" {
		time.Sleep(time.Minute)
		return
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:"+port)
	if err != nil {
		panic(err)
	}
	server := &http.Server{ReadHeaderTimeout: time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 2 * time.Second,
		Handler: fixtureIdentityHandler(root, memberDirectory, cluster, pid, os.Getenv("COHCTL_WINDOWS_READY") == "1")}
	rescue := time.AfterFunc(time.Minute, func() { _ = server.Close() })
	defer rescue.Stop()
	defer listener.Close()
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func openExecutableTaskRoot() (*os.Root, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	directory, err := executableTaskDirectory(executable)
	if err != nil {
		return nil, err
	}
	return openTaskRoot(directory)
}

func control() (resultErr error) {
	// Include request setup and console attachment in the independent lifetime.
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	root, err := openExecutableTaskRoot()
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	request, err := readConsoleRequest(root)
	if err != nil {
		return err
	}
	args, err := request.arguments(root.Name())
	if err != nil {
		return err
	}
	executable, err := approvedCLI(root)
	if err != nil {
		return err
	}
	output, err := openConsoleOutput(root)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, output.Close()) }()
	child := exec.Command(executable, args...)
	// Keep a private console and explicitly establish the CLI event group. Native
	// controls verify both targeted Ctrl-Break and console-wide Ctrl-C delivery.
	child.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x10 | syscall.CREATE_NEW_PROCESS_GROUP}
	child.Stdin, child.Stdout, child.Stderr = os.Stdin, output, output
	if err = child.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- child.Wait() }() // Sole Wait owner, including rescue.
	waited := false
	defer func() {
		if !waited {
			resultErr = errors.Join(resultErr, errors.New("controller required owned CLI rescue"))
			if err := child.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
				resultErr = errors.Join(resultErr, fmt.Errorf("kill owned CLI: %w", err))
			}
			rescue := time.NewTimer(5 * time.Second)
			defer rescue.Stop()
			select {
			case waitErr := <-done: // Receive the existing sole Wait result; never start another Wait.
				resultErr = errors.Join(resultErr, rescueWaitResult(waitErr))
			case <-rescue.C:
				resultErr = errors.Join(resultErr, errors.New("owned CLI Wait unresolved after 5s rescue"))
			}
		}
	}()
	kernel := syscall.NewLazyDLL("kernel32.dll")
	free := kernel.NewProc("FreeConsole")
	if err := detachConsole(func() (uintptr, uintptr, error) { return free.Call() }); err != nil {
		return err
	}
	attach := kernel.NewProc("AttachConsole")
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	attachCtx, attachCancel := context.WithTimeout(ctx, 5*time.Second)
	defer attachCancel()
	attachStarted := time.Now()
	attachment, err := waitForOwnedConsole(attachCtx, done, tick.C, func() error {
		attached, _, attachErr := attach.Call(uintptr(child.Process.Pid))
		if attached == 0 {
			return attachErr
		}
		return nil
	})
	attachElapsed := time.Since(attachStarted)
	waited = attachment.Exited
	if err != nil {
		return fmt.Errorf("AttachConsole owned CLI %d after %d attempts (%s): %w", child.Process.Pid, attachment.Attempts, attachElapsed, err)
	}
	defer func() {
		resultErr = errors.Join(resultErr, detachConsole(func() (uintptr, uintptr, error) { return free.Call() }))
	}()
	// Register AFTER attach (which clears handlers). Protect only this controller.
	handler := syscall.NewCallback(func(uint32) uintptr { return 1 })
	registered, _, handlerErr := kernel.NewProc("SetConsoleCtrlHandler").Call(handler, 1)
	if registered == 0 {
		return fmt.Errorf("SetConsoleCtrlHandler: %w", handlerErr)
	}
	sent := false
	for {
		select {
		case err = <-done:
			waited = true
			return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{"cli_pid": child.Process.Pid, "argv": child.Args, "event": request.Event, "sent": sent, "exit": child.ProcessState.ExitCode(), "wait_error": fmt.Sprint(err), "rescue": false, "attach_attempts": attachment.Attempts, "attach_elapsed_ms": float64(attachElapsed) / float64(time.Millisecond)})
		case <-ctx.Done():
			return fmt.Errorf("private CLI exceeded independent 45s rescue limit: %w", ctx.Err())
		case <-tick.C:
			if sent {
				continue
			}
			ready, err := consoleSignalReady(root)
			if err != nil {
				return fmt.Errorf("private signal observation: %w", err)
			}
			if ready {
				group := uintptr(0) // Ctrl-C is console-wide, confined to this private console.
				if request.Event == 1 {
					group = uintptr(child.Process.Pid) // Ctrl-Break targets the original CLI group.
				}
				ok, _, eventErr := kernel.NewProc("GenerateConsoleCtrlEvent").Call(uintptr(request.Event), group)
				if ok == 0 {
					return fmt.Errorf("GenerateConsoleCtrlEvent: %w", eventErr)
				}
				sent = true
			}
		}
	}
}
