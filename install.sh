#!/bin/sh
set -e

REPO="JungHoonGhae/tossinvest-cli"
BINARY="tossctl"
INSTALL_DIR="/usr/local/bin"
SHARE_DIR="/usr/local/share/tossctl"

main() {
    os=$(detect_os)
    arch=$(detect_arch)

    if [ "$os" = "windows" ]; then
        echo "Error: this script does not support Windows. Use PowerShell instead:"
        echo '  Invoke-WebRequest -Uri "https://github.com/'"$REPO"'/releases/latest/download/tossctl-windows-amd64.zip" -OutFile tossctl.zip'
        echo '  Expand-Archive tossctl.zip -DestinationPath .'
        exit 1
    fi

    asset="${BINARY}-${os}-${arch}.tar.gz"
    url="https://github.com/${REPO}/releases/latest/download/${asset}"
    sha_url="${url}.sha256"

    echo "Detected: ${os}/${arch}"
    echo "Downloading ${asset}..."

    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    curl -fsSL -o "${tmpdir}/${asset}" "$url"
    curl -fsSL -o "${tmpdir}/${asset}.sha256" "$sha_url"

    echo "Verifying checksum..."
    (cd "$tmpdir" && verify_checksum "${asset}")

    echo "Extracting..."
    tar -xzf "${tmpdir}/${asset}" -C "$tmpdir"

    echo "Installing to ${INSTALL_DIR}..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
        chmod +x "${INSTALL_DIR}/${BINARY}"
        mkdir -p "${SHARE_DIR}"
        if [ -d "${tmpdir}/auth-helper" ]; then
            rm -rf "${SHARE_DIR}/auth-helper"
            mv "${tmpdir}/auth-helper" "${SHARE_DIR}/auth-helper"
        fi
    else
        sudo mv "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY}"
        sudo mkdir -p "${SHARE_DIR}"
        if [ -d "${tmpdir}/auth-helper" ]; then
            sudo rm -rf "${SHARE_DIR}/auth-helper"
            sudo mv "${tmpdir}/auth-helper" "${SHARE_DIR}/auth-helper"
        fi
    fi

    echo ""
    echo "Installing Python dependencies for auth-helper..."
    if command -v python3 >/dev/null 2>&1; then
        python3 -m pip install --quiet playwright 2>/dev/null || echo "Warning: failed to install playwright. Run 'python3 -m pip install playwright' manually."
    else
        echo "Warning: python3 not found. Install Python 3.11+ and run 'python3 -m pip install playwright'."
    fi

    echo ""
    echo "Installed $(${INSTALL_DIR}/${BINARY} version 2>/dev/null || echo "${BINARY}") to ${INSTALL_DIR}/${BINARY}"
    echo "Auth helper installed to ${SHARE_DIR}/auth-helper"
    echo ""
    echo "Next steps:"
    echo "  tossctl doctor"
    echo "  tossctl auth login"
}

detect_os() {
    case "$(uname -s)" in
        Darwin*)  echo "darwin" ;;
        Linux*)   echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)        echo "Error: unsupported OS: $(uname -s)" >&2; exit 1 ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64" ;;
        arm64|aarch64)  echo "arm64" ;;
        *)              echo "Error: unsupported architecture: $(uname -m)" >&2; exit 1 ;;
    esac
}

verify_checksum() {
    file="$1"
    expected=$(awk '{print $1}' "${file}.sha256")
    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        echo "Warning: no sha256 tool found, skipping checksum verification" >&2
        return 0
    fi
    if [ "$expected" != "$actual" ]; then
        echo "Error: checksum mismatch" >&2
        echo "  expected: $expected" >&2
        echo "  actual:   $actual" >&2
        exit 1
    fi
}

main
