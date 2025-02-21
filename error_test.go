package nerror

import (
	"errors"
	"testing"
)

func TestSimpeErrorResponseWithCode(t *testing.T) {
	err1 := errors.New("math: square root of negative number")
	statsErrJSONBody, err := SimpleErrorResponseWithStatus(500, err1)
	if err != nil {
		t.Fatalf(err.Error())
	}

	errorJSON := `{"message":"math: square root of negative number","status":500}`

	if string(statsErrJSONBody) != errorJSON {
		t.Fatalf("Responses are not equal %s with %s", string(statsErrJSONBody), errorJSON)
	}

}

func TestSimpeErrorResponseWithCodeV2(t *testing.T) {
	err1 := errors.New("math: square root of negative number")
	statsErrJSONBody, err := SimpleErrorResponseWithCodeV2(500, err1)
	if err != nil {
		t.Fatalf(err.Error())
	}

	errorJSON := `{"error":{"message":"math: square root of negative number","status":500}}`

	if string(statsErrJSONBody) != errorJSON {
		t.Fatalf("Responses are not equal %s with %s", string(statsErrJSONBody), errorJSON)
	}

}
