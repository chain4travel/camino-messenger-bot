#!/bin/bash
#
# Resolve the label from buf.build for a given dependency uri from the go.mod file
#
# This script tries to resolve the label for the protocol release from buf.build. It
# does not error out if the label is not found. Just returns "UnknownBufLabel".

if [ $# -eq 0 ]; then
	echo "Usage: $0 <buf-dependency-path>"
	echo "Example: $0 buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go"
	exit 0
fi

dep_path=$1
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
go_mod_path="$SCRIPT_DIR/../go.mod"

UNKNOWN_LABEL="UnknownBufLabel"

if [ -z "$go_mod_path" ]; then
	echo "$UNKNOWN_LABEL"
	exit 0
fi

short_hash=$(grep -E "$dep_path" "$go_mod_path" | grep -oP '(\w{12})(?=\.)' || echo "")

if [ -z "$short_hash" ]; then
	echo "$UNKNOWN_LABEL"
	exit 0
fi

label=$(curl -s -f --max-time 10 -X POST "https://buf.build/buf.registry.module.v1beta1.LabelService/ListLabels" \
	-H "Content-Type: application/json" \
	--data '{
		"pageSize": 25,
		"resourceRef": {
			"name": {
				"owner": "chain4travel",
				"module": "camino-messenger-protocol"
			}
		},
		"order": "ORDER_UPDATE_TIME_DESC",
		"archiveFilter": "ARCHIVE_FILTER_UNARCHIVED_ONLY"
	}' | jq -r --arg hash "$short_hash" '
	.labels[] | select(.commitId | startswith($hash)) | .name' | (
	release_label=$(grep "^release-" || true)
	if [ -n "$release_label" ]; then
		echo "$release_label"
	else
		head -n 1
	fi
))

if [ -z "$label" ]; then
	echo "$UNKNOWN_LABEL"
else
	echo "$label"
fi
