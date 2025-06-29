#!/bin/bash

set -e

VERSION="$1"
OUTPUT_DIR="${2:-dist}"

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version> [output_dir]"
    echo "Example: $0 v1.0.0 dist"
    exit 1
fi

echo "Building plugins for version: $VERSION"
echo "Output directory: $OUTPUT_DIR"

mkdir -p "$OUTPUT_DIR"

# Get list of plugins
plugins=$(cd ./external/plugins && ls)

if [ -z "$plugins" ]; then
    echo "No plugins found in ./external/plugins"
    exit 0
fi

# Define target platforms
platforms=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64" "windows/arm64")

for plugin in $plugins; do
    echo "Building plugin: $plugin"
    for platform in "${platforms[@]}"; do
        IFS='/' read -r os arch <<< "$platform"
        echo "Building $plugin for $os/$arch"
        
        binary_name="plugin-$plugin"
        if [ "$os" = "windows" ]; then
            binary_name="$binary_name.exe"
        fi
        
        GOOS=$os GOARCH=$arch go build \
            -tags lambda.norpc \
            -ldflags "-X github.com/imposter-project/imposter-go/internal/version.Version=$VERSION" \
            -o "$OUTPUT_DIR/$binary_name" \
            "./external/plugins/$plugin"
        
        echo "Built: $OUTPUT_DIR/$binary_name"
        
        # Compress the binary
        archive_name="plugin-${plugin}_${os}_${arch}.zip"
        (cd "$OUTPUT_DIR" && zip "$archive_name" "$binary_name")
        rm "$OUTPUT_DIR/$binary_name"
        echo "Compressed: $OUTPUT_DIR/$archive_name"
    done
done

echo "Plugin build complete. Built plugins:"
ls -la "$OUTPUT_DIR"/plugin-*
