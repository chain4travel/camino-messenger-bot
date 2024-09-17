#!/usr/bin/env bash
set -euo pipefail

# Current branch
current_branch_temp=$(git symbolic-ref -q --short HEAD || git describe --tags --always || echo unknown)
# replace / with - to be a docker tag compatible
current_branch=${current_branch_temp////-}

# camino-messenger-bot and caminoethvm git tag and sha
git_commit=${CAMINO_BOT_COMMIT:-$(git rev-parse --short HEAD)}
git_tag=${CAMINO_BOT_TAG:-$(git describe --tags --abbrev=0 --always || echo unknown)}
