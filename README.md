[![Go Report Card](https://goreportcard.com/badge/github.com/junkd0g/neji)](https://goreportcard.com/report/github.com/junkd0g/neji)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://pkg.go.dev/badge/github.com/junkd0g/neji.svg)](https://pkg.go.dev/github.com/junkd0g/neji)

# neji

**`neji`** is a lightweight Go package for generating structured **JSON error responses** and handling parameter validation errors. It provides utility functions to standardize error handling in API applications.

## Features

- 🔧 **Structured JSON Error Responses**: Generate consistent error responses in two formats
- ✅ **Parameter Validation**: Standardized missing parameter error messages  
- 🔗 **Error Wrapping**: Preserve original errors while adding context
- 🧪 **Well Tested**: Comprehensive test coverage with edge cases
- 📦 **Zero Dependencies**: Only standard library (except for tests)

## Installing

```bash
go get -u github.com/junkd0g/neji
```

## ✅ Running Tests

```bash
go test ./...
```

## 🚀 Usage

### Basic Error Response

```go
package main

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	nerror "github.com/junkd0g/neji"
)

func HelloWorld(w http.ResponseWriter, r *http.Request) {
	err := errors.New("This is the best error message ever")
	errorResponse, _ := nerror.SimpleErrorResponseWithStatus(500, err)

	w.Header().Set("Content-Type", "application/json")
	w.Write(errorResponse)
}

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", HelloWorld)
	http.ListenAndServe(":8076", router)
}
```

### API Reference

#### SimpleErrorResponseWithStatus

```go
nerror.SimpleErrorResponseWithStatus(500, err)
```

Returns:
```json
{
	"message": "Your json is wrong or something",
	"status": 500
}
```

#### SimpleErrorResponseWithCodeV2

```go
nerror.SimpleErrorResponseWithCodeV2(500, err)
```

Returns:
```json
{
	"error": {
		"status": 500,
		"message": "Your json is wrong or something"
	}
}
```

#### Parameter Validation

```go
// Generate standardized parameter error
err := nerror.ErrInvalidParameter("user_id")
// Returns: "missing parameter user_id"
```

#### Error Wrapping

```go
// Wrap errors with additional context
originalErr := errors.New("connection timeout")
wrappedErr := nerror.WrapError(originalErr, "failed to fetch data")
// Returns: "failed to fetch data: connection timeout"
```
## 📝 License

This project is licensed under the MIT License. See the LICENSE file for details.

## Authors

* **Iordanis Paschalidis** -[junkd0g](https://github.com/junkd0g)