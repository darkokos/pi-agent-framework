module github.com/darkokos/pi-agent-framework/pi-agent-lab

go 1.26.5

require github.com/spf13/cobra v1.10.2

require (
	github.com/darkokos/pi-agent-framework/pi-agent-compare v0.0.0-00010101000000-000000000000 // indirect
	github.com/darkokos/pi-agent-framework/pi-session-schema v0.0.0-00010101000000-000000000000 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/darkokos/pi-agent-framework/pi-session-schema => ../pi-session-schema

replace github.com/darkokos/pi-agent-framework/pi-agent-compare => ../pi-agent-compare
