#!/usr/bin/env bash

set -euo pipefail

# Directory above this script
cd "$( dirname "${BASH_SOURCE[0]}" )"; cd ..; pwd

# build the Go application by calling build.sh and check if failed
if ! ./scripts/build.sh; then
    exit 1
fi

go test -shuffle=on -race -timeout="${TIMEOUT:-120s}" -coverprofile="coverage.out" -covermode="atomic" $(go list ./...)
