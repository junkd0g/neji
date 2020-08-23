# neji

Package junkd0g/neji is a simple JSON error response library.

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
	errorResponse, _ := nerror.SimpeErrorResponseWithCode(500, err)

	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte(errorResponse))
}

func main() {

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", HelloWorld)

	http.ListenAndServe(":8076", router)
}

```

## Authors

* **Iordanis Paschalidis** -[junkd0g](https://github.com/junkd0g)