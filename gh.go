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

// todo: load from config
const repoOwner = "DACrowder"
const repoName = "go_report"

func NewGitHubAppClient() (*github.Client, error) {
	appTr, err := ghinstallation.NewAppsTransportKeyFromFile(
		http.DefaultTransport,
		Cfg.GHAppID,
		Cfg.GHPrivateKeyFile,
	)
	if err != nil {
		Log.Printf("error creating app transport: %v", err.Error())
		return nil, err
	}
	return github.NewClient(&http.Client{Transport: appTr}), nil
}

func NewGitHubAppInstallationClient(install int) (*github.Client, error) {
	tr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, Cfg.GHAppID, install, Cfg.GHPrivateKeyFile)
	if err != nil {
		Log.Printf("error creating app transport: %v", err.Error())
		return nil, err
	}
	return github.NewClient(&http.Client{Transport: tr}), nil
}

func CreateGitHubIssue(rpt Report) (error) {
	gh, err := NewGitHubAppInstallationClient(Cfg.GHInstallID)
	if err != nil {
		return err
	}
	body := fmt.Sprintf("--- Automated Crash Report ---\nKey: %v", rpt.key)
	issReq := github.IssueRequest{
		Title: &rpt.GID,
		Body: &body,
		Labels: &([]string{"Critical"}),
	}
	_, _, err = gh.Issues.Create(context.Background(), repoOwner, repoName, &issReq)
	if err != nil {
		Log.Printf("error creating issue on github: %v", err.Error())
		return err
	}
	Log.Println("Successfully created github issue")
	return nil
}

// given a github token, will return the associated username
func RequestAuthedUserFromToken(tkn string) (string, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tkn})
	tc := oauth2.NewClient(context.Background(), ts)
	gh := github.NewClient(tc)
	req, err := gh.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		err = errors.Wrap(err, "failed to create /user request")
		return "", err
	}
	usr := new(github.User)
	_, err = gh.Do(context.Background(), req, usr)
	if err != nil {
		return "", errors.Wrap(err, "gh.Do failed")
	}
	return *usr.Login, nil
}

// IsContributor returns a boolean status for whether the given username is a repository contributor
func IsContributorOrCollaborator(insClient *github.Client, name string) (authorized bool) {
	repo, _, err := insClient.Repositories.Get(context.Background(), repoOwner, repoName)
	if err != nil {
		Log.Printf("Failed to confirm user is contributor because of error getting repository: %v", err.Error())
		return false
	} else if repo == nil {
		Log.Println("Failed to confirm user is contributor because repository was nil")
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
		Log.Printf("Failed to create GET %v request: %v", url, err.Error())
		return false
	}
	haystack := make([]*github.User, 0, 16)
	rsp, err := ins.Do(context.Background(), r, haystack)
	if err != nil {
		switch t := err.(type) {
		case *github.RateLimitError:
			Log.Printf("Rate limit error when attempting to check repo contributors.")
			return false
		case *json.InvalidUnmarshalError, *json.UnsupportedValueError,
					*json.UnmarshalTypeError, *json.UnsupportedTypeError:
			Log.Printf("could not read %v response body: %v - %v", url, t, err.Error())
			return false
		default:
			Log.Printf("failed to get response from %v, because: %v", url, err.Error())
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