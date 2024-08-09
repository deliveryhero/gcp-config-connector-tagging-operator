#!/usr/bin/env bash
set -euo pipefail

if [ -n "$(git status --porcelain)" ]; then
  echo "Working directory is not clean. Please run make generate helm, stash your changes or remove untracked files."
  exit 1
fi
