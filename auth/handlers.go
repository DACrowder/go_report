package auth

import (
	"encoding/json"
	"github.com/go-chi/jwtauth"
	"github.com/pkg/errors"
	"go_report/failure"
	"net/http"
)

const CertCtxVar = "mssCertificate"

// Authenication middlewares

func (a *Authenticator) Verifier(next http.Handler) http.Handler {
	verifier := jwtauth.Verifier(a.jwt)(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		verifier.ServeHTTP(w, r)
	})
}

// Authenticate accepts either Appcertificate or developer jwts, rejecting unknown or invalid jwts
func (a *Authenticator) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tkn, claims, err := jwtauth.FromContext(r.Context())
		if err != nil || tkn == nil || !tkn.Valid {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		ok := claims.VerifyAudience(string(MSSAudience), true)
		if !(ok || claims.VerifyAudience(string(GHAudience), true)) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		// Token is authenticated, pass it through
		next.ServeHTTP(w, r.WithContext(r.Context()))
	})
}

// OnlyDevsAuthenticate only authenticates jwts which correspond to a github developer
func (a *Authenticator) OnlyDevsAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// jwt is linked to github, pass it through
		_, claims, err := jwtauth.FromContext(r.Context())
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		if ok := claims.VerifyAudience(string(GHAudience), true); !ok {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		if claims[string(GHUser)] == "" || claims[string(GHToken)] == "" {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Endpoints

type TokenRequest struct {
	User        string `json:"ghUser,omitempty"`
	GitHubToken string `json:"ghToken,omitempty"`
	MSSCert     string `json:"mssCert,omitempty"`
}

func (a *Authenticator) TokenExchangeHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tr := TokenRequest{}
		if err := json.NewDecoder(r.Body).Decode(&tr); err != nil {
			failure.Fail(w, ErrCreatingToken(errors.Wrap(err, "failed to decode token request body"), http.StatusBadRequest))
			return
		}
		tkn, err := a.maybeCreateJWT(tr) // we may need to do more processing here
		if err != nil {
			failure.Fail(w, err)
			return
		}
		// access granted.
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(tkn))
		return
	})
}


func (a *Authenticator) AddCertificateHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cert := r.Context().Value(string(CertCtxVar)).(string)
		if cert == "" {
			failure.Fail(w, failure.New(errors.New("No certificate found in context"), http.StatusBadRequest, "no certificate provided"))
			return
		}
		if err := a.cm.AddCertificate(cert); err != nil {
			failure.Fail(w, failure.New(err, http.StatusInternalServerError, "could not add certificate"))
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
}

func (a *Authenticator) RemoveCertificateHandler() http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cert := r.Context().Value(string(CertCtxVar)).(string)
		if cert == "" {
			failure.Fail(w, failure.New(errors.New("No certificate found in context"), http.StatusBadRequest, "no certificate provided"))
			return
		}
		if err  := a.cm.RemoveCertificate(cert); err != nil {
			failure.Fail(w, failure.New(err, http.StatusInternalServerError, "could not remove certificate"))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

