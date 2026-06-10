package nerror_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDocsHandlerJSON(t *testing.T) {
	for _, target := range []string{"/errors", "/errors?format=json"} {
		rec := httptest.NewRecorder()
		testCatalog.DocsHandler().ServeHTTP(rec, httptest.NewRequest("GET", target, nil))

		if rec.Code != 200 {
			t.Fatalf("%s: status = %d", target, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Fatalf("%s: Content-Type = %q", target, ct)
		}

		var doc struct {
			Errors []struct {
				Code              string `json:"code"`
				Status            int    `json:"status"`
				StatusText        string `json:"status_text"`
				Message           string `json:"message"`
				GRPCCode          string `json:"grpc_code"`
				RetryAfterSeconds int    `json:"retry_after_seconds"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		if len(doc.Errors) != 3 {
			t.Fatalf("want 3 entries, got %d", len(doc.Errors))
		}

		// Codes() is sorted, so the listing order is stable.
		quota := doc.Errors[1]
		if quota.Code != "quota_exceeded" ||
			quota.Status != 429 ||
			quota.StatusText != "Too Many Requests" ||
			quota.Message != "quota exceeded" ||
			quota.GRPCCode != "RESOURCE_EXHAUSTED" ||
			quota.RetryAfterSeconds != 30 {
			t.Fatalf("unexpected entry: %+v", quota)
		}
	}
}

func TestDocsHandlerMarkdown(t *testing.T) {
	for _, format := range []string{"md", "markdown"} {
		rec := httptest.NewRecorder()
		testCatalog.DocsHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/errors?format="+format, nil))

		if rec.Code != 200 {
			t.Fatalf("format=%s: status = %d", format, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/markdown") {
			t.Fatalf("format=%s: Content-Type = %q", format, ct)
		}
		if got, want := rec.Body.String(), testCatalog.Markdown(); got != want {
			t.Fatalf("format=%s: body does not match Catalog.Markdown()", format)
		}
	}
}

func TestDocsHandlerOpenAPI(t *testing.T) {
	rec := httptest.NewRecorder()
	testCatalog.DocsHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/errors?format=openapi", nil))

	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}

	want, err := testCatalog.OpenAPI()
	if err != nil {
		t.Fatal(err)
	}
	if rec.Body.String() != string(want) {
		t.Fatal("body does not match Catalog.OpenAPI()")
	}
}

func TestDocsHandlerUnknownFormat(t *testing.T) {
	rec := httptest.NewRecorder()
	testCatalog.DocsHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/errors?format=yaml", nil))

	if rec.Code != 400 {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"code":"bad_request"`) || !strings.Contains(body, `"format":"yaml"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}
