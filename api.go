package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"go_report/failure"
	"go_report/gh"
	"go_report/report"
	"net/http"
	"strings"
)

// Gets all reports with content
func GetAllHandler(s *report.Store) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys := make([]string, 0, 32)
		for k := range s.Keys(nil) {
			keys = append(keys, k)
		}
		reports, status := s.GetReportsWithKeys(keys...)
		sendResponseForRetrievedBatch(w, reports, status)
	})
}

// return content of files
func GetGroupHandler(s *report.Store) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the stored reports in json corresponding to an GID (i.e. {<id>:[...files]})
		gid := r.Context().Value(string(ReportGIDVar)).(string)
		if gid == "" {
			GetAllHandler(s)(w, r)
			return
		}
		keys := s.GetKeysByGID(gid)
		if len(keys) <= 0 {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		reports, status := s.GetReportsWithKeys(keys...)
		sendResponseForRetrievedBatch(w, reports, status)
	})
}

// return content of files
func GetBatchByTypeHandler(s *report.Store) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the stored reports in json corresponding to an GID (i.e. {<id>:[...files]})
		slvl := r.Context().Value(string(ReportSeverityLevelVar)).(string)
		if slvl == "" {
			GetAllHandler(s)(w, r)
			return
		}
		keys := make([]string, 0, 16)
		for k := range s.Keys(nil) {
			keys = append(keys, k)
		}
		severity := convertSeverityLevelString(slvl)
		reports, status := s.GetReportsWithKeys(keys...)
		for k, v := range reports {
			if v.Severity != severity {
				delete(reports, k)
			}
		}
		sendResponseForRetrievedBatch(w, reports, status)
	})
}

// return content of file
func GetReportHandler(s *report.Store) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.Context().Value(string(ReportKeyVar)).(string) // if this fails middleware is totally broken; let recoverer deal with the panic
		if ok := s.Has(query); !ok {
			found := false
			for key := range s.Keys(nil) {
				if strings.Contains(key, query) {
					found = true
					query = key
					break
				}
			}
			if !found {
				w.WriteHeader(http.StatusNotFound)
				return
			}

		}
		rb, err := s.Read(query)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if n, err := w.Write(rb); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else if n != len(rb) {
			w.WriteHeader(http.StatusPartialContent)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func PostHandler(s *report.Store, ghs *gh.Service) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// read rpt from context
		rpt := r.Context().Value(string(ReportCtxVar)).(report.Instance)
		// add to s
		k, v, err := s.CreateEntry(rpt)
		if err != nil {
			failure.Fail(w, failure.New(errors.Wrap(err, "failed to create diskv entry"), http.StatusInternalServerError, ""))
			return
		}
		rpt.Key = k
		if err := s.Write(k, v); err != nil {
			failure.Fail(w, failure.New(errors.Wrap(err, "failed to s diskv entry"), http.StatusInternalServerError, ""))
			return
		}
		splitKey := strings.Split(k, "/")
		// respond with Receipt
		rr := report.Receipt{GID: rpt.GID, FileName: splitKey[len(splitKey)-1]}
		if err := json.NewEncoder(w).Encode(&rr); err != nil {
			failure.Fail(w, failure.New(errors.Wrap(err, "failed to s diskv entry"), http.StatusInternalServerError, ""))
			return
		}
		if rpt.Severity == report.CrashType {
			logger.Println("Creating github issue for crash report")
			err = ghs.CreateGitHubIssue(github.IssueRequest{
				Title:  github.String(rpt.GID + " " + rpt.Key),
				Body:   github.String(fmt.Sprintf("---- Automated Crash Report ----\n\nKey: %v", rpt.Key)),
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

// remove all files in a group by GID
func DeleteGroupHandler(s *report.Store) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gid := r.Context().Value(string(ReportGIDVar)).(string)
		keys := s.GetKeysByGID(gid)
		if len(keys) <= 0 {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(map[string][]string{"deleted": keys}); err != nil {
			m := fmt.Sprintf("Could not delete group (%v): %v", gid, err.Error())
			failure.Fail(w, failure.New(errors.Wrap(err, "failed to s diskv entry"), http.StatusInternalServerError, m))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// remove a single file by its key
func DeleteReportHandler(s *report.Store) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		k := r.Context().Value(string(ReportKeyVar)).(string) // if we fail to convert to string, we have a big problem -> let recoverer middleware deal
		if ok := s.Has(k); !ok {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if err := s.Erase(k); err != nil {
			failure.Fail(w, failure.New(errors.Wrap(err, "failed to erase s entry "+k), http.StatusInternalServerError, ""))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func convertSeverityLevelString(slvl string) report.Type {
	switch strings.ToLower(slvl) {
	case "1", "bug":
		return report.BugType
	case "2", "crash":
		return report.CrashType
	default:
		return report.UnknownType
	}
}

func sendResponseForRetrievedBatch(w http.ResponseWriter, reports map[string]report.Instance, statusCode int) {
	switch statusCode {
	case http.StatusInternalServerError:
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	case http.StatusPartialContent:
		fallthrough
	case http.StatusOK:
		if err := json.NewEncoder(w).Encode(reports); err != nil {
			failure.Fail(w, failure.New(errors.Wrap(err, "failed to send batch response"), http.StatusInternalServerError, ""))
		}
	}
}
