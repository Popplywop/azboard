#!/usr/bin/env sh
set -e

REPO="Popplywop/azboard"
BINARY="azboard"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS="$(uname -s)"
case "${OS}" in
  Linux*)   OS=linux ;;
  Darwin*)  OS=darwin ;;
  *)
    echo "Unsupported OS: ${OS}"
    exit 1
    ;;
esac

# Detect architecture (or allow override via INSTALL_ARCH)
ARCH_RAW="${INSTALL_ARCH:-$(uname -m)}"
case "${ARCH_RAW}" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *)
    echo "Unsupported architecture: ${ARCH_RAW}"
    echo "Supported values: amd64, arm64"
    exit 1
    ;;
esac

# Fetch latest release version from GitHub API
echo "Fetching latest release..."
VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' \
  | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"

if [ -z "${VERSION}" ]; then
  echo "Failed to determine latest version. Check your internet connection."
  exit 1
fi

# Strip leading 'v' for the filename (goreleaser uses bare version in filenames)
VERSION_NUM="${VERSION#v}"

ARCHIVE="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
URL="${BASE_URL}/${ARCHIVE}"

echo "Installing ${BINARY} ${VERSION} (${OS}/${ARCH})..."

# Download to a temp dir
TMP="$(mktemp -d)"
trap 'rm -rf "${TMP}"' EXIT

curl -fsSL "${URL}"                              -o "${TMP}/${ARCHIVE}"
curl -fsSL "${URL}.sig"                         -o "${TMP}/${ARCHIVE}.sig"
curl -fsSL "${URL}.pem"                         -o "${TMP}/${ARCHIVE}.pem"

# Verify cosign signature if cosign is available
if command -v cosign >/dev/null 2>&1; then
  echo "Verifying signature with cosign..."
  cosign verify-blob \
    --certificate         "${TMP}/${ARCHIVE}.pem" \
    --signature           "${TMP}/${ARCHIVE}.sig" \
    --certificate-identity-regexp "https://github.com/${REPO}" \
    --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
    "${TMP}/${ARCHIVE}"
  echo "Signature verified."
else
  echo "cosign not found — skipping signature verification."
  echo "Install cosign to verify: https://docs.sigstore.dev/cosign/system_config/installation/"
fi

# Extract and install
tar -xzf "${TMP}/${ARCHIVE}" -C "${TMP}"

if [ -w "${INSTALL_DIR}" ]; then
  mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  echo "Installing to ${INSTALL_DIR} requires sudo..."
  sudo mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

chmod +x "${INSTALL_DIR}/${BINARY}"

echo ""
echo "azboard ${VERSION} installed to ${INSTALL_DIR}/${BINARY}"
echo "Run: azboard --version"
