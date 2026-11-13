<!--
Copyright (c) 2021, 2026 Oracle and/or its affiliates.
Licensed under the Universal Permissive License v 1.0 as shown at
https://oss.oracle.com/licenses/upl.
-->

-----
![logo](docs/images/logo-with-name.png)

![Coherence CI](https://github.com/oracle/coherence-cli/actions/workflows/build.yaml/badge.svg?branch=main)
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
* [Upgrading from 1.10.14](#upgrading-from-11014)
* [Getting Started](#getting-started)
* [Supported Coherence Versions](#supported-coherence-versions)
* [Get Involved](#need-more-help-have-a-suggestion-come-and-say-hello)

## Overview 

The Coherence command line interface, `cohctl`, is a lightweight tool, in the tradition of tools such as kubectl,
which can be scripted or used interactively to manage and monitor Coherence clusters. You can use `cohctl` to view cluster information
such as services, caches, members, etc, as well as perform various management operations against clusters.

The CLI accesses clusters using the HTTP Management over REST interface and therefore requires this to be enabled on any clusters
you want to monitor or manage. See the [Coherence Documentation](https://docs.oracle.com/en/middleware/fusion-middleware/coherence/14.1.2/rest-reference/index.html)
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

### Linux

On Linux, the installer downloads the release checksum manifest and verifies the selected binary before making it executable or copying it to `/usr/local/bin`:

<details>
<summary>Show automatic Linux installation commands</summary>

```bash
curl -fsSL --proto '=https' --proto-redir '=https' https://raw.githubusercontent.com/oracle/coherence-cli/main/scripts/install.sh | bash
```

When you install the CLI, normally administrative privileges are required as the `cohctl` executable is moved to the `/usr/local/bin` directory.

If you do not want to copy to `/usr/local/bin` automatically, you can set the following environment variable `export COPY=false`, before running the above to download only and then
manually copy to your desired location.

</details>

Alternatively, download and verify the Linux artifact manually without running the installer:

<details>
<summary>Show manual Linux download and checksum verification commands</summary>

```bash
(
set -euo pipefail
VERSION=$(curl -fsSL --proto '=https' --proto-redir '=https' https://oracle.github.io/coherence-cli/stable.txt)
while [[ "${VERSION}" == *$'\r' ]]; do VERSION=${VERSION%$'\r'}; done
[[ "${VERSION}" =~ ^[0-9A-Za-z][0-9A-Za-z._-]*$ ]] || { echo "Invalid version" >&2; exit 1; }
ARCH=amd64 # use arm64 on Arm systems
ASSET="cohctl-${VERSION}-linux-${ARCH}"
MANIFEST="cohctl-${VERSION}-checksums.txt"
RELEASE_URL="https://github.com/oracle/coherence-cli/releases/download/${VERSION}"
DESTINATION="${PWD}/${ASSET}"
TEMP=$(mktemp -d)
trap 'rm -rf -- "${TEMP}"' EXIT
cd "${TEMP}"

curl -fsSL --proto '=https' --proto-redir '=https' -o "${ASSET}" "${RELEASE_URL}/${ASSET}"
curl -fsSL --proto '=https' --proto-redir '=https' -o "${MANIFEST}" "${RELEASE_URL}/${MANIFEST}"
awk -v target="${ASSET}" '
    $2 == target {
        matches++
        if (NF == 2 && length($1) == 64 && $1 !~ /[^0-9a-f]/) {
            valid++
            entry = $1 "  " $2
        }
    }
    END {
        if (matches != 1 || valid != 1) exit 1
        print entry
    }
' "${MANIFEST}" > "${ASSET}.sha256"
sha256sum -c "${ASSET}.sha256"
chmod u+x "${ASSET}"
mv -- "${ASSET}" "${DESTINATION}"
)
```

</details>

### Windows

Download a Windows executable using the following command in Command Prompt or PowerShell, then copy `cohctl.exe` to a directory on your `PATH`:

```cmd
curl.exe --proto "=https" --proto-redir "=https" -fLo cohctl.exe "https://github.com/oracle/coherence-cli/releases/download/1.10.14/cohctl-1.10.14-windows-amd64.exe"
```

The example downloads `1.10.14` for x86-64. Replace `1.10.14` in both places with your desired [released version](https://github.com/oracle/coherence-cli/releases), or change `amd64` to `arm64` for Arm.

The command above only downloads the executable. To download and verify the latest stable Windows executable using SHA-256 checksums, use the PowerShell procedure below. Use `amd64` for x86-64 or change it to `arm64` for Arm. Before running, replace `C:\path\on\your\PATH\cohctl.exe` with your chosen destination in a directory on your `PATH`.

<details>
<summary>Show PowerShell installation commands</summary>

```powershell
& {
    $ErrorActionPreference = 'Stop'
    Set-StrictMode -Version Latest
    $Version = (Invoke-WebRequest "https://oracle.github.io/coherence-cli/stable.txt" -UseBasicParsing -ErrorAction Stop).Content.TrimEnd("`r", "`n")
    if ($Version -cnotmatch '\A[0-9A-Za-z][0-9A-Za-z._-]*\z') { throw 'Invalid version' }
    $Asset = "cohctl-$Version-windows-amd64.exe"
    $Manifest = "cohctl-$Version-checksums.txt"
    $ReleaseUrl = "https://github.com/oracle/coherence-cli/releases/download/$Version"
    $Temp = Join-Path ([IO.Path]::GetTempPath()) ([Guid]::NewGuid().ToString())
    $null = New-Item -ItemType Directory -Path $Temp
    try {
        $BinaryPath = Join-Path $Temp $Asset
        $ManifestPath = Join-Path $Temp $Manifest
        Invoke-WebRequest "$ReleaseUrl/$Asset" -OutFile $BinaryPath -ErrorAction Stop
        Invoke-WebRequest "$ReleaseUrl/$Manifest" -OutFile $ManifestPath -ErrorAction Stop
        $Entries = @(Get-Content -LiteralPath $ManifestPath -ErrorAction Stop | Where-Object {
            $Fields = $_ -split '\s+'
            $Fields.Count -ge 2 -and $Fields[1] -ceq $Asset
        })
        if ($Entries.Count -ne 1) { throw "Expected exactly one checksum entry for $Asset" }
        $Pattern = '\A([0-9a-f]{64})  ' + [regex]::Escape($Asset) + '\z'
        if ($Entries[0] -cnotmatch $Pattern) { throw "Malformed checksum entry for $Asset" }
        $Expected = $Matches[1]
        $Actual = (Get-FileHash -LiteralPath $BinaryPath -Algorithm SHA256 -ErrorAction Stop).Hash.ToLowerInvariant()
        if ($Actual -cne $Expected) { throw "Checksum verification failed for $Asset" }
        Copy-Item -LiteralPath $BinaryPath -Destination C:\path\on\your\PATH\cohctl.exe -ErrorAction Stop
    } finally {
        Remove-Item -LiteralPath $Temp -Recurse -Force -ErrorAction Stop
    }
}
```

</details>

For additional information, see the [Install guide](https://docs.coherence.community/coherence-cli/docs/latest/docs/installation/installation#install-windows).

Release checksums detect corruption or mismatched downloads. Because the manifest is distributed with the release, it does not provide independent provenance if both the artifact and manifest are compromised.

See the [Install guide](https://docs.coherence.community/coherence-cli/docs/latest/docs/installation/installation) for
more information on downloading and installing the CLI.

## Upgrading from 1.10.14

Preparing for **1.11.0**? Read the [upgrade checklist](docs/about/upgrading_1_11_0.adoc)
before restarting local development clusters, reinitializing monitoring, or updating TLS-related scripts.
It explains automatic migrations and the steps you need to take. See also the
[1.11.0 release notes](release-notes/1.11.0.md). These describe the upcoming release;
published versions are listed on [GitHub Releases](https://github.com/oracle/coherence-cli/releases).

## Getting Started

Documentation for the Coherence CLI is available [here](https://docs.coherence.community/coherence-cli/docs/latest)

The fastest way to experience the Coherence CLI is to follow the
[Quick Start guide](https://docs.coherence.community/coherence-cli/docs/latest/docs/about/quickstart).

## Supported Coherence Versions

The CLI supports and is certified against the following Community and Commercial editions of Coherence:

**Coherence Community Edition**
* 24.09+
* 22.06+
* 14.1.2-0-+
* 15.1.1-0-*

> Note: As CE 14.1.2-0-0 is out, CE 14.1.1-0+ is no longer supported or tested. Please use CE 14.1.2-0+

**Coherence Grid/ Enterprise Edition**
* 12.2.1.4.x - minimum patch level of 12.2.1.4.10+ required
* 14.1.1.0.x - minimum patch level of 14.1.1.0.5+ required
* 14.1.1.2206.x Feature Pack
* 14.1.2.0.x
* 15.1.1.0.x

> Note: If you are on a patch set below the minimum recommended above, then CLI may work, but some functionality may not be available. It
> is always recommended to upgrade to the latest Coherence patch as soon as you are able to.

## Need more help? Have a suggestion? Come and say "Hello!"

We have a **public Slack channel** where you can get in touch with us to ask questions about using the Coherence CLI
or give us feedback or suggestions about what features and improvements you would like to see. We would love
to hear from you. To join our channel,
please [visit this site to get an invitation](https://join.slack.com/t/oraclecoherence/shared_invite/enQtNzcxNTQwMTAzNjE4LTJkZWI5ZDkzNGEzOTllZDgwZDU3NGM2YjY5YWYwMzM3ODdkNTU2NmNmNDFhOWIxMDZlNjg2MzE3NmMxZWMxMWE).  
The invitation email will include details of how to access our Slack
workspace.  After you are logged in, please come to `#coherence-cli` and say, "hello!"

## Contributing

This project welcomes contributions from the community. Before submitting a pull request, please [review our contribution guide](./CONTRIBUTING.md)

## Security

Please consult the [security guide](./SECURITY.md) for our responsible security vulnerability disclosure process

## License

Copyright (c) 2021, 2026 Oracle and/or its affiliates.

Released under the Universal Permissive License v1.0 as shown at
<https://oss.oracle.com/licenses/upl/>.
