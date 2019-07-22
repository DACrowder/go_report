package auth

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"
	"github.com/pkg/errors"
	"go_report/gh"
	"go_report/msscerts"
	"log"
	"net/http"
	"time"
)

const ExpiresOneYear = (time.Minute * 60 * 24 * 365)

type JwtAudience string
const (
	GHAudience  JwtAudience = "github"
	MSSAudience JwtAudience = "mss"
)

type JwtClaimKey string
const (
	GHUser         JwtClaimKey = "ghuname"
	GHToken        JwtClaimKey = "ghtkn"
	MSSCertificate JwtClaimKey = "mssCert"
)


type Authenticator struct {
	cm *msscerts.Manager
	ghs *gh.Service
	jwt *jwtauth.JWTAuth
}

func New(cfg Secrets, logger *log.Logger) (s *Authenticator) {
	msscerts.Init(cfg.MSSCertsFile, logger)
	repo := gh.Repo{
		Owner: cfg.GHRepoOwner,
		Name:  cfg.GHRepoName,
	}
	shh := gh.Secrets{
		PrivateKeyFile: cfg.GHPrivateKeyFile,
		AppID: cfg.GHAppID,
		InstallID: cfg.GHInstallID,
		WebhookSecret: cfg.GHWebhookSecret,
		ClientID: cfg.GHClientID,
		ClientSecret: cfg.GHClientSecret,
	}
	return &Authenticator{
		cm: msscerts.GetManager(),
		ghs: gh.New(repo, shh),
		jwt: jwtauth.New(jwt.SigningMethodHS512.Name, []byte(cfg.JWTKey), nil),
	}
}

func (a *Authenticator) maybeCreateJWT(tr TokenRequest) (tkn string, err error) {
	// Enforce token request is either cert based, or github based.
	if tr.MSSCert != "" && (tr.GitHubToken != "" || tr.User != "") {
		return "", ErrMSSGHTokenRequest
	} else if tr.MSSCert == "" && (tr.GitHubToken == "" || tr.User == "") {
		return "", ErrIncompleteTokenRequest
	}
	// select token request branch
	if tr.MSSCert != "" {
		tkn, err = a.newSignedAppJWT(tr.MSSCert)
	} else {
		tkn, err = a.newSignedDevJWT(tr.User, tr.GitHubToken)
	}
	// report any failures
	if err == jwtauth.ErrUnauthorized {
		return "", ErrCreatingToken(err, http.StatusUnauthorized)
	} else if err != nil {
		return "", ErrCreatingToken(err, http.StatusServiceUnavailable)
	} else if tkn == "" {
		return "", ErrCreatingToken(errors.New("returned tkn was empty string"), http.StatusServiceUnavailable)

	}
	return tkn, nil
}

func (a *Authenticator) newSignedDevJWT(user string, ghTkn string) (tkn string, err error) {
	ok, err := a.ghs.VerifyDeveloperToken(user, ghTkn)
	if err != nil {
		return "", err
	} else if !ok {
		return "", jwtauth.ErrUnauthorized
	}

	_, tkn, err = a.jwt.Encode(jwt.MapClaims{
		"aud":           string(GHAudience),
		string(GHUser):  user,
		string(GHToken): ghTkn,
		"iss":           "mss_go_report",
		"iat":           time.Now().Unix(),
		"exp":           time.Now().Add(ExpiresOneYear).Unix(),
		"nbf":           time.Now().Unix(),
	})
	return tkn, nil
}

func (a *Authenticator) newSignedAppJWT(mssCert string) (tkn string, err error) {
	if ok, err := a.cm.Verify(mssCert); err != nil {
		return "", err
	} else if !ok {
		return "", jwtauth.ErrUnauthorized
	}
	_, tkn, err = a.jwt.Encode(jwt.MapClaims{
		"aud":                  string(MSSAudience),
		string(MSSCertificate): mssCert,
		"iss":                  "mss_go_report",
		"iat":                  time.Now().Unix(),
		"exp":                  time.Now().Add(ExpiresOneYear).Unix(),
		"nbf":                  time.Now().Unix(),
	})
	return tkn, nil
}

