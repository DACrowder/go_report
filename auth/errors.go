package auth

import (
	"github.com/pkg/errors"
	"go_report/failure"
	"net/http"
)

var (
	ErrMSSGHTokenRequest = failure.New(
		errors.New("! TokenRequest had MSSCert and Github user+token"),
		http.StatusBadRequest,
		"Supply an MSSCertificate or a Github login user + OAuth2 token. Not both.",
	)
	ErrIncompleteTokenRequest = failure.New(
		errors.New("TokenRequest requires github username + oauth2 token"),
		http.StatusBadRequest,
		"Authentication requires a github username and oauth2 token",
	)

	ErrCreatingToken = func(err error, st int) *failure.RequestFailure {
		return failure.New(
			errors.Wrap(err, "jwt creation unauthorized"),
			st,
			"",
		)
	}
)