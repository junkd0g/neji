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
// @param message The name of the missing or invalid parameter.
// @return        Formatted error message.
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
// @param err     The original error to wrap.
// @param message Additional context message to include.
// @return        Wrapped error message.
func WrapError(err error, message string) error {
	return fmt.Errorf("%s: %w", message, err)
}
