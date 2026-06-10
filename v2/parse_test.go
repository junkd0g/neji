package nerror_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	nerror "github.com/junkd0g/neji/v2"
)

// roundTrip serves err through write and parses the recorded response,
// exercising the full wire round-trip.
func roundTrip(t *testing.T, write func(http.ResponseWriter, error), err error) *nerror.Error {
	t.Helper()
	rec := httptest.NewRecorder()
	write(rec, err)

	parsed := nerror.Parse(rec.Result())
	if parsed == nil {
		t.Fatal("Parse returned nil for an error response")
	}
	var e *nerror.Error
	if !errors.As(parsed, &e) {
		t.Fatalf("Parse returned %T, want *nerror.Error", parsed)
	}
	return e
}

func TestParseRoundTripEnvelope(t *testing.T) {
	e := roundTrip(t, nerror.Write, testCatalog.New("quota_exceeded").With("limit", "100/min"))

	if !errors.Is(e, testCatalog.New("quota_exceeded")) {
		t.Fatal("errors.Is must match the catalog code across the wire")
	}
	if e.Status != 429 || e.Message != "quota exceeded" {
		t.Fatalf("unexpected parsed error: %+v", e)
	}
	if e.Details["limit"] != "100/min" {
		t.Fatalf("details = %v", e.Details)
	}
	if e.CorrelationID == "" {
		t.Fatal("correlation ID should survive the round-trip")
	}
	if e.RetryAfter != 30*time.Second {
		t.Fatalf("RetryAfter = %s, want 30s from the Retry-After header", e.RetryAfter)
	}
}

func TestParseRoundTripProblem(t *testing.T) {
	e := roundTrip(t, nerror.WriteProblem, testCatalog.New("invalid_payload"))

	if !errors.Is(e, testCatalog.New("invalid_payload")) {
		t.Fatal("errors.Is must match the catalog code across the wire")
	}
	if e.Status != 422 || e.Message != "validation failed" {
		t.Fatalf("unexpected parsed error: %+v", e)
	}
}

func TestParseNonErrorStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.WriteHeader(http.StatusOK)

	if err := nerror.Parse(rec.Result()); err != nil {
		t.Fatalf("Parse of a 200 should be nil, got %v", err)
	}
}

func TestParseNonJSONBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: 502,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("<html>502 Bad Gateway</html>")),
	}

	err := nerror.Parse(resp)
	var e *nerror.Error
	if !errors.As(err, &e) {
		t.Fatalf("Parse returned %T", err)
	}
	if e.Status != 502 || e.Code != "http_error" || e.Message != "Bad Gateway" {
		t.Fatalf("unexpected fallback error: %+v", e)
	}
}
