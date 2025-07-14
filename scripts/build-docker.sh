#!/bin/bash

# Script to build Docker image for local testing
# Usage: ./scripts/build-docker.sh [distro] [tag] [platform]

set -e

# Default values
DISTRO=${1:-"core"}
PLATFORM=${3:-"linux/amd64,linux/arm64"}
VERSION=$(git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo "dev")

# Set default tag based on distro
if [ -z "$2" ]; then
    if [ "$DISTRO" = "all" ]; then
        TAG="outofcoffee/imposter-all:5-beta"
    else
        TAG="outofcoffee/imposter:5-beta"
    fi
else
    TAG="$2"
fi

# Validate distro
if [[ ! -d "distro/${DISTRO}" ]]; then
    echo "Error: Distro '${DISTRO}' not found in distro/ directory"
    echo "Available distros:"
    ls -1 distro/ 2>/dev/null || echo "  (none found)"
    exit 1
fi

echo "Building Docker image..."
echo "Distro: ${DISTRO}"
echo "Tag: ${TAG}"
echo "Platform: ${PLATFORM}"
echo "Version: ${VERSION}"

# Build multi-platform image
docker buildx build \
    --platform "${PLATFORM}" \
    --build-arg VERSION="${VERSION}" \
    --tag "${TAG}" \
    --file "distro/${DISTRO}/Dockerfile" \
    --load \
    .

echo "✅ Docker image built successfully!"
echo "Run with: docker run --rm -p 8080:8080 -v \$(pwd)/examples/rest/simple:/opt/imposter/config ${TAG}"
