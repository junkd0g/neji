// Package nerror turns your API's errors into data.
//
// Errors are declared once in a Catalog — a map of stable, machine-readable
// codes to their HTTP status, message and metadata. The same *Error value
// renders as JSON over HTTP (Write), as RFC 9457 problem+json (WriteProblem),
// maps to a gRPC status code (GRPCCode), and logs structured fields via
// log/slog. The catalog can also generate its own documentation (Markdown,
// OpenAPI), so the docs can never go stale.
package nerror

import (
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// Debug switches neji into development mode: Write and WriteProblem
// include the wrapped cause chain and a stack trace (captured when the
// error was created) in the response body.
//
// Leave it false in production — responses then carry none of those
// internals, only a correlation ID that links the client's response to
// your logs (see OnWrite). Set once at startup, before serving traffic.
var Debug bool

// Error is a structured API error. It carries everything needed to render
// the error over any protocol: a stable machine-readable Code, an HTTP
// Status, a human-readable Message, optional per-instance Details, and an
// optional wrapped cause.
//
// Error implements the error interface as well as Unwrap (for errors.Is /
// errors.As chains) and slog.LogValuer (for structured logging).
type Error struct {
	// Code is a stable, machine-readable identifier such as "user_not_found".
	// Clients should program against Code, never against Message.
	Code string `json:"code"`

	// Status is the HTTP status code for this error.
	Status int `json:"status"`

	// Message is a human-readable description, safe to show to API clients.
	Message string `json:"message"`

	// Details holds optional structured context for this specific
	// occurrence, e.g. which fields failed validation.
	Details map[string]any `json:"details,omitempty"`

	// RetryAfter, when non-zero, signals the client may retry after this
	// duration. Write sets the Retry-After header from it.
	RetryAfter time.Duration `json:"-"`

	// Type is an optional RFC 9457 problem type URI. WriteProblem falls
	// back to "about:blank" when empty.
	Type string `json:"-"`

	// GRPC, when non-zero, overrides the gRPC code derived from Status.
	GRPC GRPCCode `json:"-"`

	// CorrelationID links a client-visible response to server logs.
	// Write stamps one automatically when empty; set it yourself with
	// WithCorrelationID to reuse a request or trace ID.
	CorrelationID string `json:"correlation_id,omitempty"`

	// Err is the wrapped cause. It is never serialized to clients; it is
	// reachable via errors.Unwrap and included in slog output.
	Err error `json:"-"`

	// stack holds program counters captured at creation when Debug is on.
	stack []uintptr
}

// New creates an Error with the given HTTP status, machine-readable code
// and human-readable message.
func New(status int, code, message string) *Error {
	e := &Error{Status: status, Code: code, Message: message}
	if Debug {
		e.stack = captureStack()
	}
	return e
}

// Error implements the error interface. The wrapped cause, if any, is
// appended so log lines stay informative; it is never sent to API clients.
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %s", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped cause, enabling errors.Is and errors.As to
// traverse the chain.
func (e *Error) Unwrap() error {
	return e.Err
}

// Is reports whether target is an *Error with the same Code. This makes
// errors.Is(err, Catalog.New("user_not_found")) match regardless of
// per-instance details or wrapped causes.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Code == e.Code
}

// Wrap returns a copy of e with err attached as the cause. The original
// error is not mutated, so catalog entries can be used as sentinels.
func (e *Error) Wrap(err error) *Error {
	c := e.clone()
	c.Err = err
	if Debug && c.stack == nil {
		c.stack = captureStack()
	}
	return c
}

// WithCorrelationID returns a copy of e carrying the given correlation ID,
// e.g. an existing request or trace ID. When unset, Write stamps a random
// one.
func (e *Error) WithCorrelationID(id string) *Error {
	c := e.clone()
	c.CorrelationID = id
	return c
}

// With returns a copy of e with the key/value pair added to Details.
func (e *Error) With(key string, value any) *Error {
	c := e.clone()
	if c.Details == nil {
		c.Details = make(map[string]any, 1)
	}
	c.Details[key] = value
	return c
}

// WithMessage returns a copy of e with the message replaced. The Code, and
// therefore client-side handling, stays the same.
func (e *Error) WithMessage(format string, args ...any) *Error {
	c := e.clone()
	c.Message = fmt.Sprintf(format, args...)
	return c
}

// GRPCCode returns the gRPC status code for this error: the explicit GRPC
// override when set, otherwise a code derived from the HTTP status.
func (e *Error) GRPCCode() GRPCCode {
	if e.GRPC != 0 {
		return e.GRPC
	}
	return grpcFromHTTP(e.Status)
}

// LogValue implements slog.LogValuer, so logging a *Error with log/slog
// emits its code, status, message and cause as structured fields.
func (e *Error) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("code", e.Code),
		slog.Int("status", e.Status),
		slog.String("message", e.Message),
	}
	if e.CorrelationID != "" {
		attrs = append(attrs, slog.String("correlation_id", e.CorrelationID))
	}
	if e.Err != nil {
		attrs = append(attrs, slog.String("cause", e.Err.Error()))
	}
	if len(e.Details) > 0 {
		attrs = append(attrs, slog.Any("details", e.Details))
	}
	return slog.GroupValue(attrs...)
}

func (e *Error) clone() *Error {
	c := *e
	c.Details = maps.Clone(e.Details)
	return &c
}

// captureStack records the callers of a constructor (skipping runtime
// internals, captureStack itself and the constructor frame).
func captureStack() []uintptr {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(3, pcs)
	return pcs[:n]
}

// stackFrames renders the captured stack as "function file:line" strings.
// Leading frames inside neji itself (chained constructors like
// Catalog.Wrap -> Catalog.New) are trimmed so the first frame is the
// user's creation site.
func (e *Error) stackFrames() []string {
	if len(e.stack) == 0 {
		return nil
	}
	const selfPkg = "github.com/junkd0g/neji/v2."
	frames := runtime.CallersFrames(e.stack)
	var out []string
	for {
		f, more := frames.Next()
		if len(out) == 0 && strings.HasPrefix(f.Function, selfPkg) {
			if !more {
				break
			}
			continue
		}
		out = append(out, fmt.Sprintf("%s %s:%d", f.Function, f.File, f.Line))
		if !more {
			break
		}
	}
	return out
}

// Shorthand constructors for ad-hoc errors outside a Catalog. For errors
// that are part of your API's contract, prefer declaring them in a Catalog.

// BadRequest returns a 400 error with code "bad_request".
func BadRequest(message string) *Error {
	return New(http.StatusBadRequest, "bad_request", message)
}

// Unauthorized returns a 401 error with code "unauthorized".
func Unauthorized(message string) *Error {
	return New(http.StatusUnauthorized, "unauthorized", message)
}

// Forbidden returns a 403 error with code "forbidden".
func Forbidden(message string) *Error {
	return New(http.StatusForbidden, "forbidden", message)
}

// NotFound returns a 404 error with code "not_found".
func NotFound(message string) *Error {
	return New(http.StatusNotFound, "not_found", message)
}

// Conflict returns a 409 error with code "conflict".
func Conflict(message string) *Error {
	return New(http.StatusConflict, "conflict", message)
}

// UnprocessableEntity returns a 422 error with code "unprocessable_entity".
func UnprocessableEntity(message string) *Error {
	return New(http.StatusUnprocessableEntity, "unprocessable_entity", message)
}

// TooManyRequests returns a 429 error with code "too_many_requests".
func TooManyRequests(message string) *Error {
	return New(http.StatusTooManyRequests, "too_many_requests", message)
}

// Internal returns a 500 error with code "internal".
func Internal(message string) *Error {
	return New(http.StatusInternalServerError, "internal", message)
}

// Unavailable returns a 503 error with code "unavailable".
func Unavailable(message string) *Error {
	return New(http.StatusServiceUnavailable, "unavailable", message)
}
