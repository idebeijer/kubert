#!/usr/bin/env bash
set -e

# --- Configuration ---
GITHUB_REPO="idebeijer/kubert"
BINARY_NAME="kubert"

info() { echo '[INFO] ' "$@"; }
fatal() { echo '[ERROR] ' "$@" >&2; exit 1; }

# 1. Verify Dependencies
check_cosign() {
    if ! command -v cosign >/dev/null 2>&1; then
        fatal "cosign is required but not installed. See https://docs.sigstore.dev/cosign/system_config/installation/"
    fi
}

# 2. Verify Downloader
verify_downloader() {
    if command -v curl >/dev/null 2>&1; then
        DOWNLOADER="curl"
    elif command -v wget >/dev/null 2>&1; then
        DOWNLOADER="wget"
    else
        fatal "Can not find curl or wget"
    fi
}

download() {
    case $DOWNLOADER in
        curl) curl -sfL "$2" -o "$1" ;;
        wget) wget -qO "$1" "$2" ;;
    esac
}

# 3. Setup Temp Space
setup_tmp() {
    TMP_DIR=$(mktemp -d -t kubert-verify.XXXXXXXXXX)
    cleanup() { rm -rf "${TMP_DIR}"; }
    trap cleanup INT EXIT
}

# 4. Get Metadata & Versions
get_release_version() {
    METADATA_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    TAG=$(curl -s "${METADATA_URL}" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    VERSION=${TAG#v}
    if [ -z "${TAG}" ]; then
        fatal "Failed to determine latest release version"
    fi
}

# 5. Verify Checksums Blob
verify_blob() {
    CHECKSUMS_FILE="${BINARY_NAME}_${VERSION}_checksums.txt"
    BUNDLE_FILE="${CHECKSUMS_FILE}.sigstore.json"
    BASE_URL="https://github.com/${GITHUB_REPO}/releases/download/${TAG}"

    info "Downloading ${CHECKSUMS_FILE}..."
    download "${TMP_DIR}/${CHECKSUMS_FILE}" "${BASE_URL}/${CHECKSUMS_FILE}"

    info "Downloading ${BUNDLE_FILE}..."
    download "${TMP_DIR}/${BUNDLE_FILE}" "${BASE_URL}/${BUNDLE_FILE}"

    info "Verifying signature of ${CHECKSUMS_FILE} with cosign..."
    cosign verify-blob \
        --certificate-identity "https://github.com/${GITHUB_REPO}/.github/workflows/release.yml@refs/tags/${TAG}" \
        --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
        --bundle "${TMP_DIR}/${BUNDLE_FILE}" \
        "${TMP_DIR}/${CHECKSUMS_FILE}"

    info "Signature verification successful for ${CHECKSUMS_FILE} (${TAG})"
}

# --- Execute ---
{
    check_cosign
    verify_downloader
    setup_tmp
    get_release_version
    verify_blob
}