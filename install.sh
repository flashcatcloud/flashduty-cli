#!/bin/sh
# Flashduty CLI installer
# Usage: curl -sSL https://raw.githubusercontent.com/flashcatcloud/flashduty-cli/main/install.sh | sh
set -e

REPO="flashcatcloud/flashduty-cli"
BINARY="flashduty-cli"
INSTALLED_NAME="flashduty"
INSTALL_DIR="${FLASHDUTY_INSTALL_DIR:-/usr/local/bin}"

# --- helper functions ---

fail() {
    printf "Error: %s\n" "$1" >&2
    exit 1
}

info() {
    printf "[flashduty] %s\n" "$1"
}

need_cmd() {
    if ! command -v "$1" > /dev/null 2>&1; then
        fail "need '$1' (command not found)"
    fi
}

# --- detect platform ---

detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "Linux" ;;
        Darwin*) echo "Darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "Windows" ;;
        *) fail "unsupported OS: $(uname -s)" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "x86_64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) fail "unsupported architecture: $(uname -m)" ;;
    esac
}

# --- resolve version ---

resolve_version() {
    if [ -n "${FLASHDUTY_VERSION}" ]; then
        echo "${FLASHDUTY_VERSION}"
        return
    fi
    need_cmd curl
    # Use the GitHub API to get the latest release tag
    version=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    if [ -z "${version}" ]; then
        fail "could not determine latest version. Set FLASHDUTY_VERSION to install a specific version."
    fi
    echo "${version}"
}

# --- main ---

main() {
    need_cmd uname
    need_cmd tar
    need_cmd curl

    OS=$(detect_os)
    ARCH=$(detect_arch)
    VERSION=$(resolve_version)

    if [ "${OS}" = "Windows" ]; then
        EXT="zip"
        need_cmd unzip
    else
        EXT="tar.gz"
    fi

    ARCHIVE="flashduty-cli_${OS}_${ARCH}.${EXT}"
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

    info "Installing Flashduty CLI ${VERSION} (${OS}/${ARCH})"
    info "Downloading ${URL}"

    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "${TMP_DIR}"' EXIT

    HTTP_CODE=$(curl -sL -H "Accept: application/octet-stream" -o "${TMP_DIR}/${ARCHIVE}" -w "%{http_code}" "${URL}")
    if [ "${HTTP_CODE}" != "200" ]; then
        fail "download failed (HTTP ${HTTP_CODE}). Check that ${VERSION} exists at https://github.com/${REPO}/releases"
    fi

    if [ "${EXT}" = "zip" ]; then
        unzip -q "${TMP_DIR}/${ARCHIVE}" -d "${TMP_DIR}"
    else
        tar xzf "${TMP_DIR}/${ARCHIVE}" -C "${TMP_DIR}"
    fi

    if [ ! -f "${TMP_DIR}/${BINARY}" ]; then
        fail "binary '${BINARY}' not found in archive"
    fi

    # Install (rename flashduty-cli -> flashduty for convenience)
    if [ -w "${INSTALL_DIR}" ]; then
        mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${INSTALLED_NAME}"
    else
        info "Need elevated permissions to install to ${INSTALL_DIR}"
        sudo mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${INSTALLED_NAME}"
    fi
    chmod +x "${INSTALL_DIR}/${INSTALLED_NAME}"

    info "Installed to ${INSTALL_DIR}/${INSTALLED_NAME}"
    info "Run 'flashduty version' to verify"
}

main
