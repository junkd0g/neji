package nerror_test

import (
	"errors"
	"testing"

	nerror "github.com/junkd0g/neji"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrInvalidParameter(t *testing.T) {
	tests := []struct {
		name          string
		parameter     string
		expectedError string
	}{
		{
			name:          "standard parameter name",
			parameter:     "path",
			expectedError: "missing parameter path",
		},
		{
			name:          "parameter with underscore",
			parameter:     "user_id",
			expectedError: "missing parameter user_id",
		},
		{
			name:          "empty parameter name",
			parameter:     "",
			expectedError: "missing parameter ",
		},
		{
			name:          "parameter with special characters",
			parameter:     "field-name",
			expectedError: "missing parameter field-name",
		},
		{
			name:          "parameter with spaces",
			parameter:     "field name",
			expectedError: "missing parameter field name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := nerror.ErrInvalidParameter(tt.parameter)
			require.Error(t, err)
			assert.Equal(t, tt.expectedError, err.Error())
		})
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name          string
		originalError error
		wrapMessage   string
		expectedError string
	}{
		{
			name:          "wrap standard error",
			originalError: errors.New("connection timeout"),
			wrapMessage:   "failed to fetch data",
			expectedError: "failed to fetch data: connection timeout",
		},
		{
			name:          "wrap ErrInvalidParameter",
			originalError: nerror.ErrInvalidParameter("path"),
			wrapMessage:   "validation error",
			expectedError: "validation error: missing parameter path",
		},
		{
			name:          "wrap with empty message",
			originalError: errors.New("original error"),
			wrapMessage:   "",
			expectedError: ": original error",
		},
		{
			name:          "wrap empty error",
			originalError: errors.New(""),
			wrapMessage:   "context message",
			expectedError: "context message: ",
		},
		{
			name:          "chain multiple wraps",
			originalError: nerror.WrapError(errors.New("root cause"), "first wrap"),
			wrapMessage:   "second wrap",
			expectedError: "second wrap: first wrap: root cause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := nerror.WrapError(tt.originalError, tt.wrapMessage)
			require.Error(t, err)
			assert.Equal(t, tt.expectedError, err.Error())

			// Test that the original error is preserved via errors.Is
			assert.True(t, errors.Is(err, tt.originalError))
		})
	}
}
