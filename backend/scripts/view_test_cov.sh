#!/usr/bin/env bash
set -euo pipefail

PACKAGE_PATH="${1:-./...}"

go test -coverprofile=coverage.out "$PACKAGE_PATH"
go tool cover -html=coverage.out