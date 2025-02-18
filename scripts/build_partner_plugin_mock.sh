#!/bin/bash

# Set the output binary name
OUTPUT_BINARY="build/pp-mock"

# Set the main source file
MAIN_SOURCE="pp-mock/server.go"

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
		cd "$(dirname "${BASH_SOURCE[0]}")" || exit
		cd .. && pwd
	)
fi
echo "cd $CAMINOBOT_PATH"
cd "$CAMINOBOT_PATH" || exit

# Build the Go application
echo "Building partner-plugin mock..."
if [ "$DEBUG" = true ]; then
	BUILD_CMD="go build -o ${OUTPUT_BINARY} -gcflags \"all=-N -l\" ${MAIN_SOURCE}"
else
	BUILD_CMD="go build -o ${OUTPUT_BINARY} ${MAIN_SOURCE}"
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
