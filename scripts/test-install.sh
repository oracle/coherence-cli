#!/bin/bash
#
# Copyright (c) 2026 Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#
# Description: Hermetic tests for scripts/install.sh.
#
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
INSTALLER="${SCRIPT_DIR}/install.sh"
TEST_ROOT=$(mktemp -d)
MOCK_BIN="${TEST_ROOT}/bin"
LOG_FILE="${TEST_ROOT}/commands.log"
STDOUT_FILE="${TEST_ROOT}/stdout"
STDERR_FILE="${TEST_ROOT}/stderr"
ARTIFACT_FILE="${TEST_ROOT}/artifact"
MANIFEST_FILE="${TEST_ROOT}/manifest"
REAL_SHA256SUM=$(command -v sha256sum)
REAL_CHMOD=$(command -v chmod)
REAL_MV=$(command -v mv)
REAL_MKTEMP=$(command -v mktemp)
TEMP_ROOT="${TEST_ROOT}/temporary"
INSTALL_DEST="${TEST_ROOT}/installed-cohctl"
MISSING_SHA_BIN="${TEST_ROOT}/without-sha256sum"
RETAINED_DIRS=()
TEST_COUNT=0

function cleanup() {
    local directory
    for directory in "${RETAINED_DIRS[@]}"; do
        rm -rf -- "${directory}"
    done
    rm -rf -- "${TEST_ROOT}"
}
trap cleanup EXIT

mkdir -p "${MOCK_BIN}" "${TEMP_ROOT}" "${MISSING_SHA_BIN}"

function create_mocks() {
    cat > "${MOCK_BIN}/uname" <<'EOF'
#!/bin/bash
if [ "${1:-}" = "-m" ]; then
    printf '%s\n' "${MOCK_ARCH}"
else
    printf '%s\n' "${MOCK_OS}"
fi
EOF

    cat > "${MOCK_BIN}/curl" <<'EOF'
#!/bin/bash
output=""
url=""
while [ "$#" -gt 0 ]; do
    case "$1" in
        -o)
            output=$2
            shift 2
            ;;
        -sLo)
            output=$2
            shift 2
            ;;
        -*)
            shift
            ;;
        *)
            url=$1
            shift
            ;;
    esac
done

printf 'curl %s\n' "${url}" >> "${MOCK_LOG}"
case "${url}" in
    *stable.txt)
        if [ "${MOCK_STABLE_FAIL}" = "true" ]; then
            exit 22
        fi
        printf '%s' "${MOCK_STABLE}"
        ;;
    *checksums.txt)
        if [ "${MOCK_MANIFEST_FAIL}" = "true" ]; then
            exit 22
        fi
        cp "${MOCK_MANIFEST}" "${output}"
        ;;
    *Oracle-Coherence-CLI-*.pkg)
        cp "${MOCK_ARTIFACT}" "${output}"
        ;;
    *)
        if [ "${MOCK_ARTIFACT_FAIL}" = "true" ]; then
            exit 22
        fi
        cp "${MOCK_ARTIFACT}" "${output}"
        ;;
esac
EOF

    cat > "${MOCK_BIN}/sha256sum" <<'EOF'
#!/bin/bash
printf 'sha256sum %s\n' "$*" >> "${MOCK_LOG}"
exec "${REAL_SHA256SUM}" "$@"
EOF

    cat > "${MOCK_BIN}/chmod" <<'EOF'
#!/bin/bash
printf 'chmod %s\n' "$*" >> "${MOCK_LOG}"
exec "${REAL_CHMOD}" "$@"
EOF

    cat > "${MOCK_BIN}/mv" <<'EOF'
#!/bin/bash
printf 'mv %s\n' "$*" >> "${MOCK_LOG}"
exec "${REAL_MV}" "$@"
EOF

    cat > "${MOCK_BIN}/sudo" <<'EOF'
