package main

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	errors2 "go_report/failure"
	"net/http"
	"strings"
)

// Gets all reports with content
func GetAllHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys := make([]string, 0, 32)
		for k := range store.Keys(nil) {
			keys = append(keys, k)
		}
		reports, status := GetReportsWithKeys(keys...)
		sendResponseForRetrievedBatch(w, reports, status)
	})
}

// return content of files
func GetGroupHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the stored reports in json corresponding to an GID (i.e. {<id>:[...files]})
		gid := r.Context().Value(string(ReportGIDVar)).(string)
		if gid == "" {
			GetAllHandler()(w, r)
			return
		}
		keys := GetKeysByGID(gid)
		if len(keys) <= 0 {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		reports, status := GetReportsWithKeys(keys...)
		sendResponseForRetrievedBatch(w, reports, status)
	})
}

// return content of files
func GetBatchByTypeHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the stored reports in json corresponding to an GID (i.e. {<id>:[...files]})
		slvl := r.Context().Value(string(ReportSeverityLevelVar)).(string)
		if slvl == "" {
			GetAllHandler()(w, r)
			return
		}
		keys := make([]string, 0, 16)
		for k := range store.Keys(nil) {
			keys = append(keys, k)
		}
		severity := convertSeverityLevelString(slvl)
		reports, status := GetReportsWithKeys(keys...)
		for k, v := range reports {
			if v.Severity != severity {
				delete(reports, k)
			}
		}
		sendResponseForRetrievedBatch(w, reports, status)
	})
}

// return content of file
func GetReportHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.Context().Value(string(ReportKeyVar)).(string) // if this fails middleware is totally broken; let recoverer deal with the panic
		if ok := store.Has(query); !ok {
			found := false
			for key := range store.Keys(nil) {
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
		rb, err := store.Read(query)
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

func PostHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// read rpt from context
		rpt := r.Context().Value(string(ReportCtxVar)).(Report)
		// add to store
		k, v, err := CreateEntry(rpt)
		if err != nil {
			errors2.Fail(w, errors2.New(errors.Wrap(err, "failed to create diskv entry"), http.StatusInternalServerError, ""))
			return
		}
		rpt.key = k
		if err := store.Write(k, v); err != nil {
			errors2.Fail(w, errors2.New(errors.Wrap(err, "failed to store diskv entry"), http.StatusInternalServerError, ""))
			return
		}
		splitKey := strings.Split(k, "/")
		// respond with ReportReceipt
		rr := ReportReceipt{GID: rpt.GID, FileName: splitKey[len(splitKey)-1]}
		if err := json.NewEncoder(w).Encode(&rr); err != nil {
			errors2.Fail(w, errors2.New(errors.Wrap(err, "failed to store diskv entry"), http.StatusInternalServerError, ""))
			return
		}
		if rpt.Severity == Crash {
			logger.Println("Creating github issue for crash report")
			err = CreateGitHubIssue(rpt) // if this fails, its logged. Not a huge deal
			if err != nil {
				logger.Println("failed to create github issue (key=%v): %v", rpt.key, err.Error())
			} else {
				logger.Println("Successfully created github issue")
			}
		}
		w.WriteHeader(http.StatusCreated)
	})
}

// remove all files in a group by GID
func DeleteGroupHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gid := r.Context().Value(string(ReportGIDVar)).(string)
		keys := GetKeysByGID(gid)
		if len(keys) <= 0 {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(map[string][]string{"deleted": keys}); err != nil {
			m := fmt.Sprintf("Could not delete group (%v): %v", gid, err.Error())
			errors2.Fail(w, errors2.New(errors.Wrap(err, "failed to store diskv entry"), http.StatusInternalServerError, m))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// remove a single file by its key
func DeleteReportHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		k := r.Context().Value(string(ReportKeyVar)).(string) // if we fail to convert to string, we have a big problem -> let recoverer middleware deal
		if ok := store.Has(k); !ok {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if err := store.Erase(k); err != nil {
			errors2.Fail(w, errors2.New(errors.Wrap(err, "failed to erase store entry "+k), http.StatusInternalServerError, ""))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func convertSeverityLevelString(slvl string) ReportType {
	switch strings.ToLower(slvl) {
	case "1", "bug":
		return Bug
	case "2", "crash":
		return Crash
	default:
		return Unknown
	}
}

func sendResponseForRetrievedBatch(w http.ResponseWriter, reports map[string]Report, statusCode int) {
	switch statusCode {
	case http.StatusInternalServerError:
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	case http.StatusPartialContent:
		fallthrough
	case http.StatusOK:
		if err := json.NewEncoder(w).Encode(reports); err != nil {
			errors2.Fail(w, errors2.New(errors.Wrap(err, "failed to send batch response"), http.StatusInternalServerError, ""))
		}
	}
}
