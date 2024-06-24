#!/bin/bash

# Set the output binary name
OUTPUT_BINARY_SUPPLIER="build/supplier/bot"
OUTPUT_BINARY_DISTRIBUTOR="build/distributor/bot"

# Set the main source file
MAIN_SOURCE="cmd/camino-messenger-bot/main.go"

# Flag to enable debug mode
DEBUG=false

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        -d|--debug)
            DEBUG=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Build the Go application
echo "Building camino-messenger-bot supplier..."
if [ "$DEBUG" = true ]; then
    go build -o ${OUTPUT_BINARY_SUPPLIER} -gcflags "all=-N -l" ${MAIN_SOURCE}
else
    go build -o ${OUTPUT_BINARY_SUPPLIER} ${MAIN_SOURCE}
fi



echo "Building camino-messenger-bot distributor..."
if [ "$DEBUG" = true ]; then
    go build -o ${OUTPUT_BINARY_DISTRIBUTOR} -gcflags "all=-N -l" ${MAIN_SOURCE}
else
    go build -o ${OUTPUT_BINARY_DISTRIBUTOR} ${MAIN_SOURCE}
fi



if [ $? -eq 0 ]; then
    echo "Build successful!"
else
    echo "Build failed."
    exit 1
fi
