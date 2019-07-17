package main

import (
	"encoding/json"
	"encoding/pem"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	TokenExpiryDuration = (time.Minute * 60 * 24 * 365) // one year duration

	// "aud" claim values
	GHAudience = "github"
	MSSAudience = "mss"

	// Claim Keys
	GHUserClaimKey         = "ghuname"
	GHTokenClaimKey        = "ghtkn"
	MSSCertificateClaimKey = "mssCert"
)

var jwtAuth *jwtauth.JWTAuth

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
	User 		string	`json:"ghUser,omitempty"`
	GitHubToken string 	`json:"ghToken,omitempty"`
	MSSCert 	string 	`json:"mssToken,omitempty"`
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
		ok := claims.VerifyAudience(MSSAudience, true)
		if !(ok || claims.VerifyAudience(GHAudience, true)) {
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
		if ok := claims.VerifyAudience(GHAudience, true); !ok {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		if claims[GHUserClaimKey] == "" || claims[GHTokenClaimKey] == "" {
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
			Log.Printf("could not read token request body: %v", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		} else if tr.MSSCert != "" && (tr.GitHubToken != "" || tr.User != "") {
			Log.Printf("SECURITY? TokenRequest with an MSSCertificate and Github user/token was rejected")
			_, _ = w.Write([]byte("Supply an MSSCertificate or a Github login user + OAuth2 token. Not both."))
			w.WriteHeader(http.StatusBadRequest)
			return
		} else if tr.GitHubToken == "" || tr.User == "" {
			_, _ = w.Write([]byte("Authorization requires a github username and oauth2 token"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, _ = maybeCreateJWTHandler(*tr, w, r) // we may need to do more processing here
		return
	})
}

func maybeCreateJWTHandler(tr TokenRequest, w http.ResponseWriter, r *http.Request) (jwt string, err error) {
	if tr.MSSCert != "" {
		jwt, err = newSignedMSSJWT(tr.MSSCert)
	} else {
		jwt, err = newUserJWT(tr.User, tr.GitHubToken)
	}
	if err != nil {
		Log.Printf("Error creating jwt: %v", err.Error())
		if err == jwtauth.ErrUnauthorized {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		return
	} else if jwt == "" {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	// access granted.
	_, _ = w.Write([]byte(jwt))
	w.WriteHeader(http.StatusCreated)
	return
}

func newUserJWT(user string, ghTkn string) (tkn string, err error) {
	retUser, err := RequestAuthedUserFromToken(ghTkn)
	if err != nil {
		return "", err
	} else if retUser != user {
		return "", jwtauth.ErrUnauthorized
	}
	_, tkn, err = jwtAuth.Encode(jwt.MapClaims{
		"aud":           GHAudience,
		GHUserClaimKey:  user,
		GHTokenClaimKey: ghTkn,
		"iss": "mss_go_report",
		"iat" : time.Now().Unix(),
		"exp": time.Now().Add(TokenExpiryDuration).Unix(),
		"nbf": time.Now().Unix(),
	})
	return
}

func newSignedMSSJWT(mssCert string) (tkn string, err error) {
	_, tkn, err = jwtAuth.Encode(jwt.MapClaims{
		"aud": MSSAudience,
		MSSCertificateClaimKey: mssCert,
		"iss": "mss_go_report",
		"iat" : time.Now().Unix(),
		"exp": time.Now().Add(TokenExpiryDuration).Unix(),
		"nbf": time.Now().Unix(),
	})
	return
}

func GetKeyFromPemFile(fp string) ([]byte, error) {
	b, err := ioutil.ReadFile(fp)
	if err != nil {
		return []byte(""), err
	}
	blk, _ := pem.Decode(b)
	if blk == nil {
		return []byte(""), errors.New("failed to decode key from file")
	}
	return blk.Bytes, nil
}