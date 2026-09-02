// Package sessionrecord holds the generated SessionRecord/Rating structs.
//
// This file is never hand-edited beyond this directive. session_record.go is
// regenerated on every build (see scripts/build.sh in pi-agent-lab and
// herdr-pi-agent-lab) and is gitignored — there is no committed copy that
// could drift from ../schema/session-record.schema.json.
package sessionrecord

//go:generate go run github.com/atombender/go-jsonschema@v0.24.1 -p sessionrecord -t -o session_record.go ../schema/session-record.schema.json
