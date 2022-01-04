# ----------------------------------------------------------------------------------------------------------------------
# Copyright (c) 2021, 2022 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#
# ----------------------------------------------------------------------------------------------------------------------
# This is the Makefile to build the Coherence Command Line Interface (CLI.
# ----------------------------------------------------------------------------------------------------------------------

# ======================================================================================================================
# Makefile Variables
#
# The following section contains all of the variables and properties used by other targets in the Makefile
# to set things like build directories, version numbers etc.
# ======================================================================================================================

# The version of the CLI being build - this should be a valid SemVer format
VERSION ?= 1.0.0
MILESTONE ?=

# Maven version is always 1.0.0 as it is only for testing
MVN_VERSION ?= 1.0.0

# Coherence CE version to run base tests against
COHERENCE_VERSION ?= 21.12
COHERENCE_GROUP_ID ?= com.oracle.coherence.ce
COHERENCE_WKA1 ?= server1
COHERENCE_WKA2 ?= server1
COHERENCE_CLUSTER1 ?= cluster1
COHERENCE_CLUSTER2 ?= cluster1
CLUSTER_PORT ?= 7574
# Profiles to include for building
PROFILES ?=
COHERENCE_BASE_IMAGE ?= gcr.io/distroless/java:11

# ----------------------------------------------------------------------------------------------------------------------
# Options to append to the Maven command
# ----------------------------------------------------------------------------------------------------------------------
MAVEN_OPTIONS ?= -Dmaven.wagon.httpconnectionManager.ttlSeconds=25 -Dmaven.wagon.http.retryHandler.count=3
MAVEN_BUILD_OPTS :=$(USE_MAVEN_SETTINGS) -Drevision=$(MVN_VERSION) -Dcoherence.version=$(COHERENCE_VERSION) -Dcoherence.group.id=$(COHERENCE_GROUP_ID) $(MAVEN_OPTIONS)

CURRDIR := $(shell pwd)

USER_ID := $(shell echo "`id -u`:`id -g`")

# ----------------------------------------------------------------------------------------------------------------------
# Build output directories
# ----------------------------------------------------------------------------------------------------------------------
override BUILD_OUTPUT        := $(CURRDIR)/build/_output
override BUILD_BIN           := $(CURRDIR)/bin
override BINARIES_DIR        := ./binaries
override BUILD_TARGETS       := $(BUILD_OUTPUT)/targets
override TEST_LOGS_DIR       := $(BUILD_OUTPUT)/test-logs
override COVERAGE_DIR        := $(BUILD_OUTPUT)/coverage
override BUILD_PROPS         := $(BUILD_OUTPUT)/build.properties
override BUILD_DOCS          := $(BUILD_OUTPUT)/docs-gen
override PKG_DIR             := $(BINARIES_DIR)
override INSTALLER_DIR       := ./installer
override BUILD_SHARED        := $(CURRDIR)/test/test_utils/shared
override ENV_FILE            := test/test_utils/.env
override COPYRIGHT_JAR       := glassfish-copyright-maven-plugin-2.4.jar

# ----------------------------------------------------------------------------------------------------------------------
# Set the location of various build tools
# ----------------------------------------------------------------------------------------------------------------------
TOOLS_DIRECTORY   = $(CURRDIR)/build/tools
TOOLS_BIN         = $(TOOLS_DIRECTORY)/bin

# ----------------------------------------------------------------------------------------------------------------------
# The test application images used in integration tests
# ----------------------------------------------------------------------------------------------------------------------
RELEASE_IMAGE_PREFIX     ?= ghcr.io/oracle/
TEST_APPLICATION_IMAGE_1 := $(RELEASE_IMAGE_PREFIX)coherence-cli-test-1:1.0.0
TEST_APPLICATION_IMAGE_2 := $(RELEASE_IMAGE_PREFIX)coherence-cli-test-2:1.0.0

