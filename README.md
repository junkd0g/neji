[![Go Report Card](https://goreportcard.com/badge/github.com/junkd0g/neji/v2)](https://goreportcard.com/report/github.com/junkd0g/neji/v2)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://pkg.go.dev/badge/github.com/junkd0g/neji/v2.svg)](https://pkg.go.dev/github.com/junkd0g/neji/v2)

# neji — your API's errors as data

Declare every error your API can return in one place. Get stable
machine-readable codes, correct HTTP responses, panic-safe handlers,
RFC 9457 problem+json, gRPC code mapping, structured logging and
**generated documentation** — all from a single source of truth.
Zero dependencies.

> Your error documentation can never go stale: it is generated from the
> same data your API runs on.

## Installing

```sh
go get github.com/junkd0g/neji/v2
```

## Quick start

```go
package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	nerror "github.com/junkd0g/neji/v2"
)

// Every error this API can return, declared once.
var Errors = nerror.Catalog{
	"user_not_found":  {Status: 404, Message: "user does not exist"},
	"quota_exceeded":  {Status: 429, Message: "quota exceeded", RetryAfter: time.Minute},
	"invalid_payload": {Status: 422, Message: "validation failed"},
}

func main() {
	// Log every error response. The correlation ID in the log line
	// matches the one the client received.
	nerror.OnWrite = func(e *nerror.Error) {
		slog.Error("api error", "error", e)
	}
	// nerror.Debug = true // dev only: responses include cause + stack trace

	mux := http.NewServeMux()

	// Handlers return errors; neji writes the response and recovers panics.
	mux.Handle("GET /users/{id}", nerror.Handler(func(w http.ResponseWriter, r *http.Request) error {
		user, err := findUser(r.PathValue("id"))
		if err != nil {
			return Errors.Wrap("user_not_found", err) // cause stays server-side
		}
		return json.NewEncoder(w).Encode(user)
	}))

	// The API serves its own, always-current error documentation.
	mux.Handle("GET /errors", Errors.DocsHandler())

	http.ListenAndServe(":8076", mux)
}
```

```sh
$ curl -i localhost:8076/users/42
HTTP/1.1 404 Not Found
Content-Type: application/json; charset=utf-8

{"error":{"code":"user_not_found","status":404,"message":"user does not exist","correlation_id":"f3b1c2d4e5a6b7c8"}}
```

The wrapped cause (the database error) was logged with the same
correlation ID — and never sent to the client. The ID is also in the
`X-Correlation-ID` response header, and if the request carried an
`X-Request-ID`, that is the ID you get back.

## The catalog

Construct every error through the catalog so codes stay stable:

```go
Errors.New("user_not_found")                                  // the declared error
Errors.Newf("user_not_found", "user %q does not exist", id)   // override the message
Errors.New("invalid_payload").With("field", "email")          // add details
Errors.Wrap("user_not_found", err)                            // attach a cause
```

Match by code, anywhere in the chain:

```go
if Errors.Is(err, "user_not_found") { ... }              // allocation-free
if errors.Is(err, Errors.New("user_not_found")) { ... }  // also works

nerror.Code(err)   // "user_not_found" — what the client would see
nerror.Status(err) // 404
```

`nerror.Code` and `nerror.Status` mirror `Write`: a plain Go error
reports `"internal"` and `500`, nil reports `""` and `0`.

Validation failures carry their fields as data:

```go
return Errors.New("invalid_payload").
	WithField("email", "must be a valid address").
	WithField("age", "must be positive")
```

```json
{"error":{"code":"invalid_payload","status":422,"message":"validation failed",
  "details":{"fields":[{"field":"email","message":"must be a valid address"},
                       {"field":"age","message":"must be positive"}]}}}
```

`nerror.Fields(err)` reads them back, on the server or in a Go client
after `Parse`.

Catch catalog mistakes in CI:

```go
func TestCatalog(t *testing.T) {
	if err := Errors.Validate(); err != nil {
		t.Fatal(err)
	}
}
```

## Generated docs

```go
fmt.Println(Errors.Markdown())
```

| Code | HTTP Status | gRPC Code | Retryable | Message |
|------|-------------|-----------|-----------|---------|
| `invalid_payload` | 422 Unprocessable Entity | `INVALID_ARGUMENT` | no | validation failed |
| `quota_exceeded` | 429 Too Many Requests | `RESOURCE_EXHAUSTED` | after 1m0s | quota exceeded |
| `user_not_found` | 404 Not Found | `NOT_FOUND` | no | user does not exist |

`Errors.OpenAPI()` emits an OpenAPI 3.1 components fragment: a shared
`Error` schema (with the code enum, `correlation_id` and validation
`fields`) plus one reusable response per code.

And the API serves its own docs, generated live on every request:

| Request | Response |
|---------|----------|
| `GET /errors` | JSON listing of every error code |
| `GET /errors?format=md` | the Markdown table above |
| `GET /errors?format=openapi` | the OpenAPI components fragment |

## Go clients: same codes across the wire

`nerror.Parse` is the inverse of `Write` — clients get the server's
catalog codes back as `*nerror.Error`:

```go
resp, _ := client.Do(req)
defer resp.Body.Close()
if err := nerror.Parse(resp); err != nil {
	if errors.Is(err, api.Errors.New("quota_exceeded")) {
		backoff() // RetryAfter is read from the Retry-After header
	}
	return err
}
```

It understands both wire formats neji produces, reads `Retry-After` as
either seconds or an HTTP-date, and a non-JSON body (a bare nginx 502,
say) still yields a `*Error` with the status and code `http_error`.

## Debug mode and correlation IDs

In production (the default), responses never contain causes or stack
traces — only a stamped `correlation_id` that also appears in your logs
via `OnWrite`. A client reports an ID, you grep your logs, the full cause
chain is there.

In development:

```go
nerror.Debug = true // at startup, never in prod
```

```json
{
	"error": {
		"code": "user_not_found",
		"status": 404,
		"message": "user does not exist",
		"correlation_id": "f3b1c2d4e5a6b7c8",
		"cause": "sql: no rows in result set",
		"stack": ["main.GetUser /app/users.go:42", "..."]
	}
}
```

Already have a request or trace ID? `nerror.Handler` (and `WriteFor`)
reuse an incoming `X-Request-ID` header automatically, so the same ID
flows from the client through your proxy into your logs. To set one by
hand, `err.WithCorrelationID(traceID)` always wins. The header names are
`nerror.RequestIDHeader` and `nerror.CorrelationIDHeader`; set either to
`""` to turn that side off.

To log the request alongside the error, use the request-aware hook:

```go
nerror.OnWriteRequest = func(e *nerror.Error, r *http.Request) {
	slog.Error("api error", "method", r.Method, "path", r.URL.Path, "error", e)
}
```

`Handler`, `WriteFor` and `WriteProblemFor` call it; plain `Write` has no
request and calls `OnWrite`. Set both and each error is reported once.

## One error, every protocol

**RFC 9457 problem+json** — same catalog, standard format:

```go
nerror.WriteProblem(w, Errors.New("quota_exceeded"))
// Content-Type: application/problem+json
// {"type":"about:blank","title":"Too Many Requests","status":429,
//  "detail":"quota exceeded","code":"quota_exceeded","correlation_id":"..."}
```

**gRPC** — codes mirror `google.golang.org/grpc/codes`, without the
dependency:

```go
st := status.New(codes.Code(err.GRPCCode()), err.Message)
```

**Structured logging** — `*nerror.Error` implements `slog.LogValuer`:

```go
slog.Error("request failed", "error", err)
// "error":{"code":"quota_exceeded","status":429,"message":"quota exceeded",
//          "correlation_id":"f3b1c2d4e5a6b7c8","cause":"redis counter at limit"}
```

## Ad-hoc errors

For errors that aren't part of your API contract:

```go
nerror.NotFound("nothing here")
nerror.BadRequest("unreadable body")
nerror.Internal("oops").Wrap(err)
```

(`BadRequest`, `Unauthorized`, `Forbidden`, `NotFound`, `Conflict`,
`UnprocessableEntity`, `TooManyRequests`, `Internal`, `Unavailable`, or
`nerror.New(status, code, message)` for anything else.)

## Other routers

`nerror.Handler` is a plain `http.Handler`, so it drops straight into
chi, gorilla/mux or the standard mux. For echo and gin, which have their
own handler shapes, see [docs/adapters.md](docs/adapters.md): a few lines
each, tested against the real routers, no extra dependency in neji.

## Migrating from v1

| v1 | v2 |
|----|----|
| `SimpleErrorResponseWithStatus(500, err)` + manual headers/`Write` | `nerror.Write(w, nerror.Internal("...").Wrap(err))` |
| `SimpleErrorResponseWithCodeV2(500, err)` | same — the nested `{"error": {...}}` shape is the v2 default |
| `WrapError(err, "context")` | `fmt.Errorf("context: %w", err)` or `Error.Wrap` |
| `ErrInvalidParameter("user_id")` | `nerror.BadRequest("missing parameter user_id")` or a catalog code |

**v1 is frozen** — it stays installable as `github.com/junkd0g/neji` and
its documentation lives at [docs/v1.md](docs/v1.md), but new development
happens here.

## Running tests

```sh
cd v2 && go test ./...
```

## License

This project is licensed under the MIT License. See the LICENSE file for details.

## Authors

* **Iordanis Paschalidis** - [junkd0g](https://github.com/junkd0g)
