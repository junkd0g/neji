package nerror

import "encoding/json"

// ErrorMessage returns json:
/*
	{
		"message" : "Your json is wrong or something",
		"code" : 500
	}
|*/
type ErrorMessage struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

//SimpeErrorResponseWithCode returns ErrorMessage struct as an []bytes
/*
	return json response in this format:

	{
		"message" : "Your json is wrong or something",
		"code" : 500
	}

	@return array of bytes of the json object
	@return int http code status
*/
func SimpeErrorResponseWithCode(code int, err error) ([]byte, error) {
	JSONBody, errMarshal := json.Marshal(ErrorMessage{Message: err.Error(), Code: code})
	return JSONBody, errMarshal
}
