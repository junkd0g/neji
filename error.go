// Package nerror provides utility functions and structures for handling
// structured JSON error responses in API applications.
//
// It defines error response formats commonly used in APIs, including
// simple error messages and nested error structures. The package also
// offers helper functions for serializing error messages into JSON format.
package nerror

import "encoding/json"

// SimpleErrorMessage represents a basic JSON error response format.
//
// Example JSON output:
//
//	{
//	    "message": "Your JSON is wrong or something",
//	    "status": 500
//	}
type SimpleErrorMessage struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

// SimpleErrorMessageV2 represents a more structured JSON error response
// format, encapsulating the error message within an "error" object.
//
// Example JSON output:
//
//	{
//	    "error": {
//	        "status": 502,
//	        "message": "Bad gateway."
//	    }
//	}
type SimpleErrorMessageV2 struct {
	ErrorST SimpleErrorMessage `json:"error"`
}

// SimpleErrorResponseWithStatus creates a JSON-formatted error response
// using the SimpleErrorMessage structure.
//
// It returns a JSON response in the format:
//
//	{
//	    "message": "Your JSON is wrong or something",
//	    "status": 500
//	}
//
// The status parameter specifies the HTTP status code, and err provides
// the error message. Returns the JSON-encoded error response as []byte
// and any marshaling error that may occur.
func SimpleErrorResponseWithStatus(status int, err error) ([]byte, error) {
	message := ""
	if err != nil {
		message = err.Error()
	}

	JSONBody, errMarshal := json.Marshal(
		SimpleErrorMessage{
			Message: message,
			Status:  status,
		},
	)

	return JSONBody, errMarshal
}

// SimpleErrorResponseWithCodeV2 creates a JSON-formatted error response
// using the SimpleErrorMessageV2 structure.
//
// It returns a JSON response in the format:
//
//	{
//	    "error": {
//	        "status": 502,
//	        "message": "Bad gateway."
//	    }
//	}
//
// The status parameter specifies the HTTP status code, and err provides
// the error message. Returns the JSON-encoded error response as []byte
// and any marshaling error that may occur.
func SimpleErrorResponseWithCodeV2(status int, err error) ([]byte, error) {
	message := ""
	if err != nil {
		message = err.Error()
	}

	JSONBody, errMarshal := json.Marshal(
		SimpleErrorMessageV2{
			ErrorST: SimpleErrorMessage{
				Message: message,
				Status:  status,
			},
		})
	return JSONBody, errMarshal
}
