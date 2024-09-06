#!/usr/bin/env bash

# Ignore warnings about variables appearing unused since this file is not the consumer of the variables it defines.
# shellcheck disable=SC2034

set -euo pipefail

# Use lower_case variables in the scripts and UPPER_CASE variables for override
# Use the constants.sh for env overrides

CAMINOBOT_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd .. && pwd )

# Where CaminoGo binary goes
build_dir="$CAMINOBOT_PATH/build"
CAMINOBOT_BIN_PATH="$build_dir/camino-messenger-bot"


# Camino docker hub
# c4tplatform/camino-messenger-bot - defaults to local as to avoid unintentional pushes
# You should probably set it - export DOCKER_REPO='c4tplatform'
camino_bot_dockerhub_repo=${DOCKER_REPO:-"c4tplatform"}"/camino-messenger-bot"

# Current branch
current_branch_temp=$(git symbolic-ref -q --short HEAD || git describe --tags --always || echo unknown)
# replace / with - to be a docker tag compatible
current_branch=${current_branch_temp////-}

# camino-messenger-bot and caminoethvm git tag and sha
git_commit=${CAMINO_BOT_COMMIT:-$(git rev-parse --short HEAD)}
git_tag=${CAMINO_BOT_TAG:-$(git describe --tags --abbrev=0 --always || echo unknown)}

# Static compilation
static_ld_flags=''
if [ "${STATIC_COMPILATION:-}" = 1 ]
then
    export CC=musl-gcc
    which $CC > /dev/null || ( echo $CC must be available for static compilation && exit 1 )
    static_ld_flags=' -extldflags "-static" -linkmode external '
fi
