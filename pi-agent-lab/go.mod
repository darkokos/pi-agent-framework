module github.com/darkokos/pi-agent-framework/pi-agent-lab

go 1.26.5

require github.com/spf13/cobra v1.10.2

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
)

replace github.com/darkokos/pi-agent-framework/pi-session-schema => ../pi-session-schema

replace github.com/darkokos/pi-agent-framework/pi-agent-compare => ../pi-agent-compare