# ----------------------------------------------------------------------------------------------------------------------
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
# ----------------------------------------------------------------------------------------------------------------------
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# ----------------------------------------------------------------------------------------------------------------------
# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
# ----------------------------------------------------------------------------------------------------------------------
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# ----------------------------------------------------------------------------------------------------------------------
# Capture the Git commit to add to the build information that is then embedded in the Go binary
# ----------------------------------------------------------------------------------------------------------------------
GITCOMMIT              ?= $(shell git rev-list -1 HEAD)
GITREPO                := https://github.com/oracle/coherence-cli.git
SOURCE_DATE_EPOCH      := $(shell git show -s --format=format:%ct HEAD)
DATE_FMT               := "%Y-%m-%dT%H:%M:%SZ"
BUILD_DATE             := $(shell date -u -d "@$SOURCE_DATE_EPOCH" "+${DATE_FMT}" 2>/dev/null || date -u -r "${SOURCE_DATE_EPOCH}" "+${DATE_FMT}" 2>/dev/null || date -u "+${DATE_FMT}")
BUILD_USER             := $(shell whoami)

LDFLAGS          = -X main.Version=$(VERSION)$(MILESTONE) -X main.Commit=$(GITCOMMIT) -X main.Date=$(BUILD_DATE) -X main.Author=$(BUILD_USER)
GOS              = $(shell find . -type f -name "*.go" ! -name "*_test.go")

# ----------------------------------------------------------------------------------------------------------------------
# Release build options
# ----------------------------------------------------------------------------------------------------------------------
RELEASE_DRY_RUN  ?= true

# ======================================================================================================================
# Makefile targets start here
# ======================================================================================================================

# ----------------------------------------------------------------------------------------------------------------------
# Display the Makefile help - this is a list of the targets with a description.
# This target MUST be the first target in the Makefile so that it is run when running make with no arguments
# ----------------------------------------------------------------------------------------------------------------------
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


# ======================================================================================================================
# Build targets
# ======================================================================================================================
##@ Build

.PHONY: all
all: clean cohctl-all  ## Build all the Coherence CLI artefacts

# ----------------------------------------------------------------------------------------------------------------------
# Build the Java artifacts
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: build-mvn
build-mvn: ## Build the Java artefacts
ifeq ($(PROFILES),,commercial)
	mvn -B -f java $(MAVEN_BUILD_OPTS) clean install -DskipTests -P commercial
else ifeq ($(PROFILES),,federation)
	mvn -B -f java $(MAVEN_BUILD_OPTS) clean install -DskipTests -P federation
else
	mvn -B -f java  $(MAVEN_BUILD_OPTS) clean install -DskipTests
endif

# ----------------------------------------------------------------------------------------------------------------------
# Clean-up all of the build artifacts
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: clean
clean: ## Cleans the build
	@echo "Cleaning Project"
	-rm -rf build/_output
	-rm -rf bin
	-rm -rf $(BUILD_SHARED)
ifeq ($(PROFILES),,commercial)
	mvn -B -f java clean -DskipTests $(MAVEN_BUILD_OPTS) -P commercial
else
	mvn -B -f java clean install -DskipTests $(MAVEN_BUILD_OPTS)
endif
	@mkdir -p $(TEST_LOGS_DIR)
	@mkdir -p $(COVERAGE_DIR)

# ----------------------------------------------------------------------------------------------------------------------
# Configure the build properties
# ----------------------------------------------------------------------------------------------------------------------
$(BUILD_PROPS):
	@echo "Creating build directories"
	@mkdir -p $(BUILD_OUTPUT)
	@mkdir -p $(BUILD_BIN)
	@mkdir -p $(BUILD_TARGETS)
	@mkdir -p $(TEST_LOGS_DIR)
	@mkdir -p $(TOOLS_BIN)
	@mkdir -p $(COVERAGE_DIR)
	@mkdir -p $(BUILD_SHARED)

