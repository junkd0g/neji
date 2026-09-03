package nerror_test

import (
	"errors"
	"fmt"
	"testing"

	nerror "github.com/junkd0g/neji/v2"
)

func TestCodeAndStatus(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		code   string
		status int
	}{
		{"nil", nil, "", 0},
		{"catalog error", testCatalog.New("user_not_found"), "user_not_found", 404},
		{"wrapped with fmt", fmt.Errorf("lookup: %w", testCatalog.New("quota_exceeded")), "quota_exceeded", 429},
		{"joined", errors.Join(errors.New("other"), testCatalog.New("invalid_payload")), "invalid_payload", 422},
		{"plain error mirrors Write", errors.New("pq: connection refused"), "internal", 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := nerror.Code(tc.err); got != tc.code {
				t.Fatalf("Code = %q, want %q", got, tc.code)
			}
			if got := nerror.Status(tc.err); got != tc.status {
				t.Fatalf("Status = %d, want %d", got, tc.status)
			}
		})
	}
}

func TestHasCodeAndCatalogIs(t *testing.T) {
	err := fmt.Errorf("handler: %w", testCatalog.Wrap("user_not_found", errors.New("sql: no rows")))

	if !nerror.HasCode(err, "user_not_found") {
		t.Fatal("HasCode must find the code through fmt wrapping")
	}
	if !testCatalog.Is(err, "user_not_found") {
		t.Fatal("Catalog.Is must match by code")
	}
	if testCatalog.Is(err, "quota_exceeded") {
		t.Fatal("Catalog.Is matched the wrong code")
	}
	if nerror.HasCode(nil, "user_not_found") {
		t.Fatal("nil error must not match")
	}
	if nerror.HasCode(errors.New("plain"), "internal") {
		t.Fatal("HasCode must not invent codes for non-neji errors")
	}

	// Agreement with the errors.Is form documented in the README.
	joined := errors.Join(errors.New("a"), testCatalog.New("invalid_payload"))
	if nerror.HasCode(joined, "invalid_payload") != errors.Is(joined, testCatalog.New("invalid_payload")) {
		t.Fatal("HasCode and errors.Is disagree on a joined error")
	}
}

func BenchmarkHasCode(b *testing.B) {
	err := testCatalog.Wrap("user_not_found", errors.New("sql: no rows"))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !nerror.HasCode(err, "user_not_found") {
			b.Fatal("no match")
		}
	}
}
