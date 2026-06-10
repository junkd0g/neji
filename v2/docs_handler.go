package nerror

import (
	"encoding/json"
	"net/http"
)

// docsEntry is one catalog error in the JSON listing served by DocsHandler.
type docsEntry struct {
	Code              string `json:"code"`
	Status            int    `json:"status"`
	StatusText        string `json:"status_text"`
	Message           string `json:"message"`
	GRPCCode          string `json:"grpc_code"`
	RetryAfterSeconds int    `json:"retry_after_seconds,omitempty"`
	Type              string `json:"type,omitempty"`
}

// DocsHandler returns an http.Handler that serves the catalog's own
// documentation, generated live from the same data the API runs on:
//
//	mux.Handle("GET /errors", Errors.DocsHandler())
//
// Formats, selected with the "format" query parameter:
//
//	(default)        JSON listing of every error code
//	?format=md       Markdown table (see Catalog.Markdown)
//	?format=openapi  OpenAPI 3.1 components fragment (see Catalog.OpenAPI)
//
// An unknown format yields a catalog-style 400 response.
func (c Catalog) DocsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch format := r.URL.Query().Get("format"); format {
		case "", "json":
			entries := make([]docsEntry, 0, len(c))
			for _, code := range c.Codes() {
				e := c.New(code)
				entries = append(entries, docsEntry{
					Code:              code,
					Status:            e.Status,
					StatusText:        http.StatusText(e.Status),
					Message:           e.Message,
					GRPCCode:          e.GRPCCode().String(),
					RetryAfterSeconds: int(e.RetryAfter.Seconds()),
					Type:              e.Type,
				})
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"errors": entries})

		case "md", "markdown":
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			_, _ = w.Write([]byte(c.Markdown()))

		case "openapi":
			spec, err := c.OpenAPI()
			if err != nil {
				Write(w, Internal("failed to render openapi spec").Wrap(err))
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = w.Write(spec)

		default:
			Write(w, BadRequest("unknown format, want json, md or openapi").
				With("format", format))
		}
	})
}
