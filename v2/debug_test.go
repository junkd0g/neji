package nerror_test

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	nerror "github.com/junkd0g/neji/v2"
)

func TestDebugModeExposesCauseAndStack(t *testing.T) {
	nerror.Debug = true
	defer func() { nerror.Debug = false }()

	rec := httptest.NewRecorder()
	nerror.Write(rec, testCatalog.Wrap("user_not_found", errors.New("sql: no rows in result set")))

	var body struct {
		Error struct {
			Code  string   `json:"code"`
			Cause string   `json:"cause"`
			Stack []string `json:"stack"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	if body.Error.Cause != "sql: no rows in result set" {
		t.Fatalf("cause = %q", body.Error.Cause)
	}
	if len(body.Error.Stack) == 0 {
		t.Fatal("debug response should include a stack trace")
	}
	// The first frame is where the error was created: this test function.
	if !strings.Contains(body.Error.Stack[0], "TestDebugModeExposesCauseAndStack") {
		t.Fatalf("stack should start at the creation site, got %q", body.Error.Stack[0])
	}
}

func TestDebugModeProblemExposesCauseAndStack(t *testing.T) {
	nerror.Debug = true
	defer func() { nerror.Debug = false }()

	rec := httptest.NewRecorder()
	nerror.WriteProblem(rec, testCatalog.Wrap("invalid_payload", errors.New("field email is empty")))

	body := rec.Body.String()
	if !strings.Contains(body, `"cause":"field email is empty"`) {
		t.Fatalf("missing cause: %s", body)
	}
	if !strings.Contains(body, `"stack":[`) {
		t.Fatalf("missing stack: %s", body)
	}
}

func TestProdModeOmitsInternalsButKeepsCorrelationID(t *testing.T) {
	rec := httptest.NewRecorder()
	nerror.Write(rec, testCatalog.Wrap("user_not_found", errors.New("sql: no rows in result set")))

	body := rec.Body.String()
	if strings.Contains(body, "sql:") || strings.Contains(body, `"stack"`) {
		t.Fatalf("prod response leaked internals: %s", body)
	}
	if !strings.Contains(body, `"correlation_id":"`) {
		t.Fatalf("prod response missing correlation ID: %s", body)
	}
}

func TestWithCorrelationIDIsKept(t *testing.T) {
	rec := httptest.NewRecorder()
	nerror.Write(rec, testCatalog.New("user_not_found").WithCorrelationID("req-abc-123"))

	if !strings.Contains(rec.Body.String(), `"correlation_id":"req-abc-123"`) {
		t.Fatalf("explicit correlation ID not kept: %s", rec.Body.String())
	}
}
