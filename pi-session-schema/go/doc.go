package sessionrecord

//go:generate cp ../schema/session-record.schema.json ./session-record.schema.json
//go:generate go run github.com/atombender/go-jsonschema@v0.24.1 -p sessionrecord -t -o session_record.go ../schema/session-record.schema.json
