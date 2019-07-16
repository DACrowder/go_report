package main

import (
	"context"
	"fmt"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	"net/http"
)


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
	_, _, err = gh.Issues.Create(context.Background(), "DACrowder", "go_report", &issReq)
	if err != nil {
		Log.Printf("error creating issue on github: %v", err.Error())
		return err
	}
	Log.Println("Successfully created github issue")
	return nil
}
