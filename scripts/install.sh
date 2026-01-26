#!/usr/bin/env bash
set -e

# --- Configuration ---
GITHUB_REPO="idebeijer/kubert"
BINARY_NAME="kubert"
DEFAULT_BIN_DIR="/usr/local/bin"
BIN_DIR=${1:-"${DEFAULT_BIN_DIR}"}

info() { echo '[INFO] ' "$@"; }
fatal() { echo '[ERROR] ' "$@" >&2; exit 1; }

# 1. Detect OS
setup_verify_os() {
    OS=$(uname | tr '[:upper:]' '[:lower:]')
    case "${OS}" in
        darwin|linux) ;;
        *) fatal "Unsupported operating system ${OS}" ;;
    esac
}

# 2. Detect Arch
setup_verify_arch() {
    ARCH=$(uname -m)
    case ${ARCH} in
        arm64|aarch64) ARCH="arm64" ;;
        amd64|x86_64)  ARCH="amd64" ;;
        *) fatal "Unsupported architecture ${ARCH}" ;;
    esac
}

# 3. Verify Downloader
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

# 4. Setup Temp Space
setup_tmp() {
    TMP_DIR=$(mktemp -d -t kubert-install.XXXXXXXXXX)
    cleanup() { rm -rf "${TMP_DIR}"; }
    trap cleanup INT EXIT
}

# 5. Get Metadata & Versions
get_release_version() {
    METADATA_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    # Extract tag (e.g., v1.0.0)
    TAG=$(curl -s "${METADATA_URL}" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    # Remove 'v' prefix for the version string if necessary
    VERSION=${TAG#v}
}

# 6. Checksum Verification
verify_binary() {
    info "Verifying checksum..."
    HASH_URL="https://github.com/${GITHUB_REPO}/releases/download/${TAG}/${BINARY_NAME}_${VERSION}_checksums.txt"
    download "${TMP_DIR}/checksums.txt" "${HASH_URL}"
    
    # Standard GoReleaser archive name pattern
    FILENAME="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
    EXPECTED_HASH=$(grep "${FILENAME}" "${TMP_DIR}/checksums.txt" | cut -d ' ' -f 1)
    
    ACTUAL_HASH=$(shasum -a 256 "${TMP_DIR}/${FILENAME}" | cut -d ' ' -f 1)
    
    if [[ "${EXPECTED_HASH}" != "${ACTUAL_HASH}" ]]; then
        fatal "Checksum mismatch! Expected ${EXPECTED_HASH} but got ${ACTUAL_HASH}"
    fi
}

# 7. Download and Extract
download_and_install() {
    FILENAME="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
    URL="https://github.com/${GITHUB_REPO}/releases/download/${TAG}/${FILENAME}"
    
    info "Downloading ${URL}..."
    download "${TMP_DIR}/${FILENAME}" "${URL}"
    
    verify_binary
    
    tar -xzf "${TMP_DIR}/${FILENAME}" -C "${TMP_DIR}"
    chmod +x "${TMP_DIR}/${BINARY_NAME}"
    
    info "Installing to ${BIN_DIR}..."
    if [[ -w "${BIN_DIR}" ]]; then
        mv "${TMP_DIR}/${BINARY_NAME}" "${BIN_DIR}/"
    else
        sudo mv "${TMP_DIR}/${BINARY_NAME}" "${BIN_DIR}/"
    fi
}

# --- Execute ---
{
    setup_verify_os
    setup_verify_arch
    verify_downloader
    setup_tmp
    get_release_version
    download_and_install
    info "Successfully installed kubert ${TAG}!"
}