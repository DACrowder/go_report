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
)

type Secrets struct {
	PrivateKeyFile string `json:"ghPrivateKeyFile"` // pem encoded rsa key
	AppID          int    `json:"ghAppID"`
	InstallID      int    `json:"ghInstallID"`
	WebhookSecret  string `json:"ghWebhookSecret"`
	ClientID       string `json:"ghClientID"`
	ClientSecret   string `json:"ghClientSecret"`
}

type Repo struct {
	Owner string  `json:"targetRepoOwner"`
	Name string	`json:"targetRepoName"`
}

type Service struct {
	Secrets
	Repo
}

func New(repo Repo, shh Secrets) *Service {
	return &Service{
		Secrets: shh,
		Repo: repo,
	}
}

func (s *Service) newTokenClient(tkn string) (*github.Client) {
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

func (s *Service) CreateGitHubIssue(issReq *github.IssueRequest) error {
	gh, err := s.newInstallationClient()
	if err != nil {
		return err
	}
	_, _, err = gh.Issues.Create(context.Background(), s.Repo.Owner, s.Repo.Name, issReq)
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
	ok, err := IsInCCList(insClient, repo.GetCollaboratorsURL(), name)
	if err != nil {
		return false, err
	}
	if !ok { // not a collaborator...
		return IsInCCList(insClient, *repo.ContributorsURL, name) // is contributor?
	}
	return true, nil // is collaborator.
}

// IsInCCList checks whether a given username is a contributor or collaborator to the target repository
// where ins is the repository installation client (see NewGitHubInstallationClient)
func IsInCCList(ins *github.Client, url string, uname string) (isPresent bool, err error) {
	r, err := ins.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, errors.Errorf("Failed to create GET %v request: %v", url, err.Error())
	}
	haystack := make([]*github.User, 0, 16)
	rsp, err := ins.Do(context.Background(), r, &haystack)
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
	ok := false
	for _, n := range haystack {
		if *n.Login == uname {
			ok = true
			break
		}
	}
	return ok, nil
}
