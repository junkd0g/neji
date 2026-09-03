package nerror

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Markdown renders the catalog as a Markdown table — one row per error
// code, sorted, ready to paste into a README or serve from a docs endpoint.
// Because it is generated from the same data the API runs on, it can never
// go stale.
func (c Catalog) Markdown() string {
	var b strings.Builder
	b.WriteString("| Code | HTTP Status | gRPC Code | Retryable | Message |\n")
	b.WriteString("|------|-------------|-----------|-----------|---------|\n")
	for _, code := range c.Codes() {
		e := c.New(code)
		retryable := "no"
		if e.RetryAfter > 0 {
			retryable = fmt.Sprintf("after %s", e.RetryAfter)
		}
		fmt.Fprintf(&b, "| `%s` | %d %s | `%s` | %s | %s |\n",
			code, e.Status, http.StatusText(e.Status), e.GRPCCode(), retryable, e.Message)
	}
	return b.String()
}

// OpenAPI renders the catalog as an OpenAPI 3.1 components fragment: a
// shared Error schema (and the FieldError schema it references) plus one
// reusable response per error code, each with a concrete example. Merge it into your spec or serve it directly:
//
//	spec, _ := Errors.OpenAPI()
//
// Codes are emitted under components.responses keyed by their code, so
// "user_not_found" becomes #/components/responses/user_not_found.
func (c Catalog) OpenAPI() ([]byte, error) {
	responses := make(map[string]any, len(c))
	for _, code := range c.Codes() {
		e := c.New(code)
		example := map[string]any{
			"error": map[string]any{
				"code":    e.Code,
				"status":  e.Status,
				"message": e.Message,
			},
		}
		responses[code] = map[string]any{
			"description": e.Message,
			"content": map[string]any{
				"application/json": map[string]any{
					"schema":  map[string]any{"$ref": "#/components/schemas/Error"},
					"example": example,
				},
			},
		}
	}

	doc := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{
				"Error": map[string]any{
					"type":     "object",
					"required": []string{"error"},
					"properties": map[string]any{
						"error": map[string]any{
							"type":     "object",
							"required": []string{"code", "status", "message"},
							"properties": map[string]any{
								"code": map[string]any{
									"type":        "string",
									"description": "Stable machine-readable error code.",
									"enum":        c.Codes(),
								},
								"status": map[string]any{
									"type":        "integer",
									"description": "HTTP status code.",
								},
								"message": map[string]any{
									"type":        "string",
									"description": "Human-readable description.",
								},
								"details": map[string]any{
									"type":        "object",
									"description": "Optional structured context for this occurrence.",
									"properties": map[string]any{
										"fields": map[string]any{
											"type":        "array",
											"description": "Invalid input fields, when the error is a validation failure.",
											"items":       map[string]any{"$ref": "#/components/schemas/FieldError"},
										},
									},
								},
								"correlation_id": map[string]any{
									"type":        "string",
									"description": "Opaque ID that links this response to server-side logs.",
								},
							},
						},
					},
				},
				"FieldError": map[string]any{
					"type":     "object",
					"required": []string{"field", "message"},
					"properties": map[string]any{
						"field": map[string]any{
							"type":        "string",
							"description": "Name of the invalid input field.",
						},
						"message": map[string]any{
							"type":        "string",
							"description": "What is wrong with the field.",
						},
					},
				},
			},
			"responses": responses,
		},
	}

	return json.MarshalIndent(doc, "", "  ")
}
