package main

import (
	"context"
	"fmt"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
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

// given username & bearer token string, get a Github client for the user
func NewGitHubUserClient(user string, tkn string) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tkn})
	tc := oauth2.NewClient(context.Background(), ts)
	gh := github.NewClient(tc)
	gh.Authorizations.Check(context.Background(), user, tkn)
	return gh, nil
}

// IsContributor returns a boolean status for whether the given username is a repository contributor
func IsContributor(userClient *github.Client) bool {
	repo, _, err := userClient.Repositories.Get(context.Background(), repoOwner, repoName)
	if err != nil {
		Log.Printf("Failed to confirm user is contributor because of error: %v")
		return false
	} else if repo == nil {
		return false
	}
	return true
}