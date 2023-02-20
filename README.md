<!--
Copyright (c) 2021, 2023 Oracle and/or its affiliates.
Licensed under the Universal Permissive License v 1.0 as shown at
https://oss.oracle.com/licenses/upl.
-->

-----
![logo](docs/images/logo-with-name.png)

![Coherence CI](https://github.com/oracle/coherence-cli/workflows/CI/badge.svg?branch=main)
[![License](http://img.shields.io/badge/license-UPL%201.0-blue.svg)](https://oss.oracle.com/licenses/upl/)

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=oracle_coherence-cli&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=oracle_coherence-cli)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=oracle_coherence-cli&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=oracle_coherence-cli)

[![Go Report Card](https://goreportcard.com/badge/github.com/oracle/coherence-cli)](https://goreportcard.com/report/github.com/oracle/coherence-cli)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/oracle/coherence-cli)

# Coherence Command Line Interface (CLI)

![Coherence Demo](assets/cohctl-terminal.gif "Coherence CLI Demo")

## Contents

* [Overview](#overview)
* [Why use the Coherence CLI?](#why-use-the-coherence-cli)
* [Install the CLI](#install-the-cli)
* [Getting Started](#getting-started)
* [Supported Coherence Versions](#supported-coherence-versions)
* [Get Involved](#need-more-help-have-a-suggestion-come-and-say-hello)

## Overview 

The Coherence command line interface, `cohctl`, is a lightweight tool, in the tradition of tools such as kubectl,
which can be scripted or used interactively to manage and monitor Coherence clusters. You can use `cohctl` to view cluster information
such as services, caches, members, etc, as well as perform various management operations against clusters.

The CLI accesses clusters using the HTTP Management over REST interface and therefore requires this to be enabled on any clusters
you want to monitor or manage. See the [Coherence Documentation](https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/rest-reference/index.html)
for more information on setting up Management over REST.

NOTE: The CLI does not replace current management and monitoring tools such as the [Coherence VisualVM Plugin](https://github.com/oracle/coherence-visualvm),
[Enterprise Manager](https://docs.oracle.com/cd/E24628_01/install.121/e24215/coherence_getstarted.htm#GSSOA10121), or [Grafana Dashboards](https://oracle.github.io/coherence-operator/docs/latest/#/docs/metrics/040_dashboards), but complements and
provides a lightweight and scriptable alternative.

## Why use the Coherence CLI?

The CLI complements your existing Coherence management tools and allows you to:

* Interactively monitor your Coherence clusters from a lightweight terminal-based interface
* Monitor service "StatusHA" during rolling restarts of Coherence clusters
* Script Coherence monitoring and incorporate results into other management tooling
* Output results in various formats including text, JSON and utilize JsonPath to extract attributes of interest
* Gather information that may be useful for Oracle Support to help diagnose issues
* Connect to standalone or WebLogic Server based clusters from commercial versions 12.2.1.4 and above as well as the latest [Coherence Community Edition](https://github.com/oracle/coherence) (CE) versions
* Retrieve thread dumps across members
* Create, start and scale development clusters (experimental)

## Install the CLI

The Coherence CLI is available for macOS, Linux and Windows for x86 and Arm based processors.

### Mac / Linux

For macOS and Linux platforms, use the following to install the latest version of the CLI:

```bash
curl -sL https://raw.githubusercontent.com/oracle/coherence-cli/main/install.sh | bash
```

> Note: On linux, `sudo` is required to copy `cohctl` to the destination directory `/usr/local/bin/`

### Windows

For Windows, use the curl command below, then copy `cohctl.exe` to a directory in your PATH:

```cmd
curl -sLo cohctl.exe "https://github.com/oracle/coherence-cli/releases/download/1.4.2/cohctl-1.4.2-windows-amd64.exe"
```

> Note: Change the **amd64** to **arm** for ARM based processor in the URL above.

See the [Install guide](https://oracle.github.io/coherence-cli/docs/latest/#/docs/installation/01_installation) for
more information on downloading and installing the CLI.

## Getting Started

Documentation for the Coherence CLI is available [here](https://oracle.github.io/coherence-cli/docs/latest)

The fastest way to experience the Coherence CLI is to follow the
[Quick Start guide](https://oracle.github.io/coherence-cli/docs/latest/#/docs/about/03_quickstart).

## Supported Coherence Versions

The CLI supports and is certified against the following Community and Commercial editions of Coherence:

**Coherence Community Edition**
* 22.09
* 22.06.x
* 21.12.x
* 14.1.1-0-11+

**Coherence Grid/ Enterprise Edition**
* 12.2.1.4.x - minimum patch level of 12.2.1.4.10+ required
* 14.1.1.0.x - minimum patch level of 14.1.1.0.5+ required
* 14.1.1.2206.x Feature Pack

> Note: If you are on a patch set below the minimum recommended above, then CLI may work, but some functionality may not be available. It
> is always recommended to upgrade to the latest Coherence patch as soon as you are able to.

## Need more help? Have a suggestion? Come and say "Hello!"

We have a **public Slack channel** where you can get in touch with us to ask questions about using the Coherence CLI
or give us feedback or suggestions about what features and improvements you would like to see. We would love
to hear from you. To join our channel,
please [visit this site to get an invitation](https://join.slack.com/t/oraclecoherence/shared_invite/enQtNzcxNTQwMTAzNjE4LTJkZWI5ZDkzNGEzOTllZDgwZDU3NGM2YjY5YWYwMzM3ODdkNTU2NmNmNDFhOWIxMDZlNjg2MzE3NmMxZWMxMWE).  
The invitation email will include details of how to access our Slack
workspace.  After you are logged in, please come to `#cli` and say, "hello!"

