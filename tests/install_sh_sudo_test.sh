#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
TMP_DIR=$(mktemp -d)
cleanup() {
    chmod -R u+w "${TMP_DIR}" 2>/dev/null || true
    rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

FAKE_BIN="${TMP_DIR}/bin"
FIXTURES="${TMP_DIR}/fixtures"
mkdir -p "${FAKE_BIN}" "${FIXTURES}"

ARCHIVE="flashduty-cli_Linux_x86_64.tar.gz"

make_fixtures() {
    cli_dir="${TMP_DIR}/cli"
    mkdir -p "${cli_dir}"
    cat > "${cli_dir}/flashduty-cli" <<'EOS'
#!/bin/sh
echo "fake flashduty $*"
EOS
    chmod +x "${cli_dir}/flashduty-cli"
    (cd "${cli_dir}" && tar czf "${FIXTURES}/${ARCHIVE}" flashduty-cli)
    if command -v sha256sum >/dev/null 2>&1; then
        sum=$(sha256sum "${FIXTURES}/${ARCHIVE}" | awk '{print $1}')
    else
        sum=$(shasum -a 256 "${FIXTURES}/${ARCHIVE}" | awk '{print $1}')
    fi
    printf '%s  %s\n' "${sum}" "${ARCHIVE}" > "${FIXTURES}/checksums.txt"
}

make_fake_commands() {
    cat > "${FAKE_BIN}/uname" <<'EOS'
#!/bin/sh
case "$1" in
    -s) echo Linux ;;
    -m) echo x86_64 ;;
    *) /usr/bin/uname "$@" ;;
esac
EOS
    chmod +x "${FAKE_BIN}/uname"

    cat > "${FAKE_BIN}/curl" <<EOS
#!/bin/sh
out=""
url=""
while [ "\$#" -gt 0 ]; do
    case "\$1" in
        --proto) shift 2 ;;
        --tlsv1.2|-fsSL) shift ;;
        -o) out="\$2"; shift 2 ;;
        -*) shift ;;
        *) url="\$1"; shift ;;
    esac
done

case "\${url}" in
    */${ARCHIVE}) src="${FIXTURES}/${ARCHIVE}" ;;
    */checksums.txt) src="${FIXTURES}/checksums.txt" ;;
    *) echo "unexpected curl URL: \${url}" >&2; exit 22 ;;
esac

if [ -n "\${out}" ]; then
    cp "\${src}" "\${out}"
else
    cat "\${src}"
fi
EOS
    chmod +x "${FAKE_BIN}/curl"

    cat > "${FAKE_BIN}/sudo" <<'EOS'
#!/bin/sh
printf '%s\n' "sudo $*" >> "${SUDO_LOG}"
if [ "${1:-}" = "-n" ] && [ "${2:-}" = "true" ]; then
    exit 1
fi
if [ "${1:-}" = "mv" ]; then
    dir=$(dirname -- "$3")
    chmod u+w "${dir}" 2>/dev/null || true
    mv "$2" "$3"
    chmod a-w "${dir}" 2>/dev/null || true
    exit 0
fi
exec "$@"
EOS
    chmod +x "${FAKE_BIN}/sudo"
}

run_install() {
    home_dir="$1"
    install_dir="$2"
    out_file="$3"
    env PATH="${FAKE_BIN}:$PATH" \
        HOME="${home_dir}" \
        SHELL=/bin/false \
        FLASHDUTY_VERSION=v9.9.9 \
        MIRROR_URL=https://mirror.example/flashduty-cli \
        FLASHDUTY_INSTALL_DIR="${install_dir}" \
        SUDO_LOG="${SUDO_LOG}" \
        sh < "${ROOT}/install.sh" > "${out_file}" 2>&1
}

run_install_with_tty() {
    home_dir="$1"
    install_dir="$2"
    out_file="$3"
    runner="${TMP_DIR}/run-install-with-tty.sh"
    cat > "${runner}" <<EOS
#!/bin/sh
exec env PATH="${FAKE_BIN}:\$PATH" \\
    HOME="${home_dir}" \\
    SHELL=/bin/false \\
    FLASHDUTY_VERSION=v9.9.9 \\
    MIRROR_URL=https://mirror.example/flashduty-cli \\
    FLASHDUTY_INSTALL_DIR="${install_dir}" \\
    SUDO_LOG="${SUDO_LOG}" \\
    sh < "${ROOT}/install.sh"
EOS
    chmod +x "${runner}"

    if script --version >/dev/null 2>&1; then
        script -q -e -c "${runner}" /dev/null > "${out_file}" 2>&1
    else
        script -q /dev/null "${runner}" > "${out_file}" 2>&1
    fi
}

assert_file_exists() {
    if [ ! -e "$1" ]; then
        echo "expected file to exist: $1" >&2
        exit 1
    fi
}

assert_file_missing() {
    if [ -e "$1" ]; then
        echo "expected file to be absent: $1" >&2
        exit 1
    fi
}

assert_contains() {
    if ! grep -Fq "$2" "$1"; then
        echo "expected $1 to contain: $2" >&2
        echo "--- $1 ---" >&2
        cat "$1" >&2
        exit 1
    fi
}

test_non_tty_falls_back_to_user_bin() {
    SUDO_LOG="${TMP_DIR}/sudo-non-tty.log"
    export SUDO_LOG
    : > "${SUDO_LOG}"
    home_dir="${TMP_DIR}/home-non-tty"
    install_dir="${TMP_DIR}/system-non-tty"
    out_file="${TMP_DIR}/non-tty.out"
    mkdir -p "${home_dir}" "${install_dir}"
    chmod 555 "${install_dir}"

    run_install "${home_dir}" "${install_dir}" "${out_file}"

    assert_contains "${out_file}" "Install dir not writable and no passwordless sudo; installing to ${home_dir}/.local/bin"
    assert_file_exists "${home_dir}/.local/bin/flashduty"
    assert_file_missing "${install_dir}/flashduty"
}

test_tty_prompts_for_interactive_sudo() {
    SUDO_LOG="${TMP_DIR}/sudo-tty.log"
    export SUDO_LOG
    : > "${SUDO_LOG}"
    home_dir="${TMP_DIR}/home-tty"
    install_dir="${TMP_DIR}/system-tty"
    out_file="${TMP_DIR}/tty.out"
    mkdir -p "${home_dir}" "${install_dir}"
    chmod 555 "${install_dir}"

    run_install_with_tty "${home_dir}" "${install_dir}" "${out_file}"

    assert_contains "${out_file}" "Need elevated permissions to install to ${install_dir}"
    assert_contains "${SUDO_LOG}" "sudo mv"
    assert_file_exists "${install_dir}/flashduty"
    assert_file_missing "${home_dir}/.local/bin/flashduty"
}

make_fixtures
make_fake_commands
test_non_tty_falls_back_to_user_bin
test_tty_prompts_for_interactive_sudo
