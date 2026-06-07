# neji

[![Go Report Card](https://goreportcard.com/badge/github.com/junkd0g/neji)](https://goreportcard.com/report/github.com/junkd0g/neji)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://pkg.go.dev/badge/github.com/junkd0g/neji.svg)](https://pkg.go.dev/github.com/junkd0g/neji)

`neji` is a small Go package for generating structured JSON error responses
and standardising parameter-validation error messages. The import path is
`github.com/junkd0g/neji`; the package name is `nerror`.

## Why

Every Go HTTP service ends up reinventing the same two error envelopes — a
flat `{message,status}` one and a nested `{error:{message,status}}` one. `neji`
gives you both, plus tiny helpers for "missing parameter X" and `fmt.Errorf`
wrapping, without dragging in a framework.

## Features

- Two structured JSON error response shapes (flat and nested).
- Standardised "missing parameter X" message via `ErrInvalidParameter`.
- `WrapError` for `errors.Is` / `errors.As`-compatible wrapping.
- Zero runtime dependencies (`testify` for tests only).

## Requirements

- Go 1.23 or newer.

## Installation

```sh
go get -u github.com/junkd0g/neji
```

## Quick start

```go
package main

import (
    "errors"
    "net/http"

    nerror "github.com/junkd0g/neji"
)

func HelloWorld(w http.ResponseWriter, r *http.Request) {
    err := errors.New("this is the best error message ever")
    body, _ := nerror.SimpleErrorResponseWithStatus(http.StatusInternalServerError, err)

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusInternalServerError)
    w.Write(body)
}

func main() {
    http.HandleFunc("/", HelloWorld)
    http.ListenAndServe(":8076", nil)
}
```

## API Reference

### Types

```go
type SimpleErrorMessage struct {
    Message string `json:"message"`
    Status  int    `json:"status"`
}

type SimpleErrorMessageV2 struct {
    ErrorST SimpleErrorMessage `json:"error"`
}
```

### Functions

```go
func SimpleErrorResponseWithStatus(status int, err error) ([]byte, error)
func SimpleErrorResponseWithCodeV2(status int, err error) ([]byte, error)
func ErrInvalidParameter(message string) error
func WrapError(err error, message string) error
```

#### `SimpleErrorResponseWithStatus`

Returns the flat envelope:

```json
{
    "message": "your JSON is wrong or something",
    "status": 500
}
```

#### `SimpleErrorResponseWithCodeV2`

Returns the nested envelope:

```json
{
    "error": {
        "status": 500,
        "message": "your JSON is wrong or something"
    }
}
```

#### `ErrInvalidParameter`

```go
err := nerror.ErrInvalidParameter("user_id")
// err.Error() == "missing parameter user_id"
```

#### `WrapError`

```go
original := errors.New("connection timeout")
wrapped := nerror.WrapError(original, "failed to fetch data")
// wrapped.Error() == "failed to fetch data: connection timeout"
// errors.Is(wrapped, original) == true
```

## Testing

```sh
go test ./...
```

With coverage:

```sh
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Contributing

1. Fork the repository.
2. Create a feature branch (`git checkout -b feature/your-feature`).
3. Commit your changes.
4. Push to the branch (`git push origin feature/your-feature`).
5. Open a pull request.

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE)
file for details.

## Author

Iordanis Paschalidis — [@junkd0g](https://github.com/junkd0g)
