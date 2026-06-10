package nerror

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
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
var OnWrite func(*Error)

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
// status code, a correlation ID and, for retryable errors, the
// Retry-After header — the full response in one call.
//
// If err is (or wraps) a *Error, its status and fields are used. Any other
// error becomes a generic 500 with code "internal"; its message is NOT
// sent to the client, so internal details never leak by accident. With
// Debug on, the response additionally includes the cause chain and a
// stack trace.
func Write(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	e := prepare(err)

	var body any = envelope{Error: e}
	if Debug {
		body = struct {
			Error debugError `json:"error"`
		}{debugError{e, causeOf(e), e.stackFrames()}}
	}
	writeJSON(w, e, body)
}

// WriteProblem sends err to the client as an RFC 9457 problem details
// document with Content-Type application/problem+json. The error's Type
// URI is used when set, "about:blank" otherwise; Code, Details and the
// correlation ID are included as extension members.
//
// Non-*Error values and Debug are handled exactly as in Write.
func WriteProblem(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	e := prepare(err)

	typeURI := e.Type
	if typeURI == "" {
		typeURI = "about:blank"
	}
	body := problem{
		Type:          typeURI,
		Title:         http.StatusText(e.Status),
		Status:        e.Status,
		Detail:        e.Message,
		Code:          e.Code,
		Details:       e.Details,
		CorrelationID: e.CorrelationID,
	}
	if Debug {
		body.Cause = causeOf(e)
		body.Stack = e.stackFrames()
	}
	writeJSON(w, e, body)
}

// prepare normalizes err to a *Error and stamps a correlation ID when none
// is set. Unknown errors map to a generic 500 so their internals are never
// serialized to clients.
func prepare(err error) *Error {
	var e *Error
	if !errors.As(err, &e) {
		e = Internal("internal server error").Wrap(err)
	}
	if e.CorrelationID == "" {
		e = e.WithCorrelationID(newCorrelationID())
	}
	return e
}

func causeOf(e *Error) string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func newCorrelationID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, e *Error, body any) {
	if OnWrite != nil {
		OnWrite(e)
	}

	if e.RetryAfter > 0 {
		seconds := int(math.Ceil(e.RetryAfter.Seconds()))
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
	}

	payload, err := json.Marshal(body)
	if err != nil {
		// Details contained something unmarshalable; degrade to a bare
		// response rather than sending a broken body.
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(e.Status)
	w.Write(payload)
}
