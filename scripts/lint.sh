#!/bin/bash

set -euo pipefail

EXPECTED_VERSION="1.60"

# Function to check if a command exists
golangci_lint_installed() {
    golangci-lint --version | grep $EXPECTED_VERSION > /dev/null 2>&1
	if [ $? -eq 0 ]; then
		return 0
	else
		return 1
	fi
}

# Function to install golangci-lint on Ubuntu
# When the golangci-lint version is updated here, also update it in .github/workflows/ci.yml
install_golangci_lint() {
    echo "Installing golangci-lint..."
    go install -v github.com/golangci/golangci-lint/cmd/golangci-lint@v$EXPECTED_VERSION
}

# Check if golangci-lint is installed
if golangci_lint_installed; then
    echo "golangci-lint is already installed."
else
    echo "golangci-lint is not installed (with the right version)."
    install_golangci_lint
fi

# Run golangci-lint
echo "Running golangci-lint..."
golangci-lint run --config .golangci.yml
