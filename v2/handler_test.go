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
