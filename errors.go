package main

import (
	"encoding/json"
	"log"
	"net/http"
	"github.com/pkg/errors"
)
//contains a error message to be sent out as json
type ErrorMessage struct {
	Code        int    `json:"code"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

//sends an http error (same as http.error) in json format
func SendError(w http.ResponseWriter, s int, description string) {
	errMsg := ErrorMessage{
		Code:        s,
		Status:      http.StatusText(s),
		Description: description,
	}
	err := json.NewEncoder(w).Encode(errMsg)
	if err != nil {
		Log.Println("Failed to send error to client, json encoding failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(s)
}

func NewFailure(err error, code int, safeMsg string) *ErrRequestFailed {
	return &ErrRequestFailed{
		error:   err,
		status:  code,
		safeMsg: safeMsg,
	}
}

type ErrRequestFailed struct {
	error
	status  int
	safeMsg string
}

func (e ErrRequestFailed) StatusCode() int {
	return e.status
}

func (e ErrRequestFailed) Message() string {
	if e.status == 0 {
		return ""
	}
	if e.safeMsg == "" {
		return http.StatusText(e.status)
	}
	return e.safeMsg
}

func (e ErrRequestFailed) Error() string {
	return e.error.Error()
}

func (e ErrRequestFailed) Err() error {
	return e.error
}

// HandleError writes an http.Error + logs the error & its call-tree (if available)
// additional parsing for RequestFailure errors is performed, and the safeMessage + code are sent to the user
// If the error is not a RequestFailure, HandleError defaults to a basic http500 error.
func HandleError(w http.ResponseWriter, err error) {
	switch e := errors.Cause(err).(type) {
	case *json.UnsupportedValueError, *json.UnsupportedTypeError, *json.SyntaxError, *json.UnmarshalTypeError:
		log.Printf("%+v\n", err)
		SendError(w, http.StatusBadRequest, "JSON format error")
	case ErrRequestFailed:
		log.Printf("%v\t%+v\n", e.Err(), e)
		SendError(w, e.StatusCode(), e.Message())
	default:
		log.Printf("%+v\n", err)
		SendError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
}

