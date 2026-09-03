package nerror

import (
	"fmt"
	"net/http"
	"sort"
	"time"
)

// Definition declares one error in a Catalog: its HTTP status, default
// message and protocol metadata. Definitions are data — the single source
// of truth for every error your API can return.
type Definition struct {
	// Status is the HTTP status code (4xx or 5xx).
	Status int

	// Message is the default human-readable message, safe for API clients.
	Message string

	// RetryAfter, when non-zero, marks the error as retryable after this
	// duration. Write sets the Retry-After header from it.
	RetryAfter time.Duration

	// GRPC optionally overrides the gRPC code derived from Status.
	GRPC GRPCCode

	// Type is an optional RFC 9457 problem type URI for WriteProblem.
	Type string
}

// Catalog maps stable machine-readable codes to their Definition. Declare
// it once, package-level, and construct every error through it:
//
//	var Errors = nerror.Catalog{
//	    "user_not_found":  {Status: 404, Message: "user does not exist"},
//	    "quota_exceeded":  {Status: 429, Message: "quota exceeded", RetryAfter: time.Minute},
//	    "invalid_payload": {Status: 422, Message: "validation failed"},
//	}
//
//	return Errors.New("user_not_found")
//
// Because the catalog is data, it can also generate its own documentation:
// see Markdown and OpenAPI.
type Catalog map[string]Definition

// New returns the Error declared under code. Each call returns a fresh
// value, so callers may attach details or a cause without affecting others.
//
// An unknown code yields a 500 Error with code "unknown_error" — a visible
// bug signal rather than a panic. Catch these at startup with Validate.
func (c Catalog) New(code string) *Error {
	def, ok := c[code]
	e := &Error{
		Code:       code,
		Status:     def.Status,
		Message:    def.Message,
		RetryAfter: def.RetryAfter,
		Type:       def.Type,
		GRPC:       def.GRPC,
	}
	if !ok {
		e.Status = http.StatusInternalServerError
		e.Code = "unknown_error"
		e.Message = fmt.Sprintf("unknown error code %q", code)
	}
	if Debug {
		// Captured for unknown codes as well: that stack points straight
		// at the typo.
		e.stack = captureStack()
	}
	return e
}

// Newf returns the Error declared under code with its message replaced by
// the formatted one. The code, status and metadata are unchanged.
func (c Catalog) Newf(code, format string, args ...any) *Error {
	return c.New(code).WithMessage(format, args...)
}

// Wrap returns the Error declared under code with err attached as the
// cause. The cause is never serialized to clients.
func (c Catalog) Wrap(code string, err error) *Error {
	return c.New(code).Wrap(err)
}

// Has reports whether code is declared in the catalog.
func (c Catalog) Has(code string) bool {
	_, ok := c[code]
	return ok
}

// Codes returns all declared codes, sorted, for deterministic iteration.
func (c Catalog) Codes() []string {
	codes := make([]string, 0, len(c))
	for code := range c {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

// Validate checks every definition for mistakes that would otherwise only
// surface at request time: empty codes, missing messages and statuses
// outside the 4xx–5xx error range. Call it once at startup (or in a test):
//
//	func TestCatalog(t *testing.T) {
//	    if err := Errors.Validate(); err != nil {
//	        t.Fatal(err)
//	    }
//	}
func (c Catalog) Validate() error {
	for _, code := range c.Codes() {
		def := c[code]
		if code == "" {
			return fmt.Errorf("nerror: catalog contains an empty code")
		}
		if def.Message == "" {
			return fmt.Errorf("nerror: code %q has no message", code)
		}
		if def.Status < 400 || def.Status > 599 {
			return fmt.Errorf("nerror: code %q has status %d, want 400-599", code, def.Status)
		}
	}
	return nil
}