# ----------------------------------------------------------------------------------------------------------------------
# Build the Coherence CLI Test Image
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: build-federation-images
build-federation-images: ## Build the Test images for federation
	@echo "${MAVEN_BUILD_OPTS}"
	@ ./scripts/check_image.sh $(COHERENCE_BASE_IMAGE)
	mvn -B -f java/coherence-cli-test clean package jib:dockerBuild -DskipTests -P federation1$(PROFILES) -Djib.to.image=$(TEST_APPLICATION_IMAGE_1) -Dcoherence.test.base.image=$(COHERENCE_BASE_IMAGE) $(MAVEN_BUILD_OPTS)
	mvn -B -f java/coherence-cli-test clean package jib:dockerBuild -DskipTests -P federation2$(PROFILES) -Djib.to.image=$(TEST_APPLICATION_IMAGE_2) -Dcoherence.test.base.image=$(COHERENCE_BASE_IMAGE) $(MAVEN_BUILD_OPTS)
	echo "COHERENCE_IMAGE1=$(TEST_APPLICATION_IMAGE_1)" > $(ENV_FILE)
	echo "COHERENCE_IMAGE2=$(TEST_APPLICATION_IMAGE_2)" >> $(ENV_FILE)
	echo "CURRENT_UID=$(USER_ID)" >> $(ENV_FILE)

# ----------------------------------------------------------------------------------------------------------------------
# Build the Coherence CLI Test Image
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: build-test-images
build-test-images: ## Build the Test images
	@echo "${MAVEN_BUILD_OPTS}"
	@ ./scripts/check_image.sh $(COHERENCE_BASE_IMAGE)
	mvn -B -f java/coherence-cli-test clean package jib:dockerBuild -DskipTests -P member1$(PROFILES) -Djib.to.image=$(TEST_APPLICATION_IMAGE_1) -Dcoherence.test.base.image=$(COHERENCE_BASE_IMAGE) $(MAVEN_BUILD_OPTS)
	mvn -B -f java/coherence-cli-test clean package jib:dockerBuild -DskipTests -P member2$(PROFILES) -Djib.to.image=$(TEST_APPLICATION_IMAGE_2) -Dcoherence.test.base.image=$(COHERENCE_BASE_IMAGE) $(MAVEN_BUILD_OPTS)
	echo "COHERENCE_IMAGE1=$(TEST_APPLICATION_IMAGE_1)" > $(ENV_FILE)
	echo "COHERENCE_IMAGE2=$(TEST_APPLICATION_IMAGE_2)" >> $(ENV_FILE)
	echo "CURRENT_UID=$(USER_ID)" >> $(ENV_FILE)

# ----------------------------------------------------------------------------------------------------------------------
# Internal make step that builds the Coherence CLI for local platform
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: cohctl
cohctl: $(BUILD_BIN)/cohctl-local   ## Build the Coherence CLI binary for the local platform

# ----------------------------------------------------------------------------------------------------------------------
# Internal make step that builds the Coherence CLI
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: cohctl-all
cohctl-all: $(BUILD_PROPS) $(GOS)  ## Build the Coherence CLI binary for all supported platforms
	@echo "Building Coherence CLI for all supported platforms"
	@echo "Linux amd64 (x64)"
	mkdir -p $(BUILD_BIN)/linux/amd64 || true
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_BIN)/linux/amd64/cohctl ./cohctl

	@echo "Linux arm64"
	mkdir -p $(BUILD_BIN)/linux/arm64 || true
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 GO111MODULE=on go build -trimpath -ldflags "$(LDFLAGS)" -a -o $(BUILD_BIN)/linux/arm64/cohctl ./cohctl

	@echo "Linux i386"
	mkdir -p $(BUILD_BIN)/linux/386 || true
	CGO_ENABLED=0 GOOS=linux GOARCH=386 GO111MODULE=on go build -trimpath -ldflags "$(LDFLAGS)" -a -o $(BUILD_BIN)/linux/386/cohctl ./cohctl

	@echo "Windows amd64 (x64)"
	mkdir -p $(BUILD_BIN)/windows/amd64 || true
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 GO111MODULE=on go build -trimpath -ldflags "$(LDFLAGS)" -a -o $(BUILD_BIN)/windows/amd64/cohctl.exe ./cohctl

	@echo "Windows arm"
	mkdir -p $(BUILD_BIN)/windows/arm || true
	CGO_ENABLED=0 GOOS=windows GOARCH=arm GO111MODULE=on go build -trimpath -ldflags "$(LDFLAGS)" -a -o $(BUILD_BIN)/windows/arm/cohctl.exe ./cohctl

