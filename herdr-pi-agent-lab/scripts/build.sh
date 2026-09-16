#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

(cd ../pi-session-schema/go && go generate ./...)

go build -o bin/pi-agent-lab-pane ./cmd/pi-agent-lab-pane
go build -o bin/pi-agent-lab-registry-pane ./cmd/pi-agent-lab-registry-pane
go build -o bin/pi-agent-lab-config-pane ./cmd/pi-agent-lab-config-pane
go build -o bin/pi-agent-lab-menu-pane ./cmd/pi-agent-lab-menu-pane
