package httpclient

import (
	"fmt"
	"net/http"
)

// Constants for error management
const (
	MsgErrCannotObtain              = "cannotObtain"
	MsgErrCannotUnmarshal           = "cannotUnmarshal"
	MsgErrCannotParse               = "cannotParse"
	MsgErrUnkownHTTPContentType     = "unkownHTTPContentType"
	MsgErrUnknownResponseStatusCode = "unknownResponseStatusCode"

	PrmTokenProviderURL = "tokenProviderURL"
	PrmAPIURL           = "APIURL"
	PrmTokenMsg         = "token"
	PrmResponse         = "response"
)

// HTTPError is returned when an error occured while contacting the keycloak instance.
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("%d:%s", e.StatusCode, e.Message)
}

// IsSuccess is true when HTTP status code is 2xx or 3xx
func (e HTTPError) IsSuccess() bool {
	return e.StatusCode < http.StatusBadRequest
}

// IsError is true when HTTP request failed
func (e HTTPError) IsError() bool {
	return e.StatusCode >= http.StatusBadRequest
}

// IsErrorFromClient is true when HTTP request failed and the cause was related to the client request
func (e HTTPError) IsErrorFromClient() bool {
	return e.StatusCode >= http.StatusBadRequest && e.StatusCode < http.StatusInternalServerError
}

// IsErrorFromServer is true when HTTP request failed and the cause was related to the HTTP server
func (e HTTPError) IsErrorFromServer() bool {
	return e.StatusCode >= http.StatusInternalServerError
}

// Status returns the HTTP status code
func (e HTTPError) Status() int {
	return e.StatusCode
}

// ErrorMessage returns the HTTP error message
func (e HTTPError) ErrorMessage() string {
	return e.Message
}
