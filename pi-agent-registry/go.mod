module github.com/darkokos/pi-agent-framework/pi-agent-registry

go 1.26.5

replace github.com/darkokos/pi-agent-framework/pi-session-schema => ../pi-session-schema

require (
	github.com/darkokos/pi-agent-framework/pi-session-schema v0.0.0-00010101000000-000000000000
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
)

require golang.org/x/text v0.14.0 // indirect
