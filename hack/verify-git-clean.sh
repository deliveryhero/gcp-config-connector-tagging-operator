#!/usr/bin/env bash
set -euo pipefail
go mod tidy
if [ -n "$(git status --porcelain)" ]; then
  echo "Working directory is not clean. Please run make generate helm or run go mod tidy, stash your changes or remove untracked files."
  exit 1
fi
