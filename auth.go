package main

import (
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"
	"github.com/pkg/errors"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Secrets struct {
	JWTKey string `json:"jwtKey"`
	// GitHub secrets
	GHPrivateKeyFile string `json:"ghPrivateKeyFile"` // pem encoded rsa key
	GHAppID          int    `json:"ghAppID"`
	GHInstallID      int    `json:"ghInstallID"`
	GHWebhookSecret  string `json:"ghWebhookSecret"`
	GHClientID       string `json:"ghClientID"`
	GHClientSecret   string `json:"ghClientSecret"`
}

type TokenRequest struct {
	User        string `json:"ghUser,omitempty"`
	GitHubToken string `json:"ghToken,omitempty"`
	MSSCert     string `json:"mssToken,omitempty"`
}

//ReadConfig reads a _secrets.json file into a Config struct
func ReadSecrets(fp string) (s Secrets, err error) {
	shh := Secrets{}
	if ok := filepath.IsAbs(fp); !ok {
		return shh, errors.New("path to secrets must be an absolute path")
	}
	fd, err := os.Open(fp)
	if err != nil {
		return shh, err
	}
	if err = json.NewDecoder(fd).Decode(&shh); err != nil {
		return shh, err
	}
	if ok := filepath.IsAbs(shh.GHPrivateKeyFile); !ok {
		return shh, errors.New("path to gh private key must be an absolute path")
	}
	return shh, fd.Close()
}

func Authenticator(next http.Handler) http.Handler {
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
		next.ServeHTTP(w, r)
	})
}

func OnlyDevsAuthenticator(next http.Handler) http.Handler {
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

func TokenExchangeHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tr := new(TokenRequest)
		if err := json.NewDecoder(r.Body).Decode(tr); err != nil {
			Fail(w, ErrCreatingToken(errors.Wrap(err, "failed to decode token request body"), http.StatusBadRequest))
			return
		} else if tr.MSSCert != "" && (tr.GitHubToken != "" || tr.User != "") {
			Fail(w, ErrMSSGHTokenRequest)
			return
		} else if tr.GitHubToken == "" || tr.User == "" {
			Fail(w, ErrIncompleteTokenRequest)
			return
		}
		_, _ = tr.maybeCreateJWT(w) // we may need to do more processing here
		return
	})
}

func (tr TokenRequest) maybeCreateJWT(w http.ResponseWriter) (jwt string, err error) {
	if tr.MSSCert != "" {
		jwt, err = newSignedMSSJWT(tr.MSSCert)
	} else {
		jwt, err = newUserJWT(tr.User, tr.GitHubToken)
	}

	if err == jwtauth.ErrUnauthorized {
		Fail(w, ErrCreatingToken(err, http.StatusUnauthorized))
		return
	} else if err != nil {
		Fail(w, ErrCreatingToken(err, http.StatusServiceUnavailable))
		return
	} else if jwt == "" {
		Fail(w, ErrCreatingToken(errors.New("returned jwt was empty string"), http.StatusServiceUnavailable))
		return
	}
	// access granted.
	_, _ = w.Write([]byte(jwt))
	w.WriteHeader(http.StatusCreated)
	return
}

func newUserJWT(user string, ghTkn string) (tkn string, err error) {
	retUser, err := CheckTokenUser(ghTkn)
	if err != nil {
		return "", err
	} else if retUser != user {
		return "", jwtauth.ErrUnauthorized
	}

	_, tkn, err = jwtAuth.Encode(jwt.MapClaims{
		"aud":           string(GHAudience),
		string(GHUser):  user,
		string(GHToken): ghTkn,
		"iss":           "mss_go_report",
		"iat":           time.Now().Unix(),
		"exp":           time.Now().Add(ExpiresOneYear).Unix(),
		"nbf":           time.Now().Unix(),
	})
	return
}

func newSignedMSSJWT(mssCert string) (tkn string, err error) {
	_, tkn, err = jwtAuth.Encode(jwt.MapClaims{
		"aud":                  string(MSSAudience),
		string(MSSCertificate): mssCert,
		"iss":                  "mss_go_report",
		"iat":                  time.Now().Unix(),
		"exp":                  time.Now().Add(ExpiresOneYear).Unix(),
		"nbf":                  time.Now().Unix(),
	})
	return
}
