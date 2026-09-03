# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

All changes are additive; v2.0.0 code keeps compiling and behaving the
same unless noted under Changed.

### Added
- `X-Correlation-ID` response header carrying the same ID as the JSON
  body. Rename or disable it via `nerror.CorrelationIDHeader`.
- `Handler`, `WriteFor` and `WriteProblemFor` reuse an incoming
  `X-Request-ID` (fallback `X-Correlation-ID`) as the correlation ID, so
  one ID flows end to end. Configurable via `nerror.RequestIDHeader`;
  unsafe or over-long values are ignored.
- `OnWriteRequest func(*Error, *http.Request)`: request-aware logging and
  metrics hook. Preferred over `OnWrite` for request-aware writes, so
  setting both never double-reports.
- `WriteFor` and `WriteProblemFor`: `Write`/`WriteProblem` with the
  request in scope, for handlers of any shape.
- `nerror.Code(err)`, `nerror.Status(err)`, `nerror.HasCode(err, code)`
  and `Catalog.Is(err, code)`: allocation-free matching and inspection.
- `FieldError`, `Error.WithField`, `Error.WithFields` and
  `nerror.Fields(err)`: validation failures as structured
  `details.fields`, readable on both sides of the wire.
- `Parse` accepts `Retry-After` as an HTTP-date as well as seconds.
- `OpenAPI` describes `correlation_id` and `details.fields`, and emits a
  `FieldError` schema.
- `docs/adapters.md`: tested recipes for chi, echo and gin.
- `Handler`'s writer implements `http.Flusher` and `Unwrap`, so
  `http.ResponseController` reaches the original writer.

### Changed
- `Handler` no longer appends an error body to a response that has
  already started. A late error is only reported to the write hooks; a
  late panic is re-raised so `net/http` logs it and closes the
  connection. Previously this produced a "superfluous WriteHeader" log
  and a corrupt body.
- `Catalog.New` with an unknown code now captures a stack trace in debug
  mode, pointing at the typo.
- `newCorrelationID` falls back to `math/rand/v2` should `crypto/rand`
  fail on Go 1.23 (it cannot fail on 1.24+).
- `v2/README.md` is now a short pointer to the repository README instead
  of a diverging copy.

## [2.0.0] - 2026-06-10

### Added
- `github.com/junkd0g/neji/v2`: catalog-based error API. Declare every
  error once in a `Catalog`; get `Write` (JSON envelope), `WriteProblem`
  (RFC 9457 problem+json), `Handler` with panic recovery, gRPC code
  mapping, `slog.LogValuer`, correlation IDs, debug mode with cause and
  stack, `Parse` for Go clients, and generated Markdown / OpenAPI /
  `DocsHandler` documentation. Zero dependencies.
- CI matrix testing on Go 1.23, 1.24, and 1.25 for both modules.
- `gofmt`, `go vet`, and `-race` checks in CI.
- `govulncheck` security scan job.
- `CHANGELOG.md` following Keep a Changelog format.

### Changed
- v1 hardened against nil errors and frozen; new development happens in v2.
- CI workflow renamed from `build.yaml` to `ci.yml`.
- CI actions upgraded: `setup-go@v4` → `v5`, `golangci-lint-action@v3` → `v8`;
  `errcheck` findings fixed.
- Dropped manual `actions/cache@v3` step — `setup-go@v5` caches modules implicitly.
- Dropped 100%-coverage gate in CI in favour of printing coverage; coverage gating belongs in review.
- README rewritten around the v2 API; v1 documentation moved to `docs/v1.md`.
- `.gitignore` normalised to match other `junkd0g/*` repos.
