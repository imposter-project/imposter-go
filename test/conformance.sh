#!/usr/bin/env bash

set -e

IMPOSTER_PROJECT_DIR=~/projects/imposter/imposter-engine

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
IMPOSTER_GO_DIR=$( cd $SCRIPT_DIR/.. && pwd )

if [ ! -d "$IMPOSTER_PROJECT_DIR" ]; then
    echo "Imposter project dir does not exist"
    exit 1
fi

cd $IMPOSTER_GO_DIR
make build

cd $IMPOSTER_PROJECT_DIR
export TEST_ENGINE=go
export IMPOSTER_GO_PATH="$IMPOSTER_GO_DIR/imposter-go"
./gradlew clean test -PonlyVerticleTests
