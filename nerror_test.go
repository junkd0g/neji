package nerror_test

import (
	"testing"

	nerror "github.com/junkd0g/neji"
	"github.com/stretchr/testify/assert"
)

func Test_ErrInvalidParameter(t *testing.T) {
	t.Run("Create successfully a Common object with optionalTemplatePath", func(t *testing.T) {
		errMessage := nerror.ErrInvalidParameter("path")
		assert.Equal(t, errMessage.Error(), "missing parameter path")
	})

}
