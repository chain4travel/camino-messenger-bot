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

echo "Starting build process..."

if [ -z "${CAMINOBOT_PATH}" ]; then
    # camino-messenger-bot root folder
    CAMINOBOT_PATH=$(
        cd "$(dirname "${BASH_SOURCE[0]}")"
        cd .. && pwd
    )
fi
echo "cd $CAMINOBOT_PATH"
cd "$CAMINOBOT_PATH"

# Load the constants
echo "Preparing constants..."
source "$CAMINOBOT_PATH"/scripts/constants.sh

echo "  DEBUG                  : $DEBUG"
echo "  git_tag                : $git_tag"
echo "  git_commit             : $git_commit"
echo "  protocolbuffers_release: $protocolbuffers_release"
echo "  grpc_release           : $grpc_release"

LDFLAGS="-X github.com/chain4travel/camino-messenger-bot/internal/version.AppGitCommit=$git_commit"
LDFLAGS="$LDFLAGS -X github.com/chain4travel/camino-messenger-bot/internal/version.AppVersion=$git_tag"
LDFLAGS="$LDFLAGS -X github.com/chain4travel/camino-messenger-bot/internal/version.BufBuildPBCMPRelease=$protocolbuffers_release"
LDFLAGS="$LDFLAGS -X github.com/chain4travel/camino-messenger-bot/internal/version.BufBuildGRPCCMPRelease=$grpc_release"

# Build the Go application
echo "Building camino-messenger-bot..."
if [ "$DEBUG" = true ]; then
    BUILD_CMD="go build -o ${OUTPUT_BINARY} -ldflags \"$LDFLAGS\" -gcflags \"all=-N -l\" ${MAIN_SOURCE}"
else
    BUILD_CMD="go build -o ${OUTPUT_BINARY} -ldflags \"$LDFLAGS\" ${MAIN_SOURCE}"
fi

echo "$BUILD_CMD"
eval "$BUILD_CMD"

if [ $? -eq 0 ]; then
    echo "Output binary: ${CAMINOBOT_PATH}/${OUTPUT_BINARY}"
    echo "Build successful!"
else
    echo "Build failed."
    exit 1
fi
