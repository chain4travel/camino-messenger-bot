#!/bin/bash

set -euo pipefail

# Function to check if a command exists
golangci_lint_installed() {
   golangci-lint --version >/dev/null 2>&1
}

# Function to install golangci-lint on Ubuntu
install_golangci_lint() {
    echo "Installing golangci-lint..."
    go install -v github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60
}

# Check if golangci-lint is installed
if golangci_lint_installed ; then
    echo "golangci-lint is already installed."
else
    echo "golangci-lint is not installed."
    install_golangci_lint
fi

# Run golangci-lint
echo "Running golangci-lint..."
golangci-lint run --config .golangci.yml