# ----------------------------------------------------------------------------------------------------------------------
# Internal make step that builds the Coherence CLI for Mac AMD
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: cohctl-mac-amd
cohctl-mac-amd:  $(BUILD_PROPS) $(GOS)  ## Build the Coherence CLI binary for Mac AMD
	@echo "Apple amd64 (x64)"
	mkdir -p $(BUILD_BIN)/darwin/amd64 || true
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 GO111MODULE=on go build -trimpath -ldflags "$(LDFLAGS)" -a -o $(BUILD_BIN)/darwin/amd64/cohctl ./cohctl

# ----------------------------------------------------------------------------------------------------------------------
# Internal make step that builds the Coherence CLI for lMac ARM
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: cohctl-mac-arm
cohctl-mac-arm:  $(BUILD_PROPS) $(GOS)  ## Build the Coherence CLI binary for Mac ARM
	@echo "Apple Silicon (M1)"
	mkdir -p $(BUILD_BIN)/darwin/arm64 || true
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 GO111MODULE=on go build -trimpath -ldflags "$(LDFLAGS)" -a -o $(BUILD_BIN)/darwin/arm64/cohctl ./cohctl

# ----------------------------------------------------------------------------------------------------------------------
# Build a MacOS Package
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: mac-pkg
mac-pkg:  ## Make a MacOS Package
	@echo "Making MacOS Package"
	@mkdir -p $(BUILD_BIN)/pkg
	@cp $(BUILD_BIN)/cohctl $(BUILD_BIN)/pkg
	@chmod 755 $(BUILD_BIN)/cohctl
	sudo pkgbuild --ownership preserve --install-location /usr/local/bin --version $(VERSION)$(MILESTONE) --root $(BUILD_BIN)/pkg --identifier com.oracle.coherence.cohctl $(PKG_DIR)/cohctl-$(VERSION).pkg

$(BUILD_BIN)/cohctl-local: $(BUILD_PROPS) $(GOS)
	@echo "Building Coherence CLI for local platform"
	CGO_ENABLED=0 GO111MODULE=on go build -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_BIN)/cohctl ./cohctl

# ----------------------------------------------------------------------------------------------------------------------
# Generate output to be included into the commands reference documentation
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: generate-docs
generate-docs: cohctl  ## Generate Doc Snippets
	@echo "Generating Doc Snippets"
	./scripts/generate-doc-snippets.sh $(BUILD_DOCS) $(BUILD_BIN)

