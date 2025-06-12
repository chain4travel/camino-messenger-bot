#!/bin/bash

# Set the output binary name
OUTPUT_BINARY="build/camino-messenger-bot"

# Set the main source file
MAIN_SOURCE="main.go"

# Flag to enable debug mode
DEBUG=false

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
	key="$1"
	case $key in
	-d | --debug)
		DEBUG=true
		shift
		;;
	*)
		echo "Unknown option: $1"
		exit 1
		;;
	esac
done

# Changes to the minimum golang version must also be replicated in
# scripts/build.sh (here)
# Dockerfile
# README.md
# go.mod
go_version_minimum="1.23"

go_version() {
    go version | sed -nE -e 's/[^0-9.]+([0-9.]+).+/\1/p'
}

version_lt() {
    # Return true if $1 is a lower version than than $2,
    local ver1=$1
    local ver2=$2
    # Reverse sort the versions, if the 1st item != ver1 then ver1 < ver2
    if  [[ $(echo -e -n "$ver1\n$ver2\n" | sort -rV | head -n1) != "$ver1" ]]; then
        return 0
    else
        return 1
    fi
}

if version_lt "$(go_version)" "$go_version_minimum"; then
    echo "camino-messenger-bot requires Go >= $go_version_minimum, Go $(go_version) found." >&2
    exit 1
fi

echo "Starting build process..."

if [ -z "${CAMINOBOT_PATH}" ]; then
	# camino-messenger-bot root folder
	CAMINOBOT_PATH=$(
		cd "$(dirname "${BASH_SOURCE[0]}")" || exit
		cd .. && pwd
	)
fi
echo "cd $CAMINOBOT_PATH"
cd "$CAMINOBOT_PATH" || exit

# Load the constants
echo "Preparing constants..."
source "$CAMINOBOT_PATH"/scripts/constants.sh

echo "  DEBUG                  : $DEBUG"
echo "  git_tag                : $git_tag"
echo "  git_commit             : $git_commit"
echo "  protocolbuffers_release: $protocolbuffers_release"
echo "  grpc_release           : $grpc_release"

LDFLAGS="-X github.com/chain4travel/camino-messenger-bot/v11/internal/version.AppGitCommit=$git_commit"
LDFLAGS="$LDFLAGS -X github.com/chain4travel/camino-messenger-bot/v11/internal/version.AppVersion=$git_tag"
LDFLAGS="$LDFLAGS -X github.com/chain4travel/camino-messenger-bot/v11/internal/version.BufBuildPBCMPRelease=$protocolbuffers_release"
LDFLAGS="$LDFLAGS -X github.com/chain4travel/camino-messenger-bot/v11/internal/version.BufBuildGRPCCMPRelease=$grpc_release"

# Build the Go application
echo "Building camino-messenger-bot..."
if [ "$DEBUG" = true ]; then
	BUILD_CMD="go build -o ${OUTPUT_BINARY} -ldflags \"$LDFLAGS\" -gcflags \"all=-N -l\" ${MAIN_SOURCE}"
else
	BUILD_CMD="go build -o ${OUTPUT_BINARY} -ldflags \"$LDFLAGS\" ${MAIN_SOURCE}"
fi

echo "$BUILD_CMD"


if eval "$BUILD_CMD"
then
	echo "Output binary: ${CAMINOBOT_PATH}/${OUTPUT_BINARY}"
	echo "Build successful!"
else
	echo "Build failed."
	exit 1
fi
