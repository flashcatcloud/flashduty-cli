#!/bin/sh
# Flashduty CLI installer
# Usage: curl -sSL https://static.flashcat.cloud/flashduty-cli/install.sh | sh
#
# Environment:
#   FLASHDUTY_VERSION      Install a specific version (e.g. v0.1.2). Default: latest.
#   FLASHDUTY_INSTALL_DIR  Install directory. Default: /usr/local/bin.
#   MIRROR_URL             Fetch release assets from this https mirror prefix.
#                          Default: https://static.flashcat.cloud/flashduty-cli.
#                          The mirror must replicate
#                          GitHub's release layout
#                          (<MIRROR_URL>/releases/download/<tag>/<asset>) and expose
#                          a plain-text <MIRROR_URL>/releases/latest file containing
#                          the latest tag.
set -e

REPO="flashcatcloud/flashduty-cli"
BINARY="flashduty-cli"
INSTALLED_NAME="${INSTALLED_NAME:-flashduty}"
INSTALL_DIR="${FLASHDUTY_INSTALL_DIR:-/usr/local/bin}"

# By default release downloads are fetched from the Flashcat CDN. Set MIRROR_URL
# to another prefix to override, or to an empty string to force GitHub fallback.
DEFAULT_MIRROR_URL="https://static.flashcat.cloud/flashduty-cli"
MIRROR_URL="${MIRROR_URL-${DEFAULT_MIRROR_URL}}"
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

can_prompt_for_sudo() {
    command -v sudo > /dev/null 2>&1 || return 1
    # `curl | sh` leaves stdin as the script pipe, but sudo can still prompt via
    # the controlling terminal. In CI/agent contexts stderr is usually not a TTY,
    # so do not risk an unanswerable password prompt there.
    [ -t 2 ] || return 1
    [ -r /dev/tty ] || return 1
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

# --- shell completion (best-effort, non-intrusive) ---

# The user's interactive shell decides which completion script we need —
# completion varies by shell, not by OS/arch. Empty when it's not one we support.
detect_shell() {
    case "$(basename "${SHELL:-}" 2>/dev/null)" in
        bash) echo "bash" ;;
        zsh)  echo "zsh" ;;
        fish) echo "fish" ;;
        *)    echo "" ;;
    esac
}

# Emit the completion script for the shell named in $1. Cobra bakes the root
# command name "flashduty" into the script (#compdef / complete -c / function
# names); when installed under a different name, rewrite every occurrence so the
# completion binds to the actual command (the runtime dispatch line already uses
# the typed command word, so it needs no rewrite). The `|` sed delimiter is safe
# because a binary name can't contain it, and the rewrite is a no-op for the
# default "flashduty".
gen_completion() {
    "${BIN}" completion "$1" | sed "s|flashduty|${INSTALLED_NAME}|g"
}

# Install completion for the current shell into a directory the shell already
# auto-loads, without ever editing the user's rc files. zsh has no guaranteed
# writable fpath dir, so it only succeeds when a standard site-functions dir is
# already writable (e.g. a Homebrew install); otherwise we point at the binary's
# own per-shell setup instructions.
setup_completion() {
    [ "${OS}" = "Windows" ] && return 0
    sh_name=$(detect_shell)
    [ -z "${sh_name}" ] && return 0
    "${BIN}" completion "${sh_name}" >/dev/null 2>&1 || return 0

    case "${sh_name}" in
        fish)
            dir="${XDG_CONFIG_HOME:-${HOME}/.config}/fish/completions"
            mkdir -p "${dir}" 2>/dev/null || true
            if [ -w "${dir}" ]; then
                gen_completion fish > "${dir}/${INSTALLED_NAME}.fish" && {
                    info "Installed fish completion to ${dir}/${INSTALLED_NAME}.fish (restart fish to load)"
                    return 0
                }
            fi
            ;;
        bash)
            dir="${XDG_DATA_HOME:-${HOME}/.local/share}/bash-completion/completions"
            mkdir -p "${dir}" 2>/dev/null || true
            if [ -w "${dir}" ]; then
                gen_completion bash > "${dir}/${INSTALLED_NAME}" && {
                    info "Installed bash completion to ${dir}/${INSTALLED_NAME} (needs the bash-completion package; restart bash to load)"
                    return 0
                }
            fi
            ;;
        zsh)
            for dir in \
                "${HOMEBREW_PREFIX:-/opt/homebrew}/share/zsh/site-functions" \
                "/usr/local/share/zsh/site-functions" \
                "/usr/share/zsh/site-functions"; do
                if [ -d "${dir}" ] && [ -w "${dir}" ]; then
                    gen_completion zsh > "${dir}/_${INSTALLED_NAME}" && {
                        info "Installed zsh completion to ${dir}/_${INSTALLED_NAME}"
                        info "  Run 'rm -f ~/.zcompdump*' and restart zsh to load."
                        return 0
                    }
                fi
            done
            ;;
    esac

    # Couldn't auto-install into an auto-loaded dir (the common zsh case: no
    # writable fpath dir, and we never edit ~/.zshrc). Print the exact,
    # copy-pasteable steps so the user can finish setup in one go.
    print_manual_completion "${sh_name}"
}

