[![Go Report Card](https://goreportcard.com/badge/github.com/junkd0g/neji)](https://goreportcard.com/report/github.com/junkd0g/neji)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://pkg.go.dev/badge/github.com/your-repo/neji.svg)](https://pkg.go.dev/github.com/junkd0g/neji)

# neji

**`neji`** is a lightweight Go package for generating structured **JSON error responses**.  
It provides utility functions to standardize error handling 

## Installing

go get -u github.com/junkd0g/neji

## Running the tests

go test ./...

## Example

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
	errorResponse, _ := nerror.SimpeErrorResponseWithStatus(500, err)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte(errorResponse))
}

func main() {

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", HelloWorld)

	http.ListenAndServe(":8076", router)
}

```

```go

nerror.SimpeErrorResponseWithStatus(500, err)

/*
	will return this response

	{
		"message" : "Your json is wrong or something",
		"status" : 500
	}
*/

nerror.SimpeErrorResponseWithStatusV2(500, err)

/*
	will return this response

	{
		"error" : {
			"status" : 502,
			"message" : "Bad gateway."
  		}
	}
*/

```

## Authors

* **Iordanis Paschalidis** -[junkd0g](https://github.com/junkd0g)