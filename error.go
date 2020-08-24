package nerror

import "encoding/json"

// SimpleErrorMessage returns json:
/*
	{
		"message" : "Your json is wrong or something",
		"code" : 500
	}
|*/
type SimpleErrorMessage struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

// SimpleErrorMessageV2 returns json:
/*
	{
		"error" : {
			"status" : 502,
    		"message" : "Bad gateway."
  		}
	}
|*/
type SimpleErrorMessageV2 struct {
	ErrorST SimpleErrorMessage `json:"error"`
}

//SimpeErrorResponseWithStatus returns SimpleErrorMessage struct as an []bytes
/*
	return json response in this format:

	{
		"message" : "Your json is wrong or something",
		"status" : 500
	}

	@return array of bytes of the json object
	@return int http code status
*/
func SimpeErrorResponseWithStatus(status int, err error) ([]byte, error) {
	JSONBody, errMarshal := json.Marshal(
		SimpleErrorMessage{
			Message: err.Error(),
			Status:  status,
		},
	)

	return JSONBody, errMarshal
}

//SimpeErrorResponseWithCodeV2 returns SimpleErrorMessageV2 struct as an []bytes
/*
	return json response in this format:

	{
		"error" : {
			"status" : 502,
			"message" : "Bad gateway."
  		}
	}

	@return array of bytes of the json object
	@return int http status
*/
func SimpeErrorResponseWithCodeV2(status int, err error) ([]byte, error) {
	JSONBody, errMarshal := json.Marshal(
		SimpleErrorMessageV2{
			ErrorST: SimpleErrorMessage{
				Message: err.Error(),
				Status:  status,
			},
		})
	return JSONBody, errMarshal
}
