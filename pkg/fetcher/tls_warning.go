/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package fetcher

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
)

type tlsWarningKey struct {
	connection string
	origin     httpsOrigin
}

type tlsWarningExecution struct {
	mu     sync.Mutex
	seen   map[tlsWarningKey]bool
	stderr io.Writer
}

var commandTLSWarnings atomic.Pointer[tlsWarningExecution]

func init() {
	BeginCommandTLSWarnings(os.Stderr)
}

// BeginCommandTLSWarnings starts warning suppression for one command execution.
// Call before issuing requests. Command execution/initialization is serial; the
// request path shares this state safely across concurrent, by-value fetchers.
func BeginCommandTLSWarnings(stderr io.Writer) {
	if stderr == nil {
		stderr = os.Stderr
	}
	commandTLSWarnings.Store(&tlsWarningExecution{seen: make(map[tlsWarningKey]bool), stderr: stderr})
}

// recordTLSBypass is called only when the actual request's target TLS config
// disables verification. Build diagnostics from the normalized bound origin,
// never from the full URL, and quote fields to prevent terminal/log injection.
func (h HTTPFetcher) recordTLSBypass() {
	connection := strconv.QuoteToASCII(h.connectionName)
	destination := url.URL{Scheme: "https", Host: net.JoinHostPort(h.insecureOrigin.hostname, h.insecureOrigin.port)}
	origin := strconv.QuoteToASCII(destination.String())
	state := commandTLSWarnings.Load()
	key := tlsWarningKey{connection: h.connectionName, origin: h.insecureOrigin}
	state.mu.Lock()
	if !state.seen[key] {
		_, _ = fmt.Fprintf(state.stderr, "WARNING: TLS certificate verification is disabled for connection %s at %s.\n", connection, origin)
		state.seen[key] = true
	}
	state.mu.Unlock()

	// Every attempt is recorded, including failed requests and requests whose
	// terminal warning was already emitted. Production initializes Logger before use.
	if Logger != nil {
		Logger.Warn("TLS certificate verification disabled", zap.String("connection", connection), zap.String("origin", origin))
	}
}