# Print a concrete, copy-pasteable recipe to enable completion for $1, used when
# setup_completion can't drop the script into an auto-loaded directory. Plain
# stdout (no "[flashduty]" prefix) so the commands paste cleanly.
print_manual_completion() {
    name="${INSTALLED_NAME}"
    info "Shell completion was not auto-installed. To enable it for $1, run:"
    case "$1" in
        zsh)
            cat <<EOF

  mkdir -p ~/.zsh/completions
  ${name} completion zsh > ~/.zsh/completions/_${name}
  echo 'fpath=(~/.zsh/completions \$fpath)' >> ~/.zshrc   # one-time
  rm -f ~/.zcompdump* && exec zsh

EOF
            ;;
        bash)
            cat <<EOF

  mkdir -p ~/.local/share/bash-completion/completions
  ${name} completion bash > ~/.local/share/bash-completion/completions/${name}
  # requires the bash-completion package; then restart bash

EOF
            ;;
        fish)
            cat <<EOF

  mkdir -p ~/.config/fish/completions
  ${name} completion fish > ~/.config/fish/completions/${name}.fish
  # then restart fish

EOF
            ;;
    esac
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

    # Install (rename flashduty-cli -> flashduty for convenience).
    # Create the target dir first so a caller-provided FLASHDUTY_INSTALL_DIR that
    # doesn't exist yet is usable without sudo.
    mkdir -p "${INSTALL_DIR}" 2>/dev/null || true
    if [ -w "${INSTALL_DIR}" ]; then
        mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${INSTALLED_NAME}"
        chmod +x "${INSTALL_DIR}/${INSTALLED_NAME}"
    elif sudo -n true 2>/dev/null; then
        # Passwordless sudo is available — install to the privileged dir without
        # prompting.
        sudo mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${INSTALLED_NAME}"
        sudo chmod +x "${INSTALL_DIR}/${INSTALLED_NAME}"
    elif can_prompt_for_sudo; then
        info "Need elevated permissions to install to ${INSTALL_DIR}"
        sudo mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${INSTALLED_NAME}"
        sudo chmod +x "${INSTALL_DIR}/${INSTALLED_NAME}"
    else
        # Not writable and sudo would need an interactive password. Never block on
        # an unanswerable prompt — fall back to a user-writable directory.
        INSTALL_DIR="${HOME}/.local/bin"
        info "Install dir not writable and no passwordless sudo; installing to ${INSTALL_DIR}"
        mkdir -p "${INSTALL_DIR}"
        mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${INSTALLED_NAME}"
        chmod +x "${INSTALL_DIR}/${INSTALLED_NAME}"
    fi

    info "Installed to ${INSTALL_DIR}/${INSTALLED_NAME}"

    # Warn if install dir is not in PATH
    case ":${PATH}:" in
        *":${INSTALL_DIR}:"*) ;;
        *) info "WARNING: ${INSTALL_DIR} is not in your PATH. Add it with:"
           info "  export PATH=\"${INSTALL_DIR}:\$PATH\"" ;;
    esac

    # Best-effort shell completion; never fail the install over it.
    BIN="${INSTALL_DIR}/${INSTALLED_NAME}"
    setup_completion || true

    info "Run '${INSTALLED_NAME} version' to verify"
}

main
