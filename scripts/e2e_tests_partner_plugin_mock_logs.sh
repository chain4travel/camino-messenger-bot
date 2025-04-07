#!/bin/bash

if [ -d "$HOME/tmp/cmb-e2e" ] ; then
	target_dir="$HOME/tmp/cmb-e2e"
elif [ -d "/tmp/cmb-e2e" ] ; then
	target_dir="/tmp/cmb-e2e"
else
	echo "Error: No tmp dir found where the logs could be"
	exit 1
fi

newest_dir=$(find "$target_dir" -maxdepth 1 -type d -exec stat --format="%Y %n" {} + | sort -n | awk '{print $2}' | tail -n1)
newest_dir="${newest_dir%/}"
echo "Newest directory: $newest_dir"

# The file always is in the same place but may have different names as the
# used port may be different
# it's always: partner-plugin-<PORT>.log

log_files=$(find "$newest_dir" -type f -name "partner-plugin-*.log")

for file in $log_files; do
	echo "----------------------------------------"
	echo "Log file found: $file"
	echo "----------------------------------------"
	cat "$file"
	echo
done