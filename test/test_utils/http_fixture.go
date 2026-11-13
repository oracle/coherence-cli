// Copyright (c) 2026 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at
// https://oss.oracle.com/licenses/upl.

package test_utils

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// The test controller assigns origins through TestContext, never through a
// response or a requested URL. Remote origins are supported by the same exact
// scheme/host/effective-port contract as local Docker fixture origins.
func configuredHTTPOrigins() []string {
	if currentTestContext == nil {
		return nil
	}
	origins := []string{currentTestContext.Url, currentTestContext.RestUrl}
	return append(origins, currentTestContext.AdditionalHTTPOrigins...)
}

func fixtureOrigin(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Opaque != "" || u.Fragment != "" {
		return "", fmt.Errorf("invalid fixture HTTP URL %q", raw)
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || strings.ContainsAny(host, "%\\ \t\r\n") {
		return "", errors.New("invalid fixture host")
	}
	port := u.Port()
	if port == "" {
		if strings.HasSuffix(u.Host, ":") {
			return "", errors.New("empty fixture port")
		}
		port = "80"
		if u.Scheme == "https" {
			port = "443"
		}
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", errors.New("fixture port outside 1..65535")
	}
	return u.Scheme + "://" + net.JoinHostPort(host, strconv.Itoa(n)), nil
}

func fixtureGet(target string, assigned []string) (body []byte, resultErr error) {
	origin, err := fixtureOrigin(target)
	if err != nil {
		return emptyByte, err
	}
	allowed := false
	for _, raw := range assigned {
		if raw == "" {
			continue
		}
		candidate, err := fixtureOrigin(raw)
		if err != nil {
			return emptyByte, fmt.Errorf("invalid assigned fixture origin: %w", err)
		}
		allowed = allowed || candidate == origin
	}
	if !allowed {
		return emptyByte, fmt.Errorf("HTTP destination %s is not an assigned fixture origin", origin)
	}
	// A proxy would introduce an unassigned transport destination. Keep the normal
	// TLS verification, bound the request and reject redirects to another origin.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		next, err := fixtureOrigin(req.URL.String())
		if err != nil {
			return err
		}
		if next != origin {
			return errors.New("fixture redirect leaves assigned request origin")
		}
		if len(via) >= 10 {
			return errors.New("too many fixture redirects")
		}
		return nil
	}}
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return emptyByte, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return emptyByte, err
	}
	defer func() { resultErr = errors.Join(resultErr, resp.Body.Close()) }()
	if resp.StatusCode != http.StatusOK {
		return emptyByte, errors.New("Did not receive a 200 response code: " + resp.Status)
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return emptyByte, err
	}
	return body, nil
}
