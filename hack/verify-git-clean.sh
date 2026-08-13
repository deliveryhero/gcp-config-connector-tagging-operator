#!/usr/bin/env bash
set -euo pipefail
if [ -n "$(git status --porcelain)" ]; then
  echo "Working directory is not clean. Please run make generate or stash your changes."
  exit 1
fi
