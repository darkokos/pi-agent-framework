#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

(cd ../pi-session-schema/go && go generate ./...)

go build -o bin/pi-agent-lab ./cmd/pi-agent-lab

./scripts/install-completions.sh || echo "warning: shell completion install failed; run scripts/install-completions.sh manually" >&2
