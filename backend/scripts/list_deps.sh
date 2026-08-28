#!/usr/bin/env bash
set -euo pipefail

FOLDER="${1:-.}"

cd "$FOLDER"

go list -f '{{join .Imports "\n"}}' ./ | grep "^github.com/NicoClack/cryptic-stash/backend/"
