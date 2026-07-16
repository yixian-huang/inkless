#!/usr/bin/env bash
# Backend production build script
# Produces versioned backend binary with build metadata

set -euo pipefail

# Script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BACKEND_DIR="${PROJECT_ROOT}/backend"
cd "${BACKEND_DIR}"

# Configuration
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')")}"
BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
GIT_COMMIT="$(git rev-parse HEAD 2>/dev/null || echo 'unknown')"
GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo 'unknown')"
GO_VERSION="$(go version | awk '{print $3}')"
ARTIFACTS_DIR="${PROJECT_ROOT}/artifacts"
BINARY_NAME="blotting-api-${VERSION}"
ARTIFACT_NAME="backend-${VERSION}.tar.gz"

echo "=========================================="
echo "Backend Production Build"
echo "=========================================="
echo "Version: ${VERSION}"
echo "Build Time: ${BUILD_TIME}"
echo "Git Commit: ${GIT_COMMIT}"
echo "Git Branch: ${GIT_BRANCH}"
echo "Go Version: ${GO_VERSION}"
echo "Artifact Output: ${ARTIFACTS_DIR}/${ARTIFACT_NAME}"
echo "=========================================="

# Create artifacts directory (at project root)
mkdir -p "${PROJECT_ROOT}/artifacts"

# Verify dependencies
echo "Verifying dependencies..."
go mod verify
go mod tidy

# Run tests
echo "Running tests..."
go test -v -race ./...

# Run go vet
echo "Running go vet..."
go vet ./...

# Build binary with ldflags
echo "Building binary..."
go build \
  -ldflags="-s -w \
    -X 'main.Version=${VERSION}' \
    -X 'main.BuildTime=${BUILD_TIME}' \
    -X 'main.GitCommit=${GIT_COMMIT}' \
    -X 'main.GitBranch=${GIT_BRANCH}'" \
  -o "${ARTIFACTS_DIR}/${BINARY_NAME}" \
  ./cmd/server

# Make binary executable
chmod +x "${ARTIFACTS_DIR}/${BINARY_NAME}"

# Create build metadata
echo "Creating build metadata..."
cat > "${ARTIFACTS_DIR}/build-info.json" <<EOF
{
  "version": "${VERSION}",
  "buildTime": "${BUILD_TIME}",
  "gitCommit": "${GIT_COMMIT}",
  "gitBranch": "${GIT_BRANCH}",
  "goVersion": "${GO_VERSION}"
}
EOF

# Create tarball artifact with binary and metadata
echo "Creating artifact tarball..."
cd "${ARTIFACTS_DIR}"
tar -czf "${ARTIFACT_NAME}" "${BINARY_NAME}" build-info.json
cd "${PROJECT_ROOT}"

# Calculate checksum
echo "Calculating checksum..."
if command -v sha256sum &> /dev/null; then
  sha256sum "${ARTIFACTS_DIR}/${ARTIFACT_NAME}" > "${ARTIFACTS_DIR}/${ARTIFACT_NAME}.sha256"
elif command -v shasum &> /dev/null; then
  shasum -a 256 "${ARTIFACTS_DIR}/${ARTIFACT_NAME}" > "${ARTIFACTS_DIR}/${ARTIFACT_NAME}.sha256"
else
  echo "Warning: sha256sum or shasum not found, skipping checksum generation"
fi

# Create symlink to latest
cd "${PROJECT_ROOT}/artifacts"
ln -sf "${BINARY_NAME}" blotting-api-latest
ln -sf "${ARTIFACT_NAME}" backend-latest.tar.gz
if [ -f "${ARTIFACT_NAME}.sha256" ]; then
  ln -sf "${ARTIFACT_NAME}.sha256" backend-latest.tar.gz.sha256
fi
cd "${PROJECT_ROOT}"

echo "=========================================="
echo "Build completed successfully!"
echo "Binary: ${ARTIFACTS_DIR}/${BINARY_NAME}"
echo "Artifact: ${ARTIFACTS_DIR}/${ARTIFACT_NAME}"
echo "Size: $(du -h "${ARTIFACTS_DIR}/${ARTIFACT_NAME}" | cut -f1)"
if [ -f "${ARTIFACTS_DIR}/${ARTIFACT_NAME}.sha256" ]; then
  echo "Checksum: $(cat "${ARTIFACTS_DIR}/${ARTIFACT_NAME}.sha256" | cut -d' ' -f1)"
fi
echo "=========================================="
