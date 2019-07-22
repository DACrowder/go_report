package failure

import (
	"encoding/json"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"sync"
)

// Endpoint related sync + instance
var (
	initOnce sync.Once
	ep *Endpoint
)

type Endpoint struct {
	*log.Logger
}

func Init(logger *log.Logger) {
	initOnce.Do(func () {
		ep = &Endpoint{
			Logger: logger,
		}
	})
}

func Fail(w http.ResponseWriter, err error) {
	switch errors.Cause(err).(type) {
	case *json.UnsupportedValueError, *json.UnsupportedTypeError, *json.SyntaxError, *json.UnmarshalTypeError:
		SendError(w, http.StatusBadRequest, "JSON format error")
		ep.Printf("request failed - json format error: %v", err.Error())
	case *RequestFailure:
		rf := err.(*RequestFailure)
		ep.Println("Request failed: ", err.Error())
		SendError(w, rf.Code, rf.Msg)
	case RequestFailure:
		ep.Println("Request failed: ", err.Error())
		rf := err.(RequestFailure)
		SendError(w, rf.Code, rf.Msg)
	default:
		ep.Println("Request failed (internal server error): ", err.Error())
		SendError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}
}

//sends an http error in json format
func SendError(w http.ResponseWriter, statusCode int, userMessage string) {
	type ErrorMessage struct {
		Code        int    `json:"code"`
		Status      string `json:"status"`
		Description string `json:"userMessage"`
	}
	errorMes := ErrorMessage{
		Code:        statusCode,
		Status:      http.StatusText(statusCode),
		Description: userMessage,
	}
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(errorMes); err != nil {
		ep.Printf("Error response not sent: %v", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}