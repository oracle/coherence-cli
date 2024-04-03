<!--
Copyright (c) 2021, 2024 Oracle and/or its affiliates.
Licensed under the Universal Permissive License v 1.0 as shown at
https://oss.oracle.com/licenses/upl.
-->

-----
![logo](docs/images/logo-with-name.png)

# Development and Testing

This document outlines the development process for the Coherence CLI.

## Contents
* [Building the CLI](#building-the-cli)
* [Testing](#testing)

## Building the CLI

This section outlines how to build the CLI.

### Pre-Requisites

1. Install Go version 1.20 from [https://golang.org/doc/install](https://golang.org/doc/install).
2. Maven 3.6.3+
3. JDK 1.8+

> Note: Java and Maven are only required if you wish to run the tests.

### Build for local platform

```bash
git clone https://github.com/oracle/coherence-cli.git
cd coherence-cli
make cohctl
# Add to your PATH
export PATH=`pwd`/bin:$PATH
```

Test by running the following:

```bash
cohctl version
```

### Build for all platforms

```bash
make cohctl-all
```

The binaries for all supported platforms are available in the following directories:

* `bin/linux/amd64/cohctl`
* `bin/linux/arm64/cohctl`
* `bin/linux/386/cohctl`
* `bin/windows/amd64/cohctl.exe`
* `bin/windows/arm/cohctl.exe`
* `bin/darwin/amd64/cohctl`
* `bin/darwin/arm64/cohctl`

## Testing

### Pre-Requisites

1. Docker must be running
2. Docker Compose must be available in the path

### Unit Tests

```bash
make clean test-cohctl
```

### Run E2E tests

Runs all end-to-end tests for the commands against a two node Coherence CE cluster using the latest
Coherence CE version. This will automatically start a cluster using docker.compose.

```bash
$ make clean test-e2e-standalone 
```

### Run Discovery tests

Runs discovery tests only

```bash
make clean test-discovery 
```

### Run Compatability tests CE

```bash
COHERENCE_VERSION=21.12.3    make clean build-test-images test-e2e-standalone
COHERENCE_VERSION=14.1.1-0-7 make clean build-test-images test-e2e-standalone
```

For 21.12+ you must use the following to enable executors

```bash
PROFILES=,executor COHERENCE_VERSION=21.12.3 make clean build-test-images test-e2e-standalone
```

### Run Compatability tests Commercial

> Note: You must have the Coherence Commercial versions in your local Maven repository.

```bash
PROFILES=,commercial COHERENCE_VERSION=14.1.1-0-6  COHERENCE_GROUP_ID=com.oracle.coherence make clean build-test-images test-e2e-standalone
PROFILES=,commercial COHERENCE_VERSION=12.2.1-4-10 COHERENCE_GROUP_ID=com.oracle.coherence make clean build-test-images test-e2e-standalone
```
