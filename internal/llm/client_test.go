package llm

import "testing"

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

func TestParseTextResponse_MarkdownFenced(t *testing.T) {
	input := "```json\n{\"candidates\":[{\"cmd\":\"go test ./...\",\"reason\":\"test\",\"confidence\":0.9,\"risk\":\"safe\"}]}\n```"
	resp, err := parseTextResponse(input, "test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(resp.Candidates))
	}
	if resp.Candidates[0].Cmd != "go test ./..." {
		t.Errorf("expected cmd %q, got %q", "go test ./...", resp.Candidates[0].Cmd)
	}
}
