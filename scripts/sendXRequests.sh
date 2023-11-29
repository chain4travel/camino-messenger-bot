#!/bin/bash

if ! [[ "$0" =~ scripts/sendXRequests.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

# Check if the number of arguments provided is correct
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <number_of_requests>"
    exit 1
fi

# Store the argument in a variable
times_to_run=$1

# Change the path to your Go file below
go_file_path="examples/rpc/client.go"

# Loop to run the Go file X times in parallel
for ((i=1; i<=$times_to_run; i++))
do
#    echo "Sending $i request..."
    go run $go_file_path &
done

# Wait for all background processes to finish
wait