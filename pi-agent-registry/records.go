package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"

	sessionrecord "github.com/darkokos/pi-agent-framework/pi-session-schema/go"
)

func recordsDir() (string, error) {
	if v := os.Getenv("PI_SESSION_RECORDS"); v != "" {
		return v, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".pi", "agent", "records"), nil
}

func recordSchema() (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(sessionrecord.Schema)))
	if err != nil {
		return nil, err
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource("session-record.schema.json", doc); err != nil {
		return nil, err
	}

	return c.Compile("session-record.schema.json")
}

func ListRecords() ([]*sessionrecord.SessionRecord, error) {
	dir, err := recordsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	schema, err := recordSchema()
	if err != nil {
		return nil, err
	}

	var records []*sessionrecord.SessionRecord
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		inst, err := jsonschema.UnmarshalJSON(strings.NewReader(string(data)))
		if err != nil {
			continue
		}
		if err := schema.Validate(inst); err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: %v\n", path, err)
			continue
		}

		var record sessionrecord.SessionRecord
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}

		records = append(records, &record)
	}

	return records, nil
}
