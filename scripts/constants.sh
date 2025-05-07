#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

# Current branch
current_branch_temp=$(git symbolic-ref -q --short HEAD || git describe --tags --always || echo unknown)
# replace / with - to be a docker tag compatible
current_branch=${current_branch_temp////-}

# camino-messenger-bot git tag and sha
git_commit=${CAMINO_BOT_COMMIT:-$(git rev-parse --short HEAD)}
git_tag=${CAMINO_BOT_TAG:-$(git describe --tags --always --dirty || echo unknown)}

# get protocol releases from buf.build
grpc_release=$("${SCRIPT_DIR}"/resolve_protocol_release.sh buf.build/gen/go/chain4travel/camino-messenger-protocol/grpc/go)
protocolbuffers_release=$("${SCRIPT_DIR}"/resolve_protocol_release.sh buf.build/gen/go/chain4travel/camino-messenger-protocol/protocolbuffers/go)

export current_branch
export git_commit
export git_tag
export grpc_release
export protocolbuffers_release