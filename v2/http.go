package nerror

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	mathrand "math/rand/v2"
	"net/http"
	"strconv"
)

// OnWrite, when set, is called with every error sent by Write or
// WriteProblem, after the correlation ID is stamped. It is the place to
// hook metrics and logging — the logged correlation ID matches the one
// the client received:
//
//	nerror.OnWrite = func(e *nerror.Error) {
//	    apiErrors.WithLabelValues(e.Code, strconv.Itoa(e.Status)).Inc()
//	    slog.Error("api error", "error", e)
//	}
//
// Set it once during startup, before serving traffic; it must be safe for
// concurrent use and must not block.
//
// Errors written by Handler, WriteFor or WriteProblemFor prefer
// OnWriteRequest when it is set and fall back to OnWrite otherwise, so
// setting both never logs the same error twice.
var OnWrite func(*Error)

// OnWriteRequest is the request-aware variant of OnWrite. It receives the
// *http.Request alongside the error, so log lines can carry the method,
// path, client address or anything else on the request:
//
//	nerror.OnWriteRequest = func(e *nerror.Error, r *http.Request) {
//	    slog.Error("api error", "method", r.Method, "path", r.URL.Path, "error", e)
//	}
//
// It is called for errors written by Handler, WriteFor and WriteProblemFor.
// Plain Write and WriteProblem have no request in scope and always call
// OnWrite. Like OnWrite, set it once at startup; it must be safe for
// concurrent use and must not block.
var OnWriteRequest func(*Error, *http.Request)

// CorrelationIDHeader names the response header that carries the
// correlation ID, in addition to the JSON body, so proxies, log pipelines
// and clients can read it without parsing the response. Set it to "" at
// startup to disable the header.
var CorrelationIDHeader = "X-Correlation-ID"

// RequestIDHeader names the request header Handler, WriteFor and
// WriteProblemFor read an incoming request ID from. When a request carries
// one (and the error has no CorrelationID of its own), it is reused as the
// correlation ID, so the ID stays the same end to end: client, proxy,
// this service and its logs. CorrelationIDHeader is consulted as a
// fallback. Set it to "" at startup to disable the pickup.
//
// Incoming IDs are only accepted if they are 1–128 characters of ASCII
// letters, digits and the punctuation "-", "_", "." and ":"; anything else
// is ignored and a fresh ID is generated instead.
var RequestIDHeader = "X-Request-ID"

// maxRequestIDLen caps the length of an incoming request ID neji reuses.
const maxRequestIDLen = 128

// envelope is the default JSON response shape:
//
//	{
//	    "error": {
//	        "code": "user_not_found",
//	        "status": 404,
//	        "message": "user does not exist",
//	        "correlation_id": "f3b1c2d4e5a6b7c8"
//	    }
//	}
//
// With Debug on, the object additionally carries "cause" and "stack".
type envelope struct {
	Error *Error `json:"error"`
}

// debugError augments the client-visible Error with the internals that
// are only safe to expose in development.
type debugError struct {
	*Error
	Cause string   `json:"cause,omitempty"`
	Stack []string `json:"stack,omitempty"`
}

// problem is the RFC 9457 problem details shape, served as
// application/problem+json. Code, Details and CorrelationID ride along as
// extension members, which the RFC explicitly allows; Cause and Stack
// appear only with Debug on.
type problem struct {
	Type          string         `json:"type"`
	Title         string         `json:"title"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail,omitempty"`
	Code          string         `json:"code,omitempty"`
	Details       map[string]any `json:"details,omitempty"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Cause         string         `json:"cause,omitempty"`
	Stack         []string       `json:"stack,omitempty"`
}

// Write sends err to the client as JSON: it sets Content-Type, the HTTP
// status code, a correlation ID (in the body and, by default, the
// X-Correlation-ID header) and, for retryable errors, the Retry-After
// header — the full response in one call.
//
// If err is (or wraps) a *Error, its status and fields are used. Any other
// error becomes a generic 500 with code "internal"; its message is NOT
// sent to the client, so internal details never leak by accident. With
// Debug on, the response additionally includes the cause chain and a
// stack trace.
func Write(w http.ResponseWriter, err error) {
	write(w, nil, err, false)
}

// WriteFor is Write with the request in scope: an incoming request ID (see
// RequestIDHeader) is reused as the correlation ID, and OnWriteRequest is
// preferred over OnWrite. Handler uses it; call it yourself from handlers
// that have a different shape, such as router-specific ones.
func WriteFor(w http.ResponseWriter, r *http.Request, err error) {
	write(w, r, err, false)
}

// WriteProblem sends err to the client as an RFC 9457 problem details
// document with Content-Type application/problem+json. The error's Type
// URI is used when set, "about:blank" otherwise; Code, Details and the
// correlation ID are included as extension members.
//
// Non-*Error values and Debug are handled exactly as in Write.
func WriteProblem(w http.ResponseWriter, err error) {
	write(w, nil, err, true)
}

// WriteProblemFor is WriteProblem with the request in scope; see WriteFor.
func WriteProblemFor(w http.ResponseWriter, r *http.Request, err error) {
	write(w, r, err, true)
}

// write is the shared implementation behind Write, WriteFor, WriteProblem
// and WriteProblemFor. r may be nil.
func write(w http.ResponseWriter, r *http.Request, err error, asProblem bool) {
	if asProblem {
		w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	e := prepare(err, requestID(r))

	var body any
	switch {
	case asProblem:
		typeURI := e.Type
		if typeURI == "" {
			typeURI = "about:blank"
		}
		p := problem{
			Type:          typeURI,
			Title:         http.StatusText(e.Status),
			Status:        e.Status,
			Detail:        e.Message,
			Code:          e.Code,
			Details:       e.Details,
			CorrelationID: e.CorrelationID,
		}
		if Debug {
			p.Cause = causeOf(e)
			p.Stack = e.stackFrames()
		}
		body = p
	case Debug:
		body = struct {
			Error debugError `json:"error"`
		}{debugError{e, causeOf(e), e.stackFrames()}}
	default:
		body = envelope{Error: e}
	}

	notify(e, r)

	if CorrelationIDHeader != "" {
		w.Header().Set(CorrelationIDHeader, e.CorrelationID)
	}
	if e.RetryAfter > 0 {
		seconds := int(math.Ceil(e.RetryAfter.Seconds()))
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
	}

	payload, merr := json.Marshal(body)
	if merr != nil {
		// Details contained something unmarshalable; degrade to a bare
		// response rather than sending a broken body.
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(e.Status)
	_, _ = w.Write(payload)
}

// prepare normalizes err to a *Error and stamps a correlation ID when none
// is set: the given id when non-empty, a fresh random one otherwise.
// Unknown errors map to a generic 500 so their internals are never
// serialized to clients.
func prepare(err error, id string) *Error {
	var e *Error
	if !errors.As(err, &e) {
		e = Internal("internal server error").Wrap(err)
	}
	if e.CorrelationID == "" {
		if id == "" {
			id = newCorrelationID()
		}
		e = e.WithCorrelationID(id)
	}
	return e
}

// notify dispatches to the write hooks: OnWriteRequest when a request is
// in scope and the hook is set, OnWrite otherwise.
func notify(e *Error, r *http.Request) {
	if r != nil && OnWriteRequest != nil {
		OnWriteRequest(e, r)
		return
	}
	if OnWrite != nil {
		OnWrite(e)
	}
}

// requestID returns a usable incoming request ID from r, or "" when r is
// nil, no header is set or the value is not safe to echo back.
func requestID(r *http.Request) string {
	if r == nil {
		return ""
	}
	for _, name := range []string{RequestIDHeader, CorrelationIDHeader} {
		if name == "" {
			continue
		}
		if id := r.Header.Get(name); validRequestID(id) {
			return id
		}
	}
	return ""
}

// validRequestID reports whether an incoming ID is safe to reuse: short
// and made only of characters that cannot break a header or a log line.
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '-', c == '_', c == '.', c == ':':
		default:
			return false
		}
	}
	return true
}

func causeOf(e *Error) string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

// newCorrelationID returns 16 hex characters of randomness. crypto/rand
// cannot fail on Go 1.24+, but on older toolchains it may; a
// pseudo-random ID still serves its purpose of correlating a response
// with a log line.
func newCorrelationID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		binary.BigEndian.PutUint64(b, mathrand.Uint64())
	}
	return hex.EncodeToString(b)
}
