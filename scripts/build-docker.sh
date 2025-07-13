#!/bin/bash

# Script to build Docker image for local testing
# Usage: ./scripts/build-docker.sh [tag] [platform]

set -e

# Default values
TAG=${1:-"imposter-go:latest"}
PLATFORM=${2:-"linux/amd64,linux/arm64"}
VERSION=$(git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo "dev")

echo "Building Docker image..."
echo "Tag: ${TAG}"
echo "Platform: ${PLATFORM}"
echo "Version: ${VERSION}"

# Build multi-platform image
docker buildx build \
    --platform "${PLATFORM}" \
    --build-arg VERSION="${VERSION}" \
    --tag "${TAG}" \
    --load \
    .

echo "✅ Docker image built successfully!"
echo "Run with: docker run --rm -p 8080:8080 -v \$(pwd)/examples/rest/simple:/opt/imposter/config ${TAG}"
