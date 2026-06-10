package nerror

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"
)

// maxParseBody caps how much of an error response body Parse will read.
const maxParseBody = 1 << 20 // 1 MiB

// Parse turns an HTTP error response back into a *Error — the inverse of
// Write. It returns nil when the status is below 400; otherwise it decodes
// the body and returns a *Error, so Go clients of a neji-styled API can
// match server-declared codes across the wire:
//
//	resp, err := client.Do(req)
//	if err != nil {
//	    return err
//	}
//	defer resp.Body.Close()
//	if err := nerror.Parse(resp); err != nil {
//	    if errors.Is(err, api.Errors.New("quota_exceeded")) {
//	        backoff()
//	    }
//	    return err
//	}
//
// Parse understands both wire formats neji produces — the default
// {"error": {...}} envelope and RFC 9457 problem+json — including the
// correlation ID, and reads the Retry-After header into RetryAfter. A body
// in neither format (a plain nginx 502 page, say) still yields a *Error
// carrying the response status and code "http_error", never a nil error
// for an error status.
//
// Parse consumes resp.Body (up to 1 MiB); closing it remains the caller's
// responsibility.
func Parse(resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}

	e := &Error{
		Status:  resp.StatusCode,
		Code:    "http_error",
		Message: http.StatusText(resp.StatusCode),
	}
	if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && seconds > 0 {
		e.RetryAfter = time.Duration(seconds) * time.Second
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxParseBody))
	if err != nil || len(body) == 0 {
		return e
	}

	var env envelope
	if json.Unmarshal(body, &env) == nil && env.Error != nil && env.Error.Code != "" {
		e.Code = env.Error.Code
		e.Message = env.Error.Message
		e.Details = env.Error.Details
		e.CorrelationID = env.Error.CorrelationID
		if env.Error.Status != 0 {
			e.Status = env.Error.Status
		}
		return e
	}

	var p problem
	if json.Unmarshal(body, &p) == nil && (p.Code != "" || p.Detail != "" || p.Title != "") {
		if p.Code != "" {
			e.Code = p.Code
		}
		switch {
		case p.Detail != "":
			e.Message = p.Detail
		case p.Title != "":
			e.Message = p.Title
		}
		e.Details = p.Details
		e.CorrelationID = p.CorrelationID
		if p.Type != "" && p.Type != "about:blank" {
			e.Type = p.Type
		}
		return e
	}

	return e
}
