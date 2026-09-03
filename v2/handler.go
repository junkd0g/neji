package nerror

import (
	"fmt"
	"net/http"
)

// HandlerFunc is an http.HandlerFunc that may return an error instead of
// writing the error response by hand.
type HandlerFunc func(http.ResponseWriter, *http.Request) error

// Handler adapts a HandlerFunc into an http.Handler. A returned error is
// sent to the client with WriteFor — catalog errors keep their status and
// code, anything else becomes a generic 500 with no internals leaked:
//
//	mux.Handle("GET /users/{id}", nerror.Handler(func(w http.ResponseWriter, r *http.Request) error {
//	    user, err := db.GetUser(r.PathValue("id"))
//	    if err != nil {
//	        return Errors.Wrap("user_not_found", err)
//	    }
//	    return json.NewEncoder(w).Encode(user)
//	}))
//
// An incoming request ID (see RequestIDHeader) becomes the response's
// correlation ID, and OnWriteRequest, when set, receives the request
// alongside the error.
//
// Handler also recovers panics into a generic JSON 500, so it doubles as a
// per-route safety net. http.ErrAbortHandler is re-panicked, as net/http
// uses it deliberately to abort a response.
//
// A handler that returns an error or panics after it has started writing
// the response cannot be un-sent. In that case Handler does not append an
// error body to the partial response: a returned error is only reported
// to the write hooks, and a panic is re-raised so net/http logs it and
// closes the connection.
//
// The http.ResponseWriter passed to h is a thin wrapper that tracks
// whether the response has started. It implements http.Flusher and
// exposes the original writer through Unwrap, so http.ResponseController
// (Flush, Hijack, SetWriteDeadline, ...) works as usual.
func Handler(h HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{ResponseWriter: w}
		defer func() {
			if rec := recover(); rec != nil {
				if rec == http.ErrAbortHandler || rw.started {
					panic(rec)
				}
				write(rw, r, Internal("internal server error").
					Wrap(fmt.Errorf("panic: %v", rec)), false)
			}
		}()

		if err := h(rw, r); err != nil {
			if rw.started {
				notify(prepare(err, requestID(r)), r)
				return
			}
			write(rw, r, err, false)
		}
	})
}

// responseWriter records whether the response has started so Handler can
// tell a clean error return from one that arrives too late.
type responseWriter struct {
	http.ResponseWriter
	started bool
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.started = true
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(p []byte) (int, error) {
	rw.started = true
	return rw.ResponseWriter.Write(p)
}

// Flush implements http.Flusher so streaming handlers keep working when
// they type-assert the writer directly.
func (rw *responseWriter) Flush() {
	rw.started = true
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
		return
	}
	_ = http.NewResponseController(rw.ResponseWriter).Flush()
}

// Unwrap exposes the original writer to http.ResponseController.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}
