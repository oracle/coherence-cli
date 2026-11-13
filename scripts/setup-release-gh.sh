#!/bin/bash
#
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#
# Install the reviewed release CLI on the Linux x86_64 GitHub Actions runner.
# Pin source: https://github.com/cli/cli/releases/tag/v2.103.3
# Digest matches the release API asset metadata and gh_2.103.3_checksums.txt:
# https://github.com/cli/cli/releases/download/v2.103.3/gh_2.103.3_checksums.txt
# To upgrade, review the official release/security notes and license, record the
# new version and digest below, and compare the archive hash before execution.
# Run make test-release-gh and verify the actual binary on Linux x86_64.
set -euo pipefail

version='2.103.3'
sha256='4c806978ffce1cec70853302a56d70ac6da4f19bae7adca4e1a8ab00633dd91d'
archive="gh_${version}_linux_amd64.tar.gz"
url="https://github.com/cli/cli/releases/download/v${version}/${archive}"

if [ "$(uname -s)" != Linux ] || [ "$(uname -m)" != x86_64 ]; then
    echo 'Release GitHub CLI setup requires Linux x86_64.' >&2
    exit 1
fi
: "${RUNNER_TEMP:?RUNNER_TEMP must be set}"
: "${GITHUB_PATH:?GITHUB_PATH must be set}"
: "${GITHUB_OUTPUT:?GITHUB_OUTPUT must be set}"

# Neither public downloads nor version reporting need credentials.
unset GH_TOKEN GITHUB_TOKEN GH_ENTERPRISE_TOKEN GITHUB_ENTERPRISE_TOKEN
export GH_NO_UPDATE_NOTIFIER=1 GH_NO_EXTENSION_UPDATE_NOTIFIER=1 GH_PROMPT_DISABLED=1

install_dir=$(mktemp -d "${RUNNER_TEMP}/release-gh.XXXXXX")
trap 'rm -rf -- "${install_dir}"' EXIT
curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' \
    --tlsv1.2 --retry 3 --connect-timeout 10 --max-time 180 \
    --output "${install_dir}/${archive}" "${url}"

# Use the reviewed repository pin, never a checksum fetched at workflow runtime.
# Fail before extracting or executing any downloaded content on a mismatch.
(cd "${install_dir}" && printf '%s  %s\n' "${sha256}" "${archive}" | sha256sum --check --strict -)
tar -xzf "${install_dir}/${archive}" -C "${install_dir}" --strip-components=1 \
    "gh_${version}_linux_amd64/bin/gh" "gh_${version}_linux_amd64/LICENSE"
rm -- "${install_dir}/${archive}"

reported_version=$("${install_dir}/bin/gh" --version)
printf '%s\n' "${reported_version}"
actual_version=$(printf '%s\n' "${reported_version}" | awk 'NR == 1 {print $3}')
if [ "${actual_version}" != "${version}" ]; then
    echo "Expected GitHub CLI ${version}, got ${actual_version}." >&2
    exit 1
fi

# Publish the installation only after both integrity and version checks pass.
printf '%s\n' "${install_dir}/bin" >> "${GITHUB_PATH}"
printf 'version=%s\npath=%s/bin/gh\n' "${version}" "${install_dir}" >> "${GITHUB_OUTPUT}"
trap - EXIT
