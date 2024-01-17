#!/bin/bash

# Check if the number of arguments provided is correct
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <number_of_requests>"
    exit 1
fi

times_to_run=$1
go_file_path="examples/rpc/client.go"

# Loop to run the Go file X times in parallel
for ((i=1; i<=$times_to_run; i++))
do
    echo "Sending $i request..."
    go run $go_file_path &
done

# Wait for all background processes to finish
wait