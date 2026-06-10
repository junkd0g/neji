package nerror_test

import (
	"errors"
	"testing"

	nerror "github.com/junkd0g/neji"
	"github.com/stretchr/testify/assert"
)

func Test_ErrInvalidParameter(t *testing.T) {
	t.Run("creates missing parameter error", func(t *testing.T) {
		errMessage := nerror.ErrInvalidParameter("path")
		assert.Equal(t, "missing parameter path", errMessage.Error())
	})

}

func Test_WrapError(t *testing.T) {
	t.Run("wraps error with context", func(t *testing.T) {
		baseErr := nerror.ErrInvalidParameter("path")
		errMessage := nerror.WrapError(baseErr, "error")

		assert.Equal(t, "error: missing parameter path", errMessage.Error())
		assert.True(t, errors.Is(errMessage, baseErr))
	})

	t.Run("returns nil when original error is nil", func(t *testing.T) {
		assert.NoError(t, nerror.WrapError(nil, "error"))
	})

}
