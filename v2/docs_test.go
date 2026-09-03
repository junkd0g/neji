package nerror_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMarkdown(t *testing.T) {
	md := testCatalog.Markdown()

	if !strings.HasPrefix(md, "| Code | HTTP Status | gRPC Code | Retryable | Message |") {
		t.Fatalf("missing header:\n%s", md)
	}
	for _, want := range []string{
		"| `user_not_found` | 404 Not Found | `NOT_FOUND` | no | user does not exist |",
		"| `quota_exceeded` | 429 Too Many Requests | `RESOURCE_EXHAUSTED` | after 30s | quota exceeded |",
		"| `invalid_payload` | 422 Unprocessable Entity | `INVALID_ARGUMENT` | no | validation failed |",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("missing row %q in:\n%s", want, md)
		}
	}

	// Rows are sorted by code, so the table is stable across runs.
	if strings.Index(md, "invalid_payload") > strings.Index(md, "user_not_found") {
		t.Fatal("rows should be sorted by code")
	}
}

func TestOpenAPI(t *testing.T) {
	spec, err := testCatalog.OpenAPI()
	if err != nil {
		t.Fatal(err)
	}

	var doc struct {
		Components struct {
			Schemas   map[string]json.RawMessage `json:"schemas"`
			Responses map[string]struct {
				Description string `json:"description"`
				Content     map[string]struct {
					Schema  map[string]string `json:"schema"`
					Example map[string]any    `json:"example"`
				} `json:"content"`
			} `json:"responses"`
		} `json:"components"`
	}
	if err := json.Unmarshal(spec, &doc); err != nil {
		t.Fatal(err)
	}

	if _, ok := doc.Components.Schemas["Error"]; !ok {
		t.Fatal("missing shared Error schema")
	}
	if len(doc.Components.Responses) != 3 {
		t.Fatalf("want 3 responses, got %d", len(doc.Components.Responses))
	}

	resp, ok := doc.Components.Responses["user_not_found"]
	if !ok {
		t.Fatal("missing user_not_found response")
	}
	if resp.Description != "user does not exist" {
		t.Fatalf("description = %q", resp.Description)
	}
	content := resp.Content["application/json"]
	if content.Schema["$ref"] != "#/components/schemas/Error" {
		t.Fatalf("schema ref = %v", content.Schema)
	}
	example, _ := content.Example["error"].(map[string]any)
	if example["code"] != "user_not_found" {
		t.Fatalf("example = %v", content.Example)
	}
}

func TestOpenAPIDescribesCorrelationIDAndFields(t *testing.T) {
	spec, err := testCatalog.OpenAPI()
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Properties map[string]json.RawMessage `json:"properties"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(spec, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc.Components.Schemas["FieldError"]; !ok {
		t.Fatal("missing FieldError schema")
	}
	errProps := doc.Components.Schemas["Error"].Properties["error"].Properties
	if _, ok := errProps["correlation_id"]; !ok {
		t.Fatal("Error schema does not describe correlation_id")
	}
	if !strings.Contains(string(errProps["details"]), `#/components/schemas/FieldError`) {
		t.Fatalf("details schema does not reference FieldError: %s", errProps["details"])
	}
}
