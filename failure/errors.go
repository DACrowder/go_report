package failure

import (
	"fmt"
	"github.com/pkg/errors"
	"net/http"
)

var (
	ErrWriteCertsFailed = New(
		errors.New("Failed to write updated certificates file"),
		http.StatusInternalServerError,
		"",
	)

	ErrMSSGHTokenRequest = New(
		errors.New("! TokenRequest had MSSCert and Github user+token"),
		http.StatusBadRequest,
		"Supply an MSSCertificate or a Github login user + OAuth2 token. Not both.",
	)
	ErrIncompleteTokenRequest = New(
		errors.New("TokenRequest requires github username + oauth2 token"),
		http.StatusBadRequest,
		"Authentication requires a github username and oauth2 token",
	)

	ErrCreatingToken = func(err error, st int) *RequestFailure {
		return New(
			errors.Wrap(err, "jwt creation unauthorized"),
			st,
			"",
		)
	}
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

