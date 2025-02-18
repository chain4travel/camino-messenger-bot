#!/bin/bash

set -e

CAMINOGO_REPO="https://github.com/chain4travel/caminogo"
CONDUIT_REPO="https://github.com/chain4travel/camino-conduit"

default_version="latest"

CAMINOGO_VERSION="$default_version"
CONDUIT_VERSION="$default_version"

while [[ $# -gt 0 ]]; do
    case $1 in
        --caminogo)
            CAMINOGO_VERSION="$2"
            shift 2
            ;;
        --camino-conduit)
            CONDUIT_VERSION="$2"
            shift 2
            ;;
        *)
            echo "Unknown argument: $1"
            exit 1
            ;;
    esac
done

ORIG_DIR=$(pwd)
dependency_dir="build/dependencies"
mkdir -p "$dependency_dir"

download_and_extract() {
    local repo_name=$1
    local version=$2
    local repo_url=$3
    local dest_dir="$dependency_dir/$repo_name"

    # Remove existing directory to ensure fresh download
    if [ -d "$dest_dir" ]; then
        echo "Removing existing $repo_name directory..."
        rm -rf "$dest_dir"
    fi

    mkdir -p "$dest_dir"

    if [ "$version" = "latest" ]; then
        version=$(curl -s "https://api.github.com/repos/chain4travel/$repo_name/releases/latest" | grep -Po '"tag_name": "\K[^"]*')
    fi

    local url="https://github.com/chain4travel/$repo_name/releases/download/$version/${repo_name}-linux-amd64-${version}.tar.gz"

    echo "Downloading $repo_name version $version..."
    if curl --output /dev/null --silent --head --fail "$url"; then
        curl -L "$url" -o "$dest_dir/${repo_name}.tar.gz"
        tar -xzf "$dest_dir/${repo_name}.tar.gz" -C "$dest_dir"
        rm "$dest_dir/${repo_name}.tar.gz"
    else
        echo "Release not found for $repo_name version $version, attempting to clone and build..."
        if git ls-remote --heads --tags "$repo_url" | grep -q "$version"; then
            git clone --depth 1 --branch "$version" "$repo_url" "$dest_dir"
        elif git ls-remote "$repo_url" | grep -q "$version"; then
            git clone --depth 1 "$repo_url" "$dest_dir"
            cd "$dest_dir"
            git checkout "$version"
        else
            echo "Version/tag/commit not found for $repo_name, aborting."
            exit 1
        fi
        
        cd "$dest_dir"
        ./scripts/build.sh        
        cd "$ORIG_DIR"
    fi
}

download_and_extract "caminogo" "$CAMINOGO_VERSION" "$CAMINOGO_REPO"
download_and_extract "camino-conduit" "$CONDUIT_VERSION" "$CONDUIT_REPO"

echo "Building e2e tests..."

go test -tags=e2e -c -o build/tests_e2e

CAMINOGO_BIN_PATH=$dependency_dir/caminogo/caminogo
MATRIX_BIN_PATH=$dependency_dir/camino-conduit/camino-conduit
PARTNER_PLUGIN_BIN_PATH=build/pp-mock
CMB_BIN_PATH=build/camino-messenger-bot
CMB_DB_MIGRATIONS_PATH=migrations

CAMINOGO_BIN_PATH="$(realpath "${CAMINOGO_BIN_PATH}")"
MATRIX_BIN_PATH="$(realpath "${MATRIX_BIN_PATH}")"
PARTNER_PLUGIN_BIN_PATH="$(realpath "${PARTNER_PLUGIN_BIN_PATH}")"
CMB_BIN_PATH="$(realpath "${CMB_BIN_PATH}")"
CMB_DB_MIGRATIONS_PATH="$(realpath "${CMB_DB_MIGRATIONS_PATH}")"

echo "Running e2e tests..."

./e2e.test \
	-test.v \
	-node="${CAMINOGO_BIN_PATH}" \
	-matrix="${MATRIX_BIN_PATH}" \
	-partner-plugin="${PARTNER_PLUGIN_BIN_PATH}" \
	-cmb="${CMB_BIN_PATH}" \
	-migration="${CMB_DB_MIGRATIONS_PATH}"
