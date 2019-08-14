package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"net/http"
	"os"
	"path/filepath"
)

type Secrets struct {
	PrivateKeyFile string `json:"ghPrivateKeyFile" paramName:"GH_APP_KEY,secret"` // pem encoded rsa key
	AppID          int    `json:"ghAppID" paramName:"GH_APP_ID,secret"`
	InstallID      int    `json:"ghInstallID" paramName:"GH_INSTALL_ID,secret"`
	//WebhookSecret  string `json:"ghWebhookSecret" paramName:"GH_WEBHOOK,secret"` // not needed
	ClientID       string `json:"ghClientID" paramName:"GH_CLIENT_ID,secret"`
	ClientSecret   string `json:"ghClientSecret" paramName:"GH_CLIENT_SECRET,secret"`
}

type Repo struct {
	Owner string `json:"targetRepoOwner" paramName:"GH_REPO_OWNER,secret"`
	Name  string `json:"targetRepoName" paramName:"GH_REPO_NAME,secret"`
}

type Service struct {
	Secrets
	Repo
}

//ReadConfig reads a _secrets.json file into a Config struct
func NewFromFile(fp string) (s *Service, err error) {
	shh := new(Service)
	if ok := filepath.IsAbs(fp); !ok {
		return shh, errors.New("path to service configuration must be an absolute path")
	}
	fd, err := os.Open(fp)
	if err != nil {
		return shh, err
	}
	if err = json.NewDecoder(fd).Decode(&shh); err != nil {
		return shh, err
	}
	return shh, fd.Close()
}

func New(repo Repo, shh Secrets) *Service {
	return &Service{
		Secrets: shh,
		Repo:    repo,
	}
}

func (s *Service) newTokenClient(tkn string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tkn})
	tc := oauth2.NewClient(context.Background(), ts)
	gh := github.NewClient(tc)
	return gh
}

func (s *Service) newInstallationClient() (*github.Client, error) {
	tr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, s.AppID, s.InstallID, s.PrivateKeyFile)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating app transport: %v", err.Error())
	}
	return github.NewClient(&http.Client{Transport: tr}), nil
}

func (s *Service) CreateGitHubIssue(issReq github.IssueRequest) error {
	gh, err := s.newInstallationClient()
	if err != nil {
		return err
	}
	_, _, err = gh.Issues.Create(context.Background(), s.Repo.Owner, s.Repo.Name, &issReq)
	if err != nil {
		return err
	}
	return nil
}

// GetUserFromToken takes a user's github oauth2 token, and confirms it is both a valid
// token, and that the token belongs to a contributor/collaborator of the target repo.
// if the token request user is not the same as the github token username, false + error is returned.
// For all cases other than a successful verification, false + error is returned.
func (s *Service) VerifyDeveloperToken(user, ghTkn string) (bool, error) {
	gh := s.newTokenClient(ghTkn)
	req, err := gh.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to create /user request")
	}
	usr := new(github.User)
	_, err = gh.Do(context.Background(), req, usr)
	if err != nil {
		return false, errors.Wrap(err, "gh.Do failed")
	}
	if ok, err := s.IsContributorOrCollaborator(*usr.Login); err != nil {
		return false, errors.Wrap(err, "")
	} else if !ok {
		return false, errors.New(fmt.Sprintf("user %v is not a contributor/collaborator", *usr.Login))
	} else if user != *usr.Login {
		return false, errors.New(fmt.Sprintf("request user %v != github token user %v", user, *usr.Login))
	}
	return true, nil
}

// IsContributor returns a boolean status for whether the given username is a repository contributor
func (s *Service) IsContributorOrCollaborator(name string) (authorized bool, err error) {
	var insClient *github.Client
	insClient, err = s.newInstallationClient()
	if err != nil {
		return false, errors.Wrap(err, "Failed to get app installation client")
	}
	repo, _, err := insClient.Repositories.Get(context.Background(), s.Repo.Owner, s.Repo.Name)
	if err != nil {
		return false, errors.Errorf("Failed to confirm user is contributor because of error getting repository: %v", err.Error())
	} else if repo == nil {
		return false, errors.New("Failed to confirm user is contributor because repository was nil")
	}
	url := fmt.Sprintf("https://api.github.com/repos/%v/%v/collaborators", s.Owner, s.Name)
	ok, err := IsInCCList(insClient, url, name)
	if err != nil {
		return false, err
	}
	if !ok { // not a collaborator...
		url := fmt.Sprintf("https://api.github.com/repos/%v/%v/contributors", s.Owner, s.Name)
		return IsInCCList(insClient, url, name) // is contributor?
	}
	return true, nil // is collaborator.
}

// IsInCCList checks whether a given username is a contributor or collaborator to the target repository
// where ins is the repository installation client (see NewGitHubInstallationClient)
func IsInCCList(ins *github.Client, url string, uname string) (isPresent bool, err error) {
	r, err := ins.NewRequest(http.MethodGet, url+"/"+uname, nil)
	if err != nil {
		return false, errors.Errorf("Failed to create GET %v request: %v", url, err.Error())
	}
	rsp, err := ins.Do(context.Background(), r, nil)
	if err != nil {
		switch t := err.(type) {
		case *github.RateLimitError:
			return false, errors.Wrap(err, "Rate limit error when attempting to check repo contributors.")
		case *json.InvalidUnmarshalError, *json.UnsupportedValueError,
			*json.UnmarshalTypeError, *json.UnsupportedTypeError:
			return false, errors.Wrapf(err, "could not read %v response body: %v", url, t)
		default:
			return false, errors.Wrapf(err, "failed to get response from %v", url)
		}
	} else if rsp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return true, nil
}
