package nerror

import (
	"fmt"
	"net/http"
)

// HandlerFunc is an http.HandlerFunc that may return an error instead of
// writing the error response by hand.
type HandlerFunc func(http.ResponseWriter, *http.Request) error

// Handler adapts a HandlerFunc into an http.Handler. A returned error is
// sent to the client with Write — catalog errors keep their status and
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
// Handler also recovers panics into a generic JSON 500, so it doubles as a
// per-route safety net. http.ErrAbortHandler is re-panicked, as net/http
// uses it deliberately to abort a response.
//
// A handler that returns an error must not have written to w already;
// Handler cannot un-send a partial response.
func Handler(h HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				Write(w, Internal("internal server error").
					Wrap(fmt.Errorf("panic: %v", rec)))
			}
		}()

		if err := h(w, r); err != nil {
			Write(w, err)
		}
	})
}