#!/bin/bash
printf 'sudo %s\n' "$*" >> "${MOCK_LOG}"
[ "${MOCK_INSTALL_FAIL}" != "true" ] || exit 1
[ "$#" -eq 3 ] && [ "$1" = mv ] && [ "$3" = /usr/local/bin/ ] || exit 1
exec "${REAL_MV}" "$2" "${MOCK_INSTALL_DEST}"
EOF

    cat > "${MOCK_BIN}/open" <<'EOF'
#!/bin/bash
printf 'open %s\n' "$*" >> "${MOCK_LOG}"
exit 0
EOF

    cat > "${MOCK_BIN}/mktemp" <<'EOF'
#!/bin/bash
exec "${REAL_MKTEMP}" -d "${MOCK_TEMP_ROOT}/install.XXXXXX"
EOF

    "${REAL_CHMOD}" u+x "${MOCK_BIN}/mktemp" "${MOCK_BIN}/uname" "${MOCK_BIN}/curl" \
        "${MOCK_BIN}/sha256sum" "${MOCK_BIN}/chmod" "${MOCK_BIN}/mv" \
        "${MOCK_BIN}/sudo" "${MOCK_BIN}/open"
}

create_mocks
ln -s "${MOCK_BIN}/uname" "${MISSING_SHA_BIN}/uname"
ln -s "$(command -v bash)" "${MISSING_SHA_BIN}/bash"

function reset_case() {
    rm -rf -- "${TEMP_ROOT:?}/"*
    rm -f -- "${INSTALL_DEST}"
    TEST_INSTALL_FAIL=false
    TEST_MISSING_SHA=false
    TEST_OS=Linux
    TEST_ARCH=x86_64
    TEST_COPY=false
    TEST_VERSION=1.2.3
    TEST_STABLE="${TEST_VERSION}"$'\n'
    TEST_STABLE_FAIL=false
    TEST_MANIFEST_MODE=valid
    TEST_MANIFEST_FAIL=false
    TEST_ARTIFACT_FAIL=false
    CASE_STATUS=0
    : > "${LOG_FILE}"
    : > "${STDOUT_FILE}"
    : > "${STDERR_FILE}"
    printf 'verified cohctl fixture\n' > "${ARTIFACT_FILE}"
}

function normalized_arch() {
    case "${TEST_ARCH}" in
        x86_64)
            printf 'amd64\n'
            ;;
        aarch64|arm64)
            printf 'arm64\n'
            ;;
        *)
            printf '%s\n' "${TEST_ARCH}"
            ;;
    esac
}

function prepare_manifest() {
    local asset digest
    asset="cohctl-${TEST_VERSION}-linux-$(normalized_arch)"
    digest=$("${REAL_SHA256SUM}" "${ARTIFACT_FILE}" | awk '{print $1}')

    case "${TEST_MANIFEST_MODE}" in
        valid)
            printf '%s  unrelated-file\n%s  %s\n' "${digest}" "${digest}" "${asset}" > "${MANIFEST_FILE}"
            ;;
        missing)
            printf '%s  unrelated-file\n' "${digest}" > "${MANIFEST_FILE}"
            ;;
        duplicate)
            printf '%s  %s\n%s  %s\n' "${digest}" "${asset}" "${digest}" "${asset}" > "${MANIFEST_FILE}"
            ;;
        malformed)
            printf '%s  %s\n' "$(printf '%s' "${digest}" | tr '[:lower:]' '[:upper:]')" "${asset}" > "${MANIFEST_FILE}"
            ;;
        mismatch)
            printf '%064d  %s\n' 0 "${asset}" > "${MANIFEST_FILE}"
            ;;
        *)
            echo "Unknown manifest mode: ${TEST_MANIFEST_MODE}" >&2
            exit 1
            ;;
    esac
}

