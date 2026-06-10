package nerror_test

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	nerror "github.com/junkd0g/neji/v2"
)

func TestErrorMessage(t *testing.T) {
	err := nerror.New(404, "user_not_found", "user does not exist")
	if got, want := err.Error(), "user_not_found: user does not exist"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}

	wrapped := err.Wrap(errors.New("sql: no rows in result set"))
	want := "user_not_found: user does not exist: sql: no rows in result set"
	if got := wrapped.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("connection refused")
	err := nerror.Unavailable("database unreachable").Wrap(cause)

	if !errors.Is(err, cause) {
		t.Fatal("errors.Is should find the wrapped cause")
	}
}

func TestErrorIsMatchesByCode(t *testing.T) {
	catalog := nerror.Catalog{
		"user_not_found": {Status: 404, Message: "user does not exist"},
	}

	err := catalog.Wrap("user_not_found", errors.New("db says no"))
	if !errors.Is(err, catalog.New("user_not_found")) {
		t.Fatal("errors.Is should match two instances with the same code")
	}
	if errors.Is(err, nerror.NotFound("something else entirely")) {
		t.Fatal("errors.Is should not match a different code")
	}
}

func TestErrorMethodsDoNotMutate(t *testing.T) {
	base := nerror.BadRequest("invalid input")

	withDetail := base.With("field", "email")
	if base.Details != nil {
		t.Fatal("With must not mutate the receiver")
	}
	if withDetail.Details["field"] != "email" {
		t.Fatal("With must set the detail on the copy")
	}

	reworded := base.WithMessage("invalid input on attempt %d", 2)
	if base.Message != "invalid input" {
		t.Fatal("WithMessage must not mutate the receiver")
	}
	if reworded.Message != "invalid input on attempt 2" {
		t.Fatal("WithMessage must format the copy's message")
	}
}

func TestGRPCCodeMapping(t *testing.T) {
	cases := []struct {
		err  *nerror.Error
		want nerror.GRPCCode
	}{
		{nerror.NotFound("x"), nerror.CodeNotFound},
		{nerror.BadRequest("x"), nerror.CodeInvalidArgument},
		{nerror.UnprocessableEntity("x"), nerror.CodeInvalidArgument},
		{nerror.Unauthorized("x"), nerror.CodeUnauthenticated},
		{nerror.Forbidden("x"), nerror.CodePermissionDenied},
		{nerror.Conflict("x"), nerror.CodeAborted},
		{nerror.TooManyRequests("x"), nerror.CodeResourceExhausted},
		{nerror.Internal("x"), nerror.CodeInternal},
		{nerror.Unavailable("x"), nerror.CodeUnavailable},
		{nerror.New(418, "teapot", "x"), nerror.CodeFailedPrecondition},
		{nerror.New(599, "weird", "x"), nerror.CodeInternal},
	}
	for _, tc := range cases {
		if got := tc.err.GRPCCode(); got != tc.want {
			t.Errorf("GRPCCode() for status %d = %s, want %s", tc.err.Status, got, tc.want)
		}
	}

	override := &nerror.Error{Status: 404, Code: "x", Message: "x", GRPC: nerror.CodeDataLoss}
	if got := override.GRPCCode(); got != nerror.CodeDataLoss {
		t.Errorf("explicit GRPC override = %s, want DATA_LOSS", got)
	}
}

func TestSlogLogValue(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	err := nerror.Catalog{
		"quota_exceeded": {Status: 429, Message: "quota exceeded", RetryAfter: time.Minute},
	}.Wrap("quota_exceeded", errors.New("redis counter at limit"))

	logger.Error("request failed", "error", err)

	out := buf.String()
	for _, want := range []string{`"code":"quota_exceeded"`, `"status":429`, `"cause":"redis counter at limit"`} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %s: %s", want, out)
		}
	}
}
