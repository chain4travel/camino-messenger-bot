#!/bin/bash

set -e

CAMINOGO_REPO="https://github.com/chain4travel/caminogo"
CONDUIT_REPO="https://github.com/chain4travel/camino-conduit"

default_version="latest"

CAMINOGO_VERSION="$default_version"
CONDUIT_VERSION="$default_version"

FALLBACK_BRANCH="dev"
BUILD_SCRIPT="./scripts/build.sh"

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

	echo "Attemting to download $repo_name"

    # Remove existing directory to ensure fresh download
    if [ -d "$dest_dir" ]; then
        echo "Removing existing $repo_name directory..."
        rm -rf "$dest_dir"
    fi

    mkdir -p "$dest_dir"
    release_version=""
    if [ "$version" = "latest" ]; then
        release_version=$(curl -s "https://api.github.com/repos/chain4travel/$repo_name/releases/latest" | grep -Po '"tag_name": "\K[^"]*' || echo "")
    fi

	if [ -z "$release_version" ] ; then
		if [ "$version" = "latest" ]; then
			branch=$FALLBACK_BRANCH
		else
			branch=$version
		fi

		echo "ERROR: Unable to get the released version of $repo_name! Fallback to clone and build of the branch '$branch'."

		if git ls-remote --heads --tags "$repo_url" | grep -q "$branch"; then
            git clone --depth 1 --branch "$branch" "$repo_url" "$dest_dir"
        elif git ls-remote "$repo_url" | grep -q "$branch"; then
            git clone --depth 1 "$repo_url" "$dest_dir"
            cd "$dest_dir"
            git checkout "$branch"
        else
            echo "Version/tag/commit '$branch' not found for $repo_name, aborting."
            exit 1
        fi
        
        cd "${ORIG_DIR}/$dest_dir"
		if [ ! -f $BUILD_SCRIPT ] ; then
			echo "CRIT: No build script found at '$BUILD_SCRIPT' in cloned repository. Abort."
			exit 1
		fi
		$BUILD_SCRIPT
        cd "$ORIG_DIR"
	else
	    local url="https://github.com/chain4travel/$repo_name/releases/download/$release_version/${repo_name}-linux-amd64-${release_version}.tar.gz"

	    echo "Downloading $repo_name version $release_version..."
	    if curl --output /dev/null --silent --head --fail "$url"; then
    	    curl -s -L "$url" -o "$dest_dir/${repo_name}.tar.gz"
	        tar -xzf "$dest_dir/${repo_name}.tar.gz" -C "$dest_dir"
	        rm "$dest_dir/${repo_name}.tar.gz"
    	else
			echo "CRIT: Unable to download the release '$release_version' of $repo_name."
			exit 1
		fi
	fi
}

download_and_extract "caminogo" "$CAMINOGO_VERSION" "$CAMINOGO_REPO"
download_and_extract "camino-conduit" "$CONDUIT_VERSION" "$CONDUIT_REPO"

echo "Building e2e tests..."


E2E_BIN_OUT=build/tests_e2e

ORIG_DIR=$(pwd)
cd tests/e2e
go test -tags=e2e -c -o ../../$E2E_BIN_OUT e2e_test.go
cd "$ORIG_DIR"

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

./$E2E_BIN_OUT \
	-test.v \
	-node="${CAMINOGO_BIN_PATH}" \
	-matrix="${MATRIX_BIN_PATH}" \
	-partner-plugin="${PARTNER_PLUGIN_BIN_PATH}" \
	-cmb="${CMB_BIN_PATH}" \
	-migration="${CMB_DB_MIGRATIONS_PATH}"
