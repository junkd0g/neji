package nerror

import "fmt"

const (
	missingParameterText = "missing parameter %s"
)

// ErrInvalidParameter returns an error indicating a missing or invalid parameter.
//
// This function helps standardize parameter validation errors by formatting a
// descriptive message with the provided parameter name.
//
// Example usage:
//
//	err := ErrInvalidParameter("user_id")
//	fmt.Println(err) // Output: "missing parameter user_id"
//
// The message parameter should contain the name of the missing or invalid parameter.
// Returns a formatted error with the parameter name included in the message.
func ErrInvalidParameter(message string) error {
	return fmt.Errorf(missingParameterText, message)
}

// WrapError wraps an existing error with additional context, preserving the original error.
//
// This function is useful for adding meaningful context to errors before propagating them,
// aiding in debugging and error analysis.
//
// Example usage:
//
//	err := errors.New("connection timeout")
//	wrappedErr := WrapError(err, "failed to fetch data")
//	fmt.Println(wrappedErr) // Output: "failed to fetch data: connection timeout"
//
// The err parameter is the original error to wrap, and message provides additional
// context. Returns a new error that includes both the context message and the
// original error, maintaining the error chain for use with errors.Is and errors.As.
func WrapError(err error, message string) error {
	return fmt.Errorf("%s: %w", message, err)
}
