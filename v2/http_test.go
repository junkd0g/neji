package nerror_test

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	nerror "github.com/junkd0g/neji/v2"
)

func TestWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	nerror.Write(rec, testCatalog.New("user_not_found").With("user_id", "42"))

	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q", ct)
	}

	var body struct {
		Error struct {
			Code          string         `json:"code"`
			Status        int            `json:"status"`
			Message       string         `json:"message"`
			Details       map[string]any `json:"details"`
			CorrelationID string         `json:"correlation_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	e := body.Error
	if e.Code != "user_not_found" || e.Status != 404 || e.Message != "user does not exist" {
		t.Fatalf("unexpected body: %+v", e)
	}
	if e.Details["user_id"] != "42" {
		t.Fatalf("details = %v", e.Details)
	}
	if len(e.CorrelationID) != 16 {
		t.Fatalf("correlation_id = %q, want 16 hex chars", e.CorrelationID)
	}
}

func TestWriteSetsRetryAfter(t *testing.T) {
	rec := httptest.NewRecorder()
	nerror.Write(rec, testCatalog.New("quota_exceeded"))

	if rec.Code != 429 {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "30" {
		t.Fatalf("Retry-After = %q, want \"30\"", got)
	}
}

func TestWriteDoesNotLeakUnknownErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	nerror.Write(rec, errors.New("pq: password authentication failed for user admin"))

	if rec.Code != 500 {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "password") {
		t.Fatalf("internal error leaked to client: %s", body)
	}
	if !strings.Contains(body, `"code":"internal"`) {
		t.Fatalf("expected generic internal error, got: %s", body)
	}
}

func TestWriteUnwrapsNestedError(t *testing.T) {
	rec := httptest.NewRecorder()
	inner := testCatalog.New("invalid_payload")
	nerror.Write(rec, errors.Join(errors.New("handler failed"), inner))

	if rec.Code != 422 {
		t.Fatalf("status = %d, want 422 from the wrapped *Error", rec.Code)
	}
}

func TestWriteProblem(t *testing.T) {
	rec := httptest.NewRecorder()
	catalog := nerror.Catalog{
		"quota_exceeded": {
			Status:  429,
			Message: "quota exceeded",
			Type:    "https://example.com/errors/quota_exceeded",
		},
	}
	nerror.WriteProblem(rec, catalog.New("quota_exceeded"))

	if rec.Code != 429 {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
		t.Fatalf("Content-Type = %q", ct)
	}

	var p struct {
		Type   string `json:"type"`
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
		Code   string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Type != "https://example.com/errors/quota_exceeded" ||
		p.Title != "Too Many Requests" ||
		p.Status != 429 ||
		p.Detail != "quota exceeded" ||
		p.Code != "quota_exceeded" {
		t.Fatalf("unexpected problem document: %+v", p)
	}
}

func TestWriteProblemDefaultsTypeToAboutBlank(t *testing.T) {
	rec := httptest.NewRecorder()
	nerror.WriteProblem(rec, nerror.NotFound("nope"))

	if !strings.Contains(rec.Body.String(), `"type":"about:blank"`) {
		t.Fatalf("expected about:blank type, got: %s", rec.Body.String())
	}
}

func TestWriteSetsCorrelationIDHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	nerror.Write(rec, testCatalog.New("user_not_found"))

	header := rec.Header().Get("X-Correlation-ID")
	if len(header) != 16 {
		t.Fatalf("X-Correlation-ID = %q, want 16 hex chars", header)
	}
	if !strings.Contains(rec.Body.String(), `"correlation_id":"`+header+`"`) {
		t.Fatalf("header %q does not match body %s", header, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	nerror.WriteProblem(rec, testCatalog.New("user_not_found").WithCorrelationID("req-1"))
	if got := rec.Header().Get("X-Correlation-ID"); got != "req-1" {
		t.Fatalf("WriteProblem X-Correlation-ID = %q", got)
	}

	// The header can be renamed or disabled.
	nerror.CorrelationIDHeader = "X-Trace"
	defer func() { nerror.CorrelationIDHeader = "X-Correlation-ID" }()
	rec = httptest.NewRecorder()
	nerror.Write(rec, testCatalog.New("user_not_found"))
	if rec.Header().Get("X-Trace") == "" || rec.Header().Get("X-Correlation-ID") != "" {
		t.Fatalf("renamed header not honoured: %v", rec.Header())
	}

	nerror.CorrelationIDHeader = ""
	rec = httptest.NewRecorder()
	nerror.Write(rec, testCatalog.New("user_not_found"))
	if len(rec.Header()) != 1 { // Content-Type only
		t.Fatalf("disabled header still set: %v", rec.Header())
	}
}
