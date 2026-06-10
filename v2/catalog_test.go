package nerror_test

import (
	"strings"
	"testing"
	"time"

	nerror "github.com/junkd0g/neji/v2"
)

var testCatalog = nerror.Catalog{
	"user_not_found":  {Status: 404, Message: "user does not exist"},
	"quota_exceeded":  {Status: 429, Message: "quota exceeded", RetryAfter: 30 * time.Second},
	"invalid_payload": {Status: 422, Message: "validation failed"},
}

func TestCatalogNew(t *testing.T) {
	err := testCatalog.New("user_not_found")
	if err.Code != "user_not_found" || err.Status != 404 || err.Message != "user does not exist" {
		t.Fatalf("unexpected error: %+v", err)
	}

	if a, b := testCatalog.New("user_not_found"), testCatalog.New("user_not_found"); a == b {
		t.Fatal("New must return a fresh value on each call")
	}
}

func TestCatalogNewf(t *testing.T) {
	err := testCatalog.Newf("user_not_found", "user %q does not exist", "jordan")
	if err.Message != `user "jordan" does not exist` {
		t.Fatalf("Newf message = %q", err.Message)
	}
	if err.Code != "user_not_found" || err.Status != 404 {
		t.Fatal("Newf must keep the declared code and status")
	}
}

func TestCatalogUnknownCode(t *testing.T) {
	err := testCatalog.New("no_such_code")
	if err.Status != 500 || err.Code != "unknown_error" {
		t.Fatalf("unknown code should yield a 500 unknown_error, got %+v", err)
	}
	if !strings.Contains(err.Message, "no_such_code") {
		t.Fatalf("message should name the missing code: %q", err.Message)
	}
}

func TestCatalogCodesSorted(t *testing.T) {
	codes := testCatalog.Codes()
	want := []string{"invalid_payload", "quota_exceeded", "user_not_found"}
	if len(codes) != len(want) {
		t.Fatalf("Codes() = %v, want %v", codes, want)
	}
	for i := range want {
		if codes[i] != want[i] {
			t.Fatalf("Codes() = %v, want %v", codes, want)
		}
	}
}

func TestCatalogValidate(t *testing.T) {
	if err := testCatalog.Validate(); err != nil {
		t.Fatalf("valid catalog should pass: %v", err)
	}

	bad := []nerror.Catalog{
		{"no_message": {Status: 404}},
		{"ok_status": {Status: 200, Message: "not an error"}},
		{"": {Status: 404, Message: "empty code"}},
	}
	for _, c := range bad {
		if err := c.Validate(); err == nil {
			t.Errorf("catalog %+v should fail validation", c)
		}
	}
}
