package nerror_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nerror "github.com/junkd0g/neji/v2"
)

func TestHandlerWritesReturnedError(t *testing.T) {
	h := nerror.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return testCatalog.New("user_not_found")
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/users/42", nil))

	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"user_not_found"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestHandlerSuccessPassesThrough(t *testing.T) {
	h := nerror.Handler(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(`{"id":"42"}`))
		return err
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/users", nil))

	if rec.Code != 201 {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if rec.Body.String() != `{"id":"42"}` {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestHandlerRecoversPanics(t *testing.T) {
	h := nerror.Handler(func(w http.ResponseWriter, r *http.Request) error {
		panic("boom: secret connection string")
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != 500 {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"code":"internal"`) {
		t.Fatalf("expected generic internal error: %s", body)
	}
	if strings.Contains(body, "secret") {
		t.Fatalf("panic value leaked to client: %s", body)
	}
}

func TestHandlerRepanicsErrAbortHandler(t *testing.T) {
	h := nerror.Handler(func(w http.ResponseWriter, r *http.Request) error {
		panic(http.ErrAbortHandler)
	})

	defer func() {
		if recover() != http.ErrAbortHandler {
			t.Fatal("http.ErrAbortHandler must be re-panicked")
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	t.Fatal("should not reach here")
}

func TestOnWriteHook(t *testing.T) {
	var codes []string
	var correlationIDs []string
	nerror.OnWrite = func(e *nerror.Error) {
		codes = append(codes, e.Code)
		correlationIDs = append(correlationIDs, e.CorrelationID)
	}
	defer func() { nerror.OnWrite = nil }()

	rec := httptest.NewRecorder()
	nerror.Write(rec, testCatalog.New("user_not_found"))
	nerror.WriteProblem(httptest.NewRecorder(), testCatalog.New("quota_exceeded"))

	if len(codes) != 2 || codes[0] != "user_not_found" || codes[1] != "quota_exceeded" {
		t.Fatalf("OnWrite saw %v", codes)
	}

	// The hook must see the same correlation ID the client received,
	// so logs and responses can be matched.
	if correlationIDs[0] == "" || !strings.Contains(rec.Body.String(), correlationIDs[0]) {
		t.Fatalf("hook correlation ID %q not in response %s", correlationIDs[0], rec.Body.String())
	}
}

func TestHandlerReusesIncomingRequestID(t *testing.T) {
	h := nerror.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return testCatalog.New("user_not_found")
	})

	req := httptest.NewRequest("GET", "/users/42", nil)
	req.Header.Set("X-Request-ID", "trace-abc.123:7")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Correlation-ID"); got != "trace-abc.123:7" {
		t.Fatalf("X-Correlation-ID = %q, want the incoming request ID", got)
	}
	if !strings.Contains(rec.Body.String(), `"correlation_id":"trace-abc.123:7"`) {
		t.Fatalf("body must carry the incoming ID: %s", rec.Body.String())
	}
}

func TestHandlerFallsBackToCorrelationHeaderAndRejectsUnsafeIDs(t *testing.T) {
	h := nerror.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return testCatalog.New("user_not_found")
	})

	// Fallback: no X-Request-ID, but an X-Correlation-ID.
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Correlation-ID", "from-upstream")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Correlation-ID"); got != "from-upstream" {
		t.Fatalf("fallback header not used, got %q", got)
	}

	// Unsafe values (spaces, control characters, too long) are ignored
	// and a fresh 16-hex-char ID is generated instead.
	for _, bad := range []string{"has space", "semi;colon", strings.Repeat("a", 129), "<script>"} {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Request-ID", bad)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if got := rec.Header().Get("X-Correlation-ID"); got == bad || len(got) != 16 {
			t.Fatalf("unsafe ID %q was reused or no fresh ID generated: %q", bad, got)
		}
	}

	// An explicit WithCorrelationID always wins over the request header.
	h2 := nerror.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return testCatalog.New("user_not_found").WithCorrelationID("explicit")
	})
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "incoming")
	rec = httptest.NewRecorder()
	h2.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Correlation-ID"); got != "explicit" {
		t.Fatalf("explicit correlation ID overridden: %q", got)
	}
}

func TestOnWriteRequestHook(t *testing.T) {
	var seenPaths, seenCodes []string
	var plainCalls int
	nerror.OnWriteRequest = func(e *nerror.Error, r *http.Request) {
		seenPaths = append(seenPaths, r.Method+" "+r.URL.Path)
		seenCodes = append(seenCodes, e.Code)
	}
	nerror.OnWrite = func(e *nerror.Error) { plainCalls++ }
	defer func() { nerror.OnWriteRequest, nerror.OnWrite = nil, nil }()

	h := nerror.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return testCatalog.New("user_not_found")
	})
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("DELETE", "/users/42", nil))
	nerror.WriteFor(httptest.NewRecorder(), httptest.NewRequest("POST", "/orders", nil), testCatalog.New("quota_exceeded"))
	nerror.WriteProblemFor(httptest.NewRecorder(), httptest.NewRequest("PUT", "/x", nil), testCatalog.New("invalid_payload"))

	if len(seenPaths) != 3 || seenPaths[0] != "DELETE /users/42" || seenPaths[1] != "POST /orders" || seenPaths[2] != "PUT /x" {
		t.Fatalf("OnWriteRequest saw %v", seenPaths)
	}
	if seenCodes[0] != "user_not_found" || seenCodes[1] != "quota_exceeded" || seenCodes[2] != "invalid_payload" {
		t.Fatalf("OnWriteRequest codes = %v", seenCodes)
	}
	if plainCalls != 0 {
		t.Fatalf("OnWrite must not also fire for request-aware writes, fired %d times", plainCalls)
	}

	// Plain Write has no request: it always uses OnWrite.
	nerror.Write(httptest.NewRecorder(), testCatalog.New("user_not_found"))
	if plainCalls != 1 || len(seenPaths) != 3 {
		t.Fatalf("plain Write dispatched wrongly: OnWrite=%d, OnWriteRequest=%d", plainCalls, len(seenPaths))
	}

	// Without OnWriteRequest, request-aware writes fall back to OnWrite.
	nerror.OnWriteRequest = nil
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if plainCalls != 2 {
		t.Fatalf("Handler did not fall back to OnWrite, calls=%d", plainCalls)
	}
}

func TestHandlerDoesNotAppendErrorToStartedResponse(t *testing.T) {
	var hooked []string
	nerror.OnWrite = func(e *nerror.Error) { hooked = append(hooked, e.Code) }
	defer func() { nerror.OnWrite = nil }()

	h := nerror.Handler(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial"))
		return testCatalog.New("user_not_found")
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if rec.Code != 200 || rec.Body.String() != "partial" {
		t.Fatalf("started response was altered: %d %q", rec.Code, rec.Body.String())
	}
	if len(hooked) != 1 || hooked[0] != "user_not_found" {
		t.Fatalf("late error must still reach the hook, got %v", hooked)
	}
}

func TestHandlerRepanicsAfterResponseStarted(t *testing.T) {
	h := nerror.Handler(func(w http.ResponseWriter, r *http.Request) error {
		_, _ = w.Write([]byte("partial"))
		panic("boom")
	})

	defer func() {
		if rec := recover(); rec != "boom" {
			t.Fatalf("panic after a started response must propagate to net/http, got %v", rec)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	t.Fatal("should not reach here")
}

func TestHandlerWriterSupportsFlushAndUnwrap(t *testing.T) {
	var flushed, unwrapped bool
	h := nerror.Handler(func(w http.ResponseWriter, r *http.Request) error {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
			flushed = true
		}
		if u, ok := w.(interface{ Unwrap() http.ResponseWriter }); ok {
			_, unwrapped = u.Unwrap().(*httptest.ResponseRecorder)
		}
		// ResponseController must reach the recorder through Unwrap.
		if err := http.NewResponseController(w).Flush(); err != nil {
			return err
		}
		return nil
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if !flushed || !unwrapped || !rec.Flushed {
		t.Fatalf("flushed=%v unwrapped=%v recorder.Flushed=%v", flushed, unwrapped, rec.Flushed)
	}
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
}
