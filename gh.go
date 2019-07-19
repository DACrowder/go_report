package main

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

func NewGitHubAppInstallationClient(install int) (*github.Client, error) {
	tr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, cfg.GHAppID, install, cfg.GHPrivateKeyFile)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating app transport: %v", err.Error())
	}
	return github.NewClient(&http.Client{Transport: tr}), nil
}

func CreateGitHubIssue(rpt Report) error {
	gh, err := NewGitHubAppInstallationClient(cfg.GHInstallID)
	if err != nil {
		return err
	}
	body := fmt.Sprintf("--- Automated Crash Report ---\nKey: %v", rpt.key)
	issReq := github.IssueRequest{
		Title:  &rpt.GID,
		Body:   &body,
		Labels: &([]string{"Critical"}),
	}
	_, _, err = gh.Issues.Create(context.Background(), cfg.RepoOwner, cfg.RepoName, &issReq)
	if err != nil {
		return err
	}
	return nil
}

// CheckTokenUser takes a user's github oauth2 token, and confirms it is both a valid
// token, and that the token belongs to a contributor/collaborator of the target repo.
// It then returns the username associated with the token, e.g. for checking against a TokenRequest username
func CheckTokenUser(tkn string) (string, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tkn})
	tc := oauth2.NewClient(context.Background(), ts)
	gh := github.NewClient(tc)
	req, err := gh.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create /user request")
	}
	usr := new(github.User)
	_, err = gh.Do(context.Background(), req, usr)
	if err != nil {
		return "", errors.Wrap(err, "gh.Do failed")
	}
	if ok := IsContributorOrCollaborator(gh, *usr.Login); !ok {
		return "", errors.New(fmt.Sprintf("user %v is not a contributor/collaborator", *usr.Login))
	}
	return *usr.Login, nil
}

// IsContributor returns a boolean status for whether the given username is a repository contributor
func IsContributorOrCollaborator(insClient *github.Client, name string) (authorized bool) {
	repo, _, err := insClient.Repositories.Get(context.Background(), cfg.RepoOwner, cfg.RepoName)
	if err != nil {
		logger.Printf("Failed to confirm user is contributor because of error getting repository: %v", err.Error())
		return false
	} else if repo == nil {
		logger.Println("Failed to confirm user is contributor because repository was nil")
		return false
	}
	ok := isInCCList(insClient, repo.GetCollaboratorsURL(), name)
	if !ok { // not a collaborator...
		return isInCCList(insClient, *repo.ContributorsURL, name) // is contributor?
	}
	return true // is collaborator.
}

// isInCCList checks whether a given username is a contributor or collaborator to the target repository
// where ins is the repository installation client (see NewGitHubInstallationClient)
func isInCCList(ins *github.Client, url string, uname string) (isPresent bool) {
	r, err := ins.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		logger.Printf("Failed to create GET %v request: %v", url, err.Error())
		return false
	}
	haystack := make([]*github.User, 0, 16)
	rsp, err := ins.Do(context.Background(), r, &haystack)
	if err != nil {
		switch t := err.(type) {
		case *github.RateLimitError:
			logger.Printf("Rate limit error when attempting to check repo contributors.")
			return false
		case *json.InvalidUnmarshalError, *json.UnsupportedValueError,
			*json.UnmarshalTypeError, *json.UnsupportedTypeError:
			logger.Printf("could not read %v response body: %v - %v", url, t, err.Error())
			return false
		default:
			logger.Printf("failed to get response from %v, because: %v", url, err.Error())
			return false
		}
	} else if rsp.StatusCode == http.StatusNotFound {
		return false
	}
	ok := false
	for _, n := range haystack {
		if *n.Login == uname {
			ok = true
			break
		}
	}
	return ok
}
