package nerror

import (
	"errors"
	"net/http"
)

// Code returns the machine-readable code Write would send for err: the
// Code of the first *Error in err's chain, "internal" for any other
// non-nil error, and "" for nil.
func Code(err error) string {
	if err == nil {
		return ""
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return "internal"
}

// Status returns the HTTP status Write would send for err: the Status of
// the first *Error in err's chain, 500 for any other non-nil error, and 0
// for nil.
func Status(err error) int {
	if err == nil {
		return 0
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Status
	}
	return http.StatusInternalServerError
}

// HasCode reports whether any *Error in err's chain carries code. It is
// the allocation-free equivalent of errors.Is(err, catalog.New(code)).
func HasCode(err error, code string) bool {
	for err != nil {
		if e, ok := err.(*Error); ok && e.Code == code {
			return true
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
		case interface{ Unwrap() []error }:
			for _, inner := range x.Unwrap() {
				if HasCode(inner, code) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
	return false
}

// Is reports whether err is, or wraps, the catalog error declared under
// code:
//
//	if Errors.Is(err, "user_not_found") { ... }
//
// It matches by code alone, exactly like errors.Is(err, Errors.New(code)),
// but without constructing a throwaway error.
func (c Catalog) Is(err error, code string) bool {
	return HasCode(err, code)
}