function run_case() {
    prepare_manifest

    local case_path="${MOCK_BIN}:${PATH}"
    local copy_env=("COPY=${TEST_COPY}")
    if [ "${TEST_COPY}" = unset ]; then copy_env=(-u COPY); fi
    if [ "${TEST_MISSING_SHA}" = true ]; then case_path="${MISSING_SHA_BIN}"; fi
    set +e
    env "${copy_env[@]}" \
        PATH="${case_path}" \
        MOCK_TEMP_ROOT="${TEMP_ROOT}" \
        MOCK_INSTALL_DEST="${INSTALL_DEST}" \
        MOCK_INSTALL_FAIL="${TEST_INSTALL_FAIL}" \
        REAL_MKTEMP="${REAL_MKTEMP}" \
        MOCK_OS="${TEST_OS}" \
        MOCK_ARCH="${TEST_ARCH}" \
        MOCK_STABLE="${TEST_STABLE}" \
        MOCK_STABLE_FAIL="${TEST_STABLE_FAIL}" \
        MOCK_MANIFEST="${MANIFEST_FILE}" \
        MOCK_MANIFEST_FAIL="${TEST_MANIFEST_FAIL}" \
        MOCK_ARTIFACT="${ARTIFACT_FILE}" \
        MOCK_ARTIFACT_FAIL="${TEST_ARTIFACT_FAIL}" \
        MOCK_LOG="${LOG_FILE}" \
        REAL_SHA256SUM="${REAL_SHA256SUM}" \
        REAL_CHMOD="${REAL_CHMOD}" \
        REAL_MV="${REAL_MV}" \
        bash "${INSTALLER}" > "${STDOUT_FILE}" 2> "${STDERR_FILE}"
    CASE_STATUS=$?
    set -e
}

function fail_test() {
    echo "FAIL: $*" >&2
    echo "stdout:" >&2
    sed 's/^/  /' "${STDOUT_FILE}" >&2
    echo "stderr:" >&2
    sed 's/^/  /' "${STDERR_FILE}" >&2
    echo "commands:" >&2
    sed 's/^/  /' "${LOG_FILE}" >&2
    exit 1
}

function assert_success() {
    [ "${CASE_STATUS}" -eq 0 ] || fail_test "expected success, got status ${CASE_STATUS}"
}

function assert_cleaned() {
    [ -z "$(ls -A "${TEMP_ROOT}")" ] || fail_test "temporary download directory was not cleaned"
}

function assert_verified_first() {
    awk '
        /^sha256sum / { verified = 1 }
        /^(mv|chmod|sudo) / { if (!verified) exit 1; installed = 1 }
        END { if (!installed) exit 1 }
    ' "${LOG_FILE}" || fail_test "verification did not precede executable/installation operations"
}

function assert_failure() {
    [ "${CASE_STATUS}" -ne 0 ] || fail_test "expected failure"
    assert_cleaned
}

function assert_file_contains() {
    local file=$1
    local expected=$2
    grep -Fq -- "${expected}" "${file}" || fail_test "${file} does not contain: ${expected}"
}

function assert_log_lacks_install_operations() {
    if grep -Eq '^(chmod|mv|sudo) ' "${LOG_FILE}"; then
        fail_test "failure reached an executable or installation operation"
    fi
}

function pass_test() {
    TEST_COUNT=$((TEST_COUNT + 1))
    echo "PASS: $1"
}

for architecture in x86_64 arm64 aarch64; do
    for copy in true false unset; do
        reset_case
        TEST_ARCH=${architecture}
        TEST_COPY=${copy}
        run_case
        assert_success
        assert_verified_first
        assert_file_contains "${STDOUT_FILE}" "cohctl-${TEST_VERSION}-linux-$(normalized_arch): OK"
        if [ "${copy}" = false ]; then
            downloaded_path=$(sed -n 's/^cohctl downloaded to \([^,]*\),.*/\1/p' "${STDOUT_FILE}")
            [ -x "${downloaded_path}" ] || fail_test "no-copy did not retain an executable"
            cmp "${downloaded_path}" "${ARTIFACT_FILE}" || fail_test "retained artifact differs"
            RETAINED_DIRS+=("$(dirname "${downloaded_path}")")
            if grep -q '^sudo ' "${LOG_FILE}"; then fail_test "no-copy requested sudo"; fi
        else
            assert_cleaned
            [ -x "${INSTALL_DEST}" ] || fail_test "installation did not move verified executable"
            cmp "${INSTALL_DEST}" "${ARTIFACT_FILE}" || fail_test "installed artifact differs"
            assert_file_contains "${LOG_FILE}" "/usr/local/bin/"
        fi
        pass_test "Linux ${architecture} COPY=${copy} verification, installation, and cleanup"
    done
