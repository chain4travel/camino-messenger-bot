#!/bin/bash
set -euo pipefail

git update-index --really-refresh >> /dev/null
git diff-index --quiet HEAD
