#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

if ! command -v herdr >/dev/null 2>&1; then
	echo "herdr not found on PATH; install Herdr first (https://herdr.dev)" >&2
	exit 1
fi

./scripts/build.sh

herdr plugin link .
