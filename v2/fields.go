package nerror

import "errors"

// FieldError describes one invalid input field, e.g. which request field
// failed validation and why. Attach them to an Error with WithField or
// WithFields; they are sent to clients under details.fields:
//
//	{
//	    "error": {
//	        "code": "invalid_payload",
//	        "status": 422,
//	        "message": "validation failed",
//	        "details": {
//	            "fields": [
//	                {"field": "email", "message": "must be a valid address"},
//	                {"field": "age", "message": "must be positive"}
//	            ]
//	        }
//	    }
//	}
type FieldError struct {
	// Field names the invalid input, e.g. "email" or "address.zip".
	Field string `json:"field"`

	// Message says what is wrong with it, in words safe for API clients.
	Message string `json:"message"`
}

// fieldsKey is the Details key under which field errors are stored.
const fieldsKey = "fields"

// WithField returns a copy of e with one field error appended to
// details.fields.
func (e *Error) WithField(field, message string) *Error {
	return e.WithFields(FieldError{Field: field, Message: message})
}

// WithFields returns a copy of e with the given field errors appended to
// details.fields. Calling it with no fields returns an unchanged copy.
func (e *Error) WithFields(fields ...FieldError) *Error {
	c := e.clone()
	if len(fields) == 0 {
		return c
	}
	if c.Details == nil {
		c.Details = make(map[string]any, 1)
	}
	existing := Fields(c)
	merged := make([]FieldError, 0, len(existing)+len(fields))
	merged = append(merged, existing...)
	merged = append(merged, fields...)
	c.Details[fieldsKey] = merged
	return c
}

// Fields returns the field errors attached to the first *Error in err's
// chain, or nil when there are none. It understands both errors built
// with WithField / WithFields and errors decoded from the wire by Parse,
// so servers and Go clients can inspect them the same way:
//
//	for _, f := range nerror.Fields(err) {
//	    fmt.Printf("%s: %s\n", f.Field, f.Message)
//	}
func Fields(err error) []FieldError {
	var e *Error
	if !errors.As(err, &e) || e.Details == nil {
		return nil
	}
	switch v := e.Details[fieldsKey].(type) {
	case []FieldError:
		return v
	case []any:
		out := make([]FieldError, 0, len(v))
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			field, _ := m["field"].(string)
			message, _ := m["message"].(string)
			out = append(out, FieldError{Field: field, Message: message})
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}
	return nil
}