# ----------------------------------------------------------------------------------------------------------------------
# Build the documentation.
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: docs
docs: generate-docs ## Build the Documentation
	mvn -B -f java install -P docs -pl docs -DskipTests -Dcli.version=$(VERSION)$(MILESTONE) -Dcoherence.version=$(COHERENCE_VERSION) $(MAVEN_OPTIONS)
	mkdir -p $(BUILD_OUTPUT)/docs/images/images
	cp -R docs/images/* build/_output/docs/images/

# ----------------------------------------------------------------------------------------------------------------------
# Start a local web server to serve the documentation.
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: serve-docs
serve-docs:   ## Serve the Documentation
	@echo "Serving documentation on http://localhost:8888"
	cd $(BUILD_OUTPUT)/docs; \
	python -m SimpleHTTPServer 8888

# ======================================================================================================================
# General development related targets
# ======================================================================================================================
##@ Development

# ----------------------------------------------------------------------------------------------------------------------
# Performs a copyright check.
# To add exclusions add the file or folder pattern using the -X parameter.
# Add directories to be scanned at the end of the parameter list.
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: copyright
copyright: getcopyright ## Check copyright headers
	@java -cp scripts/$(COPYRIGHT_JAR) \
	  org.glassfish.copyright.Copyright -C scripts/copyright.txt \
	  -X bin/ \
	  -X ./test/test_utils/shared/ \
	  -X ./test/test_utils/test_utils.go \
	  -X dependency-reduced-pom.xml \
	  -X binaries/ \
	  -X build/ \
	  -X /Dockerfile \
	  -X .Dockerfile \
	  -X docs/ \
	  -X go.mod \
	  -X go.sum \
	  -X HEADER.txt \
	  -X .iml \
	  -X .jar \
	  -X jib-cache/ \
	  -X .jks \
	  -X .json \
	  -X LICENSE.txt \
	  -X Makefile \
	  -X cohctl-terminal.gif \
	  -X .md \
	  -X .mvn/ \
	  -X mvnw \
	  -X mvnw.cmd \
	  -X .png \
	  -X .sh \
	  -X temp/ \
	  -X /test-report.xml \
	  -X THIRD_PARTY_LICENSES.txt \
	  -X .tpl \
	  -X .txt \
	  -X .yaml \
	  -X pkg/data/assets/

# ----------------------------------------------------------------------------------------------------------------------
# Executes golangci-lint to perform various code review checks on the source.
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: golangci
golangci: $(TOOLS_BIN)/golangci-lint ## Go code review
	$(TOOLS_BIN)/golangci-lint run -v --timeout=5m ./pkg/...

# ======================================================================================================================
# Test targets
# ======================================================================================================================
##@ Test

# ----------------------------------------------------------------------------------------------------------------------
# Startup cluster members via docker compose
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: test-cluster-startup
test-cluster-startup: $(BUILD_PROPS) ## Startup any test cluster members using docker-compose
	cd test/test_utils && docker-compose -f docker-compose-2-members.yaml --env-file .env up -d

# ----------------------------------------------------------------------------------------------------------------------
# Shutdown any cluster members via docker compose
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: test-cluster-shutdown
test-cluster-shutdown: ## Shutdown any test cluster members using docker-compose
	cd test/test_utils && docker-compose -f docker-compose-2-members.yaml down || true

# ----------------------------------------------------------------------------------------------------------------------
# Startup standalone coherence via java -jar
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: test-coherence-startup
test-coherence-startup: $(BUILD_PROPS) ## Startup standalone cluster
	scripts/startup-clusters.sh $(TEST_LOGS_DIR) $(CLUSTER_PORT) $(COHERENCE_GROUP_ID) ${COHERENCE_VERSION}
	@echo "Clusters started up"

# ----------------------------------------------------------------------------------------------------------------------
# Shutdown coherence via java -jar
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: test-coherence-shutdown
test-coherence-shutdown: ## shutdown standalone cluster
	@ps -ef | grep shutMeDownPlease | grep -v grep | awk '{print $$2}' | xargs kill -9 || true
	@echo "Clusters shutdown"

# ----------------------------------------------------------------------------------------------------------------------
# Executes the Go unit tests
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: test-cohctl
test-cohctl: test-clean gotestsum $(BUILD_PROPS) ## Run the CLI unit tests
	@echo "Running Coherence CLI tests"
	CGO_ENABLED=0 $(GOTESTSUM) --format testname --junitfile $(TEST_LOGS_DIR)/cohctl-test.xml \
	  -- $(GO_TEST_FLAGS) -v -coverprofile=$(COVERAGE_DIR)/cover-unit.out ./pkg/cmd/... ./pkg/utils/...
	go tool cover -html=$(COVERAGE_DIR)/cover-unit.out -o $(COVERAGE_DIR)/cover-unit.html

# ----------------------------------------------------------------------------------------------------------------------
# Executes the Go end to end tests for standalone Coherence
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: test-e2e-standalone
test-e2e-standalone: test-clean gotestsum $(BUILD_PROPS) ## Run e2e tests with Coherence
	CGO_ENABLED=0 $(GOTESTSUM) --format testname --junitfile $(TEST_LOGS_DIR)/cohctl-test-e2e-standalone.xml \
	  -- $(GO_TEST_FLAGS) -v ./test/e2e/standalone/...

# ----------------------------------------------------------------------------------------------------------------------
# Executes the Go end to end tests for federation and Coherence
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: test-e2e-federation
test-e2e-federation: test-clean gotestsum $(BUILD_PROPS) ## Run e2e federation tests with Coherence
	CGO_ENABLED=0 $(GOTESTSUM) --format testname --junitfile $(TEST_LOGS_DIR)/cohctl-test-e2e-federation.xml \
	  -- $(GO_TEST_FLAGS) -v ./test/e2e/federation/...

# ----------------------------------------------------------------------------------------------------------------------
# Executes the Go discovery tests for standalone Coherence
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: test-discovery
test-discovery: test-clean gotestsum $(BUILD_PROPS) ## Run Discovery tests with Coherence
	make test-coherence-shutdown || true
	make test-coherence-startup
	CGO_ENABLED=0 $(GOTESTSUM) --format testname --junitfile $(TEST_LOGS_DIR)/cohctl-test-e2e.xml \
	  -- $(GO_TEST_FLAGS) -v  ./test/e2e/discovery/...
	make test-coherence-shutdown

# ----------------------------------------------------------------------------------------------------------------------
# Executes the Go end to end tests and unit tests for standalone Coherence with coverage
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: test-coverage
test-coverage: test-clean gotestsum $(BUILD_PROPS) ## Run e2e tests with Coherence
	make test-coherence-shutdown || true
	make test-coherence-startup
	CGO_ENABLED=0 $(GOTESTSUM) --format testname --junitfile $(TEST_LOGS_DIR)/cohctl-test-e2e.xml \
	  -- $(GO_TEST_FLAGS) -v -coverprofile=$(COVERAGE_DIR)/cover-full.out -coverpkg=./pkg/cmd,./pkg/discovery,./pkg/fetcher,./pkg/utils ./pkg/cmd/... ./pkg/discovery/... ./test/e2e/standalone/...
	go tool cover -html=$(COVERAGE_DIR)/cover-full.out -o $(COVERAGE_DIR)/cover-full.html
	go tool cover -func $(COVERAGE_DIR)/cover-full.out
	make test-coherence-shutdown


# ----------------------------------------------------------------------------------------------------------------------
# Release the Coherence Operator to the gh-pages branch.
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: release-ghpages
release-ghpages:
	./scripts/release-ghpages.sh $(VERSION)$(MILESTONE) $(BUILD_OUTPUT)

# ----------------------------------------------------------------------------------------------------------------------
# Cleans the test cache
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: test-clean
test-clean: gotestsum ## Clean the go test cache
	@echo "Cleaning test cache"
	go clean -testcache

# ----------------------------------------------------------------------------------------------------------------------
# Obtain the golangci-lint binary
# ----------------------------------------------------------------------------------------------------------------------
$(TOOLS_BIN)/golangci-lint:
	@mkdir -p $(TOOLS_BIN)
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(TOOLS_BIN) v1.43.0

# ----------------------------------------------------------------------------------------------------------------------
# Find or download gotestsum
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: gotestsum
GOTESTSUM = $(TOOLS_BIN)/gotestsum
gotestsum: ## Download gotestsum locally if necessary.
	$(call go-get-tool,$(GOTESTSUM),gotest.tools/gotestsum@v0.5.2)

# ----------------------------------------------------------------------------------------------------------------------
# Find or download copyright
# ----------------------------------------------------------------------------------------------------------------------
.PHONY: getcopyright
getcopyright: ## Download copyright jar locally if necessary.
	@test -f scripts/$(COPYRIGHT_JAR)  || curl -o scripts/$(COPYRIGHT_JAR) \
		https://repo.maven.apache.org/maven2/org/glassfish/copyright/glassfish-copyright-maven-plugin/2.4/glassfish-copyright-maven-plugin-2.4.jar

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2) into $(TOOLS_BIN)" ;\
GOBIN=$(TOOLS_BIN) go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
