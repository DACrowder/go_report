package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"go_report/domain"
	"go_report/failure"
	"go_report/gh"
	"log"
	"net/http"
)

// Gets all reports with content
func GetAllHandler(s domain.Storer) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reports, err := s.SelectAll()
		if err != nil {
			failure.Fail(w, err)
			return
		}
		if err := json.NewEncoder(w).Encode(reports); err != nil {
			failure.Fail(w, err)
			return
		}
	})
}

// return content of files
func GetGroupHandler(s domain.Storer) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the stored reports in json corresponding to an GID (i.e. {<id>:[...files]})
		gid := r.Context().Value(string(ReportGIDVar)).(string)
		if gid == "" {
			GetAllHandler(s)(w, r)
			return
		}
		reports, err := s.SelectGroup(gid)
		if err != nil {
			failure.Fail(w, err)
			return
		}
		if err = json.NewEncoder(w).Encode(reports); err != nil {
			failure.Fail(w, err)
			return
		}
	})
}

// return content of file
func GetReportHandler(s domain.Storer) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g, k := r.Context().Value(string(ReportGIDVar)).(string), r.Context().Value(string(ReportKeyVar)).(string)
		rpt, err := s.Select(domain.Receipt{
			Key: k,
			GID: g,
		})
		if err != nil {
			failure.Fail(w, err)
			return
		}
		if err := json.NewEncoder(w).Encode(rpt); err != nil {
			failure.Fail(w, errors.Wrap(err, "failed to encode report json to http writer response stream"))
			return
		}
	})
}

func PostHandler(s domain.Storer, ghs *gh.Service, logger *log.Logger) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// read rpt from context
		rpt := r.Context().Value(string(ReportCtxVar)).(domain.Report)
		// add to s
		rr, err := s.NewEntry(rpt)
		if err != nil {
			failure.Fail(w, failure.New(errors.Wrap(err, "failed to create store entry"), http.StatusInternalServerError, ""))
			return
		}
		if err := json.NewEncoder(w).Encode(&rr); err != nil {
			failure.Fail(w, failure.New(errors.Wrap(err, "failed to encode reciept"), http.StatusInternalServerError, ""))
			return
		}
		if rpt.Severity == domain.CrashType {
			logger.Println("Creating github issue for crash report")
			err = ghs.CreateGitHubIssue(github.IssueRequest{
				Title:  github.String(rr.GID + " " + rr.Key),
				Body:   github.String(fmt.Sprintf("---- Automated Crash Report ----\n\nKey: %v", rr.Key)),
				Labels: &[]string{"Critical"},
			})
			if err != nil {
				logger.Println("failed to create github issue (key=%v): %v", rpt.Key, err.Error())
			} else {
				logger.Println("Successfully created github issue")
			}
		}
		w.WriteHeader(http.StatusCreated)
	})
}

// remove a single file by its key
func DeleteReportHandler(s domain.Storer) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g, k := r.Context().Value(string(ReportGIDVar)).(string), r.Context().Value(string(ReportKeyVar)).(string) // if we fail to convert to string, we have a big problem -> let recoverer middleware deal
		if err := s.RemoveEntry(domain.Receipt{Key: k, GID:g}); err != nil {
			failure.Fail(w, err)
		}
		w.WriteHeader(http.StatusNoContent)
	})
}