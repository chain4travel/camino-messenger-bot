#!/bin/bash
set -euo pipefail

git update-index --really-refresh
git diff-index --name-status HEAD
