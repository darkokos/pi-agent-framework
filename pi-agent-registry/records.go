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

func RecordsDir(root string) string {
	return filepath.Join(root, "records")
}

func recordFile(root, sessionID string) string {
	return filepath.Join(RecordsDir(root), sessionID+".json")
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

func parseRecord(schema *jsonschema.Schema, data []byte) (*sessionrecord.SessionRecord, error) {
	inst, err := jsonschema.UnmarshalJSON(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	if err := schema.Validate(inst); err != nil {
		return nil, err
	}

	var record sessionrecord.SessionRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}

	return &record, nil
}

func GetRecord(root, sessionID string) (*sessionrecord.SessionRecord, error) {
	path := recordFile(root, sessionID)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("record %q not found: %w", sessionID, err)
	}

	schema, err := recordSchema()
	if err != nil {
		return nil, err
	}

	record, err := parseRecord(schema, data)
	if err != nil {
		return nil, fmt.Errorf("record %q failed schema validation: %w", sessionID, err)
	}

	return record, nil
}

func ListRecords(root string) ([]*sessionrecord.SessionRecord, error) {
	dir := RecordsDir(root)

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

		record, err := parseRecord(schema, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: %v\n", path, err)
			continue
		}

		records = append(records, record)
	}

	return records, nil
}
