#!/bin/bash

set -euo pipefail

if [ -z "${CAMINOBOT_PATH:-}" ]; then
	# camino-messenger-bot root folder
	CAMINOBOT_PATH=$(
		cd "$(dirname "${BASH_SOURCE[0]}")" || exit
		cd .. && pwd
	)
fi

echo "Formatting Go files in ${CAMINOBOT_PATH}..."
gofmt -w "${CAMINOBOT_PATH}"

echo "Formatting complete!"
