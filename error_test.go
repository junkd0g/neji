package nerror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleErrorResponseWithStatus(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		inputError   error
		expectedJSON string
		expectError  bool
	}{
		{
			name:         "standard error with 500 status",
			status:       500,
			inputError:   errors.New("math: square root of negative number"),
			expectedJSON: `{"message":"math: square root of negative number","status":500}`,
			expectError:  false,
		},
		{
			name:         "standard error with 400 status",
			status:       400,
			inputError:   errors.New("bad request"),
			expectedJSON: `{"message":"bad request","status":400}`,
			expectError:  false,
		},
		{
			name:         "empty error message",
			status:       500,
			inputError:   errors.New(""),
			expectedJSON: `{"message":"","status":500}`,
			expectError:  false,
		},
		{
			name:         "error with special characters",
			status:       422,
			inputError:   errors.New("invalid JSON: missing \"field\""),
			expectedJSON: `{"message":"invalid JSON: missing \"field\"","status":422}`,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SimpleErrorResponseWithStatus(tt.status, tt.inputError)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.JSONEq(t, tt.expectedJSON, string(result))
		})
	}
}

func TestSimpleErrorResponseWithCodeV2(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		inputError   error
		expectedJSON string
		expectError  bool
	}{
		{
			name:         "standard error with 500 status",
			status:       500,
			inputError:   errors.New("math: square root of negative number"),
			expectedJSON: `{"error":{"message":"math: square root of negative number","status":500}}`,
			expectError:  false,
		},
		{
			name:         "standard error with 502 status",
			status:       502,
			inputError:   errors.New("bad gateway"),
			expectedJSON: `{"error":{"message":"bad gateway","status":502}}`,
			expectError:  false,
		},
		{
			name:         "empty error message",
			status:       500,
			inputError:   errors.New(""),
			expectedJSON: `{"error":{"message":"","status":500}}`,
			expectError:  false,
		},
		{
			name:         "error with special characters",
			status:       422,
			inputError:   errors.New("invalid JSON: missing \"field\""),
			expectedJSON: `{"error":{"message":"invalid JSON: missing \"field\"","status":422}}`,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SimpleErrorResponseWithCodeV2(tt.status, tt.inputError)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.JSONEq(t, tt.expectedJSON, string(result))
		})
	}
}
