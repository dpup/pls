package llm

import (
	"encoding/json"
	"testing"
)

func TestParseTextResponse_EmptyText(t *testing.T) {
	_, err := parseTextResponse("", "claude-haiku-4-5-20251001")
	if err == nil {
		t.Fatal("expected error for empty text")
	}

	want := "the model returned an empty response — try rephrasing your intent"
	if err.Error() != want {
		t.Errorf("error message:\n got:  %q\n want: %q", err.Error(), want)
	}
}

func TestParseTextResponse_ValidJSON(t *testing.T) {
	input := `{"candidates":[{"cmd":"make test","reason":"runs tests","confidence":0.95,"risk":"safe"}]}`
	resp, err := parseTextResponse(input, "test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(resp.Candidates))
	}
	if resp.Candidates[0].Cmd != "make test" {
		t.Errorf("expected cmd %q, got %q", "make test", resp.Candidates[0].Cmd)
	}
}

func TestCandidatesOutputConfig_ValidSchema(t *testing.T) {
	cfg := candidatesOutputConfig()

	// Verify the schema serializes to valid JSON.
	data, err := json.Marshal(cfg.Format)
	if err != nil {
		t.Fatalf("schema should serialize: %v", err)
	}

	// Verify key fields are present in the serialized output.
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("schema should be valid JSON: %v", err)
	}

	schema, ok := m["schema"].(map[string]any)
	if !ok {
		t.Fatal("expected 'schema' key in output config format")
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected 'properties' in schema")
	}

	if _, ok := props["candidates"]; !ok {
		t.Error("schema should have 'candidates' property")
	}

	if schema["additionalProperties"] != false {
		t.Error("schema should have additionalProperties: false")
	}
}
