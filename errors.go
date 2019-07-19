package main

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
)

var (
	ErrWriteCertsFailed = Failure(
		errors.New("Failed to write updated certificates file"),
		http.StatusInternalServerError,
		"",
	)

	ErrMSSGHTokenRequest = Failure(
		errors.New("! TokenRequest had MSSCert and Github user+token"),
		http.StatusBadRequest,
		"Supply an MSSCertificate or a Github login user + OAuth2 token. Not both.",
	)
	ErrIncompleteTokenRequest = Failure(
		errors.New("TokenRequest requires github username + oauth2 token"),
		http.StatusBadRequest,
		"Authentication requires a github username and oauth2 token",
	)

	ErrCreatingToken = func(err error, st int) *RequestFailure {
		return Failure(
			errors.Wrap(err, "jwt creation unauthorized"),
			st,
			"",
		)
	}
)

//contains a error message to be sent out as json
type ErrorMessage struct {
	Code        int    `json:"code"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

//sends an http error (same as http.error) in json format
func SendError(w http.ResponseWriter, s int, description string) {
	errorMes := ErrorMessage{
		Code:        s,
		Status:      http.StatusText(s),
		Description: description,
	}
	w.WriteHeader(s)
	if err := json.NewEncoder(w).Encode(errorMes); err != nil {
		logger.Println("ERROR: Failed to encode error output")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

type RequestFailure struct {
	err  error  // developer level error for logging
	Code int    // http status
	Msg  string // public facing message to send
}

func Failure(err error, statusCode int, userMsg string) *RequestFailure {
	if userMsg == "" {
		userMsg = http.StatusText(statusCode)
	}
	return &RequestFailure{
		err:  err,
		Code: statusCode,
		Msg:  userMsg,
	}
}

func (f RequestFailure) Error() string {
	return fmt.Sprintf("%v - %v", http.StatusText(f.Code), f.Msg)
}

func Fail(w http.ResponseWriter, err error) {
	switch errors.Cause(err).(type) {
	case *json.UnsupportedValueError, *json.UnsupportedTypeError, *json.SyntaxError, *json.UnmarshalTypeError:
		SendError(w, http.StatusBadRequest, "JSON format error")
		logger.Printf("request failed - json format error: %v", err.Error())
	case *RequestFailure:
		rf := err.(*RequestFailure)
		logger.Println("Request failed: ", err.Error())
		SendError(w, rf.Code, rf.Msg)
	case RequestFailure:
		logger.Println("Request failed: ", err.Error())
		rf := err.(RequestFailure)
		SendError(w, rf.Code, rf.Msg)
	default:
		logger.Println("Request failed: ", err.Error())
		SendError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
}
