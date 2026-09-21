module github.com/darkokos/pi-agent-framework/pi-agent-invoke

go 1.26.5

require github.com/darkokos/pi-agent-framework/pi-agent-registry v0.0.0-00010101000000-000000000000

require (
	github.com/darkokos/pi-agent-framework/pi-session-schema v0.0.0-00010101000000-000000000000 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/darkokos/pi-agent-framework/pi-agent-registry => ../pi-agent-registry

replace github.com/darkokos/pi-agent-framework/pi-session-schema => ../pi-session-schema
