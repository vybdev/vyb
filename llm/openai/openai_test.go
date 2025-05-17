package openai

import (
	"encoding/json"
	"testing"
)

// TestGetResponseSchema loads the JSON schema from the embedded structured_output.json
// and prints its content for inspection.
func TestGetResponseSchema(t *testing.T) {
	schema := GetResponseSchema()
	// Pretty print the schema object using JSON marshalling.
	b, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("Error marshalling schema: %v", err)
	}
	t.Logf("Loaded JSON Schema:\n%s", string(b))
}
