#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

(cd ../pi-session-schema/go && go generate ./...)

go build -o bin/pi-agent-lab-pane ./cmd/pi-agent-lab-pane
