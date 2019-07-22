package failure

import (
	"fmt"
	"net/http"
)

type RequestFailure struct {
	err  error  // developer level error for logging
	Code int    // http status
	Msg  string // public facing message to send
}

func New(err error, statusCode int, userMsg string) *RequestFailure {
	if userMsg == "" {
		userMsg = http.StatusText(statusCode)
	}
	return &RequestFailure{
		err:  err,
		Code: statusCode,
		Msg:  userMsg,
	}
}

func (rf RequestFailure) Error() string {
	return fmt.Sprintf("%v - %v", http.StatusText(rf.Code), rf.Msg)
}

