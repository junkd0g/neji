package nerror

import (
	"errors"
	"testing"
)

func TestSimpleErrorResponseWithStatus(t *testing.T) {
	err1 := errors.New("math: square root of negative number")
	statsErrJSONBody, err := SimpleErrorResponseWithStatus(500, err1)
	if err != nil {
		t.Fatal(err.Error())
	}

	errorJSON := `{"message":"math: square root of negative number","status":500}`

	if string(statsErrJSONBody) != errorJSON {
		t.Fatalf("Responses are not equal %s with %s", string(statsErrJSONBody), errorJSON)
	}

}

func TestSimpleErrorResponseWithStatusNilError(t *testing.T) {
	statsErrJSONBody, err := SimpleErrorResponseWithStatus(500, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	errorJSON := `{"message":"","status":500}`

	if string(statsErrJSONBody) != errorJSON {
		t.Fatalf("Responses are not equal %s with %s", string(statsErrJSONBody), errorJSON)
	}
}

func TestSimpleErrorResponseWithCodeV2(t *testing.T) {
	err1 := errors.New("math: square root of negative number")
	statsErrJSONBody, err := SimpleErrorResponseWithCodeV2(500, err1)
	if err != nil {
		t.Fatal(err.Error())
	}

	errorJSON := `{"error":{"message":"math: square root of negative number","status":500}}`

	if string(statsErrJSONBody) != errorJSON {
		t.Fatalf("Responses are not equal %s with %s", string(statsErrJSONBody), errorJSON)
	}

}

func TestSimpleErrorResponseWithCodeV2NilError(t *testing.T) {
	statsErrJSONBody, err := SimpleErrorResponseWithCodeV2(500, nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	errorJSON := `{"error":{"message":"","status":500}}`

	if string(statsErrJSONBody) != errorJSON {
		t.Fatalf("Responses are not equal %s with %s", string(statsErrJSONBody), errorJSON)
	}
}