done

for architecture in x86_64 arm64; do
    reset_case
    TEST_ARCH=${architecture}
    TEST_COPY=true
    TEST_INSTALL_FAIL=true
    run_case
    assert_failure
    assert_verified_first
    [ ! -e "${INSTALL_DEST}" ] || fail_test "failed installation left an executable"
    pass_test "${architecture} installation-command failure cleans temporary output"
done

reset_case
TEST_MISSING_SHA=true
run_case
assert_failure
assert_log_lacks_install_operations
assert_file_contains "${STDERR_FILE}" "sha256sum is required"
pass_test "missing sha256sum fails closed"

for invalid_version in "" $'1.2.3\n1.2.4' ".1.2.3" "1.2.3/bad"; do
    reset_case
    TEST_STABLE="${invalid_version}"
    run_case
    assert_failure
    assert_log_lacks_install_operations
done
pass_test "empty, multiline, leading-separator, and invalid stable versions"

reset_case
TEST_STABLE=$'1.2.3\r\n'
run_case
assert_success
downloaded_path=$(sed -n 's/^cohctl downloaded to \([^,]*\),.*/\1/p' "${STDOUT_FILE}")
[ -x "${downloaded_path}" ] || fail_test "CRLF stable version did not install"
RETAINED_DIRS+=("$(dirname "${downloaded_path}")")
pass_test "CRLF stable version normalization"

reset_case
TEST_STABLE_FAIL=true
run_case
assert_failure
assert_log_lacks_install_operations
pass_test "stable version download failure"

reset_case
TEST_MANIFEST_FAIL=true
run_case
assert_failure
assert_log_lacks_install_operations
pass_test "manifest download failure"

reset_case
TEST_ARTIFACT_FAIL=true
run_case
assert_failure
assert_log_lacks_install_operations
pass_test "artifact download failure"

for manifest_mode in missing duplicate malformed mismatch; do
    reset_case
    TEST_MANIFEST_MODE=${manifest_mode}
    run_case
    assert_failure
    assert_log_lacks_install_operations
done
pass_test "missing, duplicate, malformed, and mismatched checksums"

reset_case
TEST_ARCH=s390x
run_case
assert_failure
assert_log_lacks_install_operations
assert_file_contains "${STDOUT_FILE}" "Unsupported architecture: s390x"
pass_test "unsupported Linux architecture"

reset_case
TEST_OS=FreeBSD
run_case
assert_failure
assert_log_lacks_install_operations
assert_file_contains "${STDOUT_FILE}" "For all other platforms"
pass_test "unsupported operating system"

reset_case
TEST_OS=Darwin
TEST_ARCH=arm64
TEST_STABLE_FAIL=true
run_case
assert_success
if grep -Fq "stable.txt" "${LOG_FILE}"; then
    fail_test "macOS path fetched stable.txt"
fi
assert_file_contains "${LOG_FILE}" \
    "Oracle-Coherence-CLI-1.10.1-darwin-arm64.pkg"
assert_file_contains "${LOG_FILE}" "open "
assert_file_contains "${STDOUT_FILE}" "Installing Coherence CLI 1.10.1 for Darwin/arm64"
pass_test "unchanged pinned macOS package flow"

echo "All ${TEST_COUNT} installer tests passed"
