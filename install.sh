#!/bin/sh
# Flashduty CLI installer
# Usage: curl -sSL https://raw.githubusercontent.com/flashcatcloud/flashduty-cli/main/install.sh | sh
#
# Environment:
#   FLASHDUTY_VERSION      Install a specific version (e.g. v0.1.2). Default: latest.
#   FLASHDUTY_INSTALL_DIR  Install directory. Default: /usr/local/bin.
#   MIRROR_URL             Fetch release assets from this https mirror prefix
#                          instead of github.com. The mirror must replicate
#                          GitHub's release layout
#                          (<MIRROR_URL>/releases/download/<tag>/<asset>) and expose
#                          a plain-text <MIRROR_URL>/releases/latest file containing
#                          the latest tag.
set -e

REPO="flashcatcloud/flashduty-cli"
BINARY="flashduty-cli"
INSTALLED_NAME="flashduty"
INSTALL_DIR="${FLASHDUTY_INSTALL_DIR:-/usr/local/bin}"

# When set, all release downloads are fetched from this prefix instead of github.com.
MIRROR_URL="${MIRROR_URL:-}"
MIRROR_URL="${MIRROR_URL%/}"
if [ -n "${MIRROR_URL}" ]; then
    case "${MIRROR_URL}" in
        https://*) : ;;
        *) printf "Error: MIRROR_URL must use https:// scheme, got: %s\n" "${MIRROR_URL}" >&2; exit 1 ;;
    esac
fi

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

sha256_of() {
    file="$1"
    if command -v sha256sum > /dev/null 2>&1; then
        sha256sum "${file}" | awk '{print $1}'
    elif command -v shasum > /dev/null 2>&1; then
        shasum -a 256 "${file}" | awk '{print $1}'
    else
        fail "need 'sha256sum' or 'shasum' to verify the download (install coreutils)"
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
    if [ -n "${MIRROR_URL}" ]; then
        # The mirror publishes a plain-text pointer with the latest tag.
        version=$(curl --proto '=https' --tlsv1.2 -fsSL "${MIRROR_URL}/releases/latest" 2>/dev/null \
            | awk 'NR==1 {gsub(/^[[:space:]]+|[[:space:]]+$/, ""); print; exit}')
    else
        # Follow the github.com/<repo>/releases/latest redirect to read the tag
        # from the resolved URL — avoids the unauthenticated api.github.com rate limit.
        effective=$(curl --proto '=https' --tlsv1.2 -sIL -o /dev/null -w '%{url_effective}' \
            "https://github.com/${REPO}/releases/latest" || true)
        version="${effective##*/}"
        [ "${version}" = "latest" ] && version=""
    fi
    if [ -z "${version}" ]; then
        fail "could not determine latest version. Set FLASHDUTY_VERSION to install a specific version."
    fi
    # Reject anything that doesn't look like a release tag — the resolved value
    # comes from a network response and is interpolated into the download URL.
    case "${version}" in
        *[!A-Za-z0-9.+-]*) fail "resolved version contains illegal characters: '${version}'" ;;
    esac
    case "${version}" in
        v[0-9]*) : ;;
        *) fail "resolved version is not a valid release tag: '${version}'" ;;
    esac
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
    if [ -n "${MIRROR_URL}" ]; then
        BASE="${MIRROR_URL}/releases/download/${VERSION}"
    else
        BASE="https://github.com/${REPO}/releases/download/${VERSION}"
    fi

    info "Installing Flashduty CLI ${VERSION} (${OS}/${ARCH})"
    info "Downloading ${BASE}/${ARCHIVE}"

    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "${TMP_DIR}"' EXIT

    if ! curl --proto '=https' --tlsv1.2 -fsSL "${BASE}/${ARCHIVE}" -o "${TMP_DIR}/${ARCHIVE}"; then
        fail "download failed for ${BASE}/${ARCHIVE}. Check that ${VERSION} exists."
    fi

    # Verify against the published checksums.txt when present. Releases cut
    # before the mirror existed don't ship one, so a missing file only warns.
    if curl --proto '=https' --tlsv1.2 -fsSL "${BASE}/checksums.txt" -o "${TMP_DIR}/checksums.txt" 2>/dev/null; then
        expected=$(awk -v a="${ARCHIVE}" '$2 == a {print $1; exit}' "${TMP_DIR}/checksums.txt")
        if [ -z "${expected}" ]; then
            fail "archive ${ARCHIVE} not listed in checksums.txt (wrong release or renamed asset)"
        fi
        actual=$(sha256_of "${TMP_DIR}/${ARCHIVE}")
        if [ "${actual}" != "${expected}" ]; then
            fail "checksum mismatch for ${ARCHIVE}: expected ${expected}, got ${actual}"
        fi
        info "Checksum OK"
    else
        info "WARNING: checksums.txt not available — skipping integrity check"
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

    # Warn if install dir is not in PATH
    case ":${PATH}:" in
        *":${INSTALL_DIR}:"*) ;;
        *) info "WARNING: ${INSTALL_DIR} is not in your PATH. Add it with:"
           info "  export PATH=\"${INSTALL_DIR}:\$PATH\"" ;;
    esac

    info "Run 'flashduty version' to verify"
}

main
