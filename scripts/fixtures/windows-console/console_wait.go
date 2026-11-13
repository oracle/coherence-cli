// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type consoleAttachment struct {
	Attempts int
	Exited   bool // The existing Wait result was consumed, even if it was nil.
}

// waitForOwnedConsole retries only ERROR_INVALID_HANDLE: the exact child has
// no console yet. AttachConsole success, not elapsed time, proves attachment.
// The caller retains the child's process handle and is the sole rescue owner.
func waitForOwnedConsole(ctx context.Context, done <-chan error, retry <-chan time.Time, attach func() error) (result consoleAttachment, err error) {
	var last error
	exited := func(waitErr error) (consoleAttachment, error) {
		result.Exited = true
		if waitErr != nil {
			return result, fmt.Errorf("owned CLI exited before console attachment: %w", waitErr)
		}
		return result, errors.New("owned CLI exited with status 0 before console attachment")
	}
	for {
		// Give an already observed exit/deadline priority over a new API call.
		select {
		case waitErr := <-done:
			return exited(waitErr)
		default:
		}
		if ctx.Err() != nil {
			return result, errors.Join(ctx.Err(), last)
		}
		result.Attempts++
		last = attach()
		if last == nil {
			return result, nil
		}
		if !errors.Is(last, errInvalidHandle) {
			return result, last // Keep terminal and unknown causes; never retry them.
		}
		select {
		case waitErr := <-done:
			return exited(waitErr)
		case <-ctx.Done():
			return result, errors.Join(ctx.Err(), last)
		case <-retry:
		}
	}
}
