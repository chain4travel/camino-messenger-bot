#!/bin/bash

# Set the target directory with tilde for home
target_dir="$HOME/tmp/cmb-e2e"

# Get the newest directory based on timestamp
newest_dir=$(ls -d -t "$target_dir"/*/ | head -n 1)

# Remove the trailing slash from the directory name
newest_dir="${newest_dir%/}"

# Print the newest directory for validation
echo "Newest directory: $newest_dir"

# Construct the full path to the log file
log_file="$newest_dir/TestE2E/ActivityV2/pp-mock/partner-plugin-10011.log"

# Check if the log file exists
if [ -f "$log_file" ]; then
  # Cat the log file
  cat "$log_file"
else
  echo "Error: Log file not found at $log_file"
  exit 1
fi