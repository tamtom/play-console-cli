#!/usr/bin/env bash
set -euo pipefail

REPO="tamtom/play-console-cli"
BIN_NAME="gplay"
DEFAULT_INSTALL_DIR="/usr/local/bin"
if [ -n "${HOME:-}" ]; then
  DEFAULT_INSTALL_DIR="${HOME}/.local/bin"
fi
INSTALL_DIR="${GPLAY_INSTALL_DIR:-${DEFAULT_INSTALL_DIR}}"

OS="$(uname -s)"
ARCH="$(uname -m)"

case "${OS}" in
  Darwin) OS="darwin" ;;
  Linux) OS="linux" ;;
  *)
    echo "Unsupported OS: ${OS}"
    exit 1
    ;;
esac

case "${ARCH}" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: ${ARCH}"
    exit 1
    ;;
esac

# Use specific version or fetch latest
if [ -n "${GPLAY_VERSION:-}" ]; then
  VERSION="${GPLAY_VERSION}"
  BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
else
  BASE_URL="https://github.com/${REPO}/releases/latest/download"
fi

ASSET="${BIN_NAME}-${OS}-${ARCH}"
BIN_URL="${BASE_URL}/${ASSET}"
CHECKSUMS_URL="${BASE_URL}/checksums.txt"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

echo "Downloading ${ASSET}..."
curl -fsSL "${BIN_URL}" -o "${TMP_DIR}/${ASSET}"

# Every installation must have exactly one valid checksum for this asset.
if ! curl -fsSL "${CHECKSUMS_URL}" -o "${TMP_DIR}/checksums.txt"; then
  echo "Cannot download checksums.txt; refusing to install an unverified binary." >&2
  exit 1
fi
EXPECTED="$(awk -v asset="${ASSET}" '
  NF == 2 { name=$2; sub(/^\*/, "", name); if (name == asset) { count++; digest=$1 } }
  END { if (count != 1 || length(digest) != 64 || digest ~ /[^0-9a-fA-F]/) exit 1; print tolower(digest) }
' "${TMP_DIR}/checksums.txt")" || {
  echo "Missing, duplicate, or invalid SHA-256 checksum for ${ASSET}." >&2
  exit 1
}
if command -v shasum >/dev/null 2>&1; then
  ACTUAL="$(shasum -a 256 "${TMP_DIR}/${ASSET}" | awk '{print $1}')"
elif command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "${TMP_DIR}/${ASSET}" | awk '{print $1}')"
else
  echo "A SHA-256 tool (shasum or sha256sum) is required; refusing to install." >&2
  exit 1
fi
if [ "${EXPECTED}" != "${ACTUAL}" ]; then
  echo "Checksum verification failed." >&2
  exit 1
fi
echo "Checksum verified."

# Create install directory
if ! mkdir -p "${INSTALL_DIR}" 2>/dev/null; then
  if command -v sudo >/dev/null 2>&1; then
    sudo mkdir -p "${INSTALL_DIR}"
  else
    echo "Cannot create ${INSTALL_DIR}; try running with sudo or set GPLAY_INSTALL_DIR."
    exit 1
  fi
fi

# Install binary
if [ -w "${INSTALL_DIR}" ]; then
  install -m 755 "${TMP_DIR}/${ASSET}" "${INSTALL_DIR}/${BIN_NAME}"
else
  if command -v sudo >/dev/null 2>&1; then
    sudo install -m 755 "${TMP_DIR}/${ASSET}" "${INSTALL_DIR}/${BIN_NAME}"
  else
    echo "Cannot write to ${INSTALL_DIR}; try running with sudo or set GPLAY_INSTALL_DIR."
    exit 1
  fi
fi

echo "Installed ${BIN_NAME} to ${INSTALL_DIR}/${BIN_NAME}"
echo "Run: ${BIN_NAME} --help"

if [[ ":${PATH}:" != *":${INSTALL_DIR}:"* ]]; then
  echo ""
  echo "Note: ${INSTALL_DIR} is not in your PATH."
  echo "Add it to your shell profile:"
  echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
fi
