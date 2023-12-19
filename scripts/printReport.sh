#!/bin/bash

if [ $# -eq 0 ]; then
  echo "Usage: $0 <filename>"
  exit 1
fi

filename=$1

if [ ! -f "$filename" ]; then
  echo "File not found: $filename"
  exit 1
fi

# Read data from the file into an array
mapfile -t data < "$filename"

# Function to calculate the average of an array
calculate_average() {
  local sum=0
  local count=${#data[@]}
  for value in "${data[@]}"; do
    sum=$((sum + value))
  done
  echo "scale=2; $sum / $count" | bc
}

# Sort the array
sorted_data=($(for i in "${data[@]}"; do echo $i; done | sort -n))

# Calculate min, max, median, and average
min=${sorted_data[0]}
max=${sorted_data[-1]}
median=${sorted_data[${#sorted_data[@]}/2]}
average=$(calculate_average)
total=${#data[@]}
# Print the results
echo "Min: $min"
echo "Max: $max"
echo "Median: $median"
echo "Average: $average"
echo "Total: $total"
