package nerror

import "fmt"

const (
	missingParameterText = "missing parameter %s"
)

// ErrInvalidParameter return an error missing paraneter + whatever is in the parameter message
func ErrInvalidParameter(message string) error {
	return fmt.Errorf(missingParameterText, message)
}

// WrapError return an error with the message + the error
func WrapError(err error, message string) error {
	return fmt.Errorf("%s: %w", message, err)
}
