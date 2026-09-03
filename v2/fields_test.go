package nerror_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	nerror "github.com/junkd0g/neji/v2"
)

func TestWithFieldAppendsAndDoesNotMutate(t *testing.T) {
	base := testCatalog.New("invalid_payload")
	one := base.WithField("email", "must be a valid address")
	two := one.WithFields(
		nerror.FieldError{Field: "age", Message: "must be positive"},
		nerror.FieldError{Field: "name", Message: "is required"},
	)

	if got := nerror.Fields(base); got != nil {
		t.Fatalf("base must stay untouched, got %v", got)
	}
	if got := nerror.Fields(one); len(got) != 1 || got[0].Field != "email" {
		t.Fatalf("one = %v", got)
	}
	got := nerror.Fields(two)
	if len(got) != 3 || got[0].Field != "email" || got[1].Field != "age" || got[2].Field != "name" {
		t.Fatalf("two = %v", got)
	}

	// Appending to two must not reach back into one's slice.
	_ = two.WithField("extra", "x")
	if len(nerror.Fields(two)) != 3 {
		t.Fatal("WithField mutated its receiver")
	}
	if len(nerror.Fields(one)) != 1 {
		t.Fatal("WithField mutated an ancestor")
	}
	if len(nerror.Fields(base.WithFields())) != 0 {
		t.Fatal("WithFields with no arguments must add nothing")
	}
}

func TestFieldsOnWireAndAfterParse(t *testing.T) {
	err := testCatalog.New("invalid_payload").
		WithField("email", "must be a valid address").
		WithField("age", "must be positive")

	rec := httptest.NewRecorder()
	nerror.Write(rec, err)

	var body struct {
		Error struct {
			Details struct {
				Fields []struct {
					Field   string `json:"field"`
					Message string `json:"message"`
				} `json:"fields"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if f := body.Error.Details.Fields; len(f) != 2 || f[0].Field != "email" || f[1].Message != "must be positive" {
		t.Fatalf("wire fields = %+v", f)
	}

	// A Go client sees the same fields after Parse, via the generic
	// map[string]any decoding.
	parsed := nerror.Parse(rec.Result())
	got := nerror.Fields(parsed)
	if len(got) != 2 || got[0] != (nerror.FieldError{Field: "email", Message: "must be a valid address"}) {
		t.Fatalf("parsed fields = %v", got)
	}

	if nerror.Fields(nil) != nil || nerror.Fields(testCatalog.New("user_not_found")) != nil {
		t.Fatal("Fields must be nil when there are none")
	}
}
