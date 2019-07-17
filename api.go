package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Gets all reports with content
func GetAllHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys := make([]string, 0, 32)
		for k := range Store.Keys(nil) {
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
		gid := r.Context().Value(ReportGIDVar).(string)
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
		slvl := r.Context().Value(ReportSeverityLevelVar).(string)
		if slvl == "" {
			GetAllHandler()(w, r)
			return
		}
		keys := make([]string, 0, 16)
		for k := range Store.Keys(nil) {
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
		query := r.Context().Value(ReportKeyVar).(string) // if this fails middleware is totally broken; let recoverer deal with the panic
		if ok := Store.Has(query); !ok {
			found := false
			for key := range Store.Keys(nil) {
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
		rb, err := Store.Read(query)
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
		rpt := r.Context().Value(ReportCtxVar).(Report)
		Log.Println("Report Post request recieved")
		// add to store
		k, v, err := CreateEntry(rpt)
		if err != nil {
			Log.Printf("could not create entry error: %v\n", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		rpt.key = k
		if err := Store.Write(k, v); err != nil {
			Log.Printf("failed to store entry: (key: %v, error: %v)", k, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		splitKey := strings.Split(k, "/")
		// respond with ReportReceipt
		rr := ReportReceipt{GID: rpt.GID, FileName: splitKey[len(splitKey)-1]}
		if err := json.NewEncoder(w).Encode(&rr); err != nil {
			Log.Printf("failed to send reciept after creation: (key: %v) %v", k, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		Log.Printf("Crash:%v, Severity: %v, Crash==Severity: %v", Crash, rpt.Severity, Crash == rpt.Severity)
		if rpt.Severity == Crash {
			Log.Println("Creating github issue for crash report")
			_ = CreateGitHubIssue(rpt) // if this fails, its logged. Not a huge deal
		}
		w.WriteHeader(http.StatusCreated)
	})
}

// remove all files in a group by GID
func DeleteGroupHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gid := r.Context().Value(ReportGIDVar).(string)
		keys := GetKeysByGID(gid)
		if len(keys) <= 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(map[string][]string{"deleted": keys}); err != nil {
			Log.Printf("Could not delete group (%v): %v", gid, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// remove a single file by its key
func DeleteReportHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		k := r.Context().Value(ReportKeyVar).(string) // if we fail to convert to string, we have a big problem -> let recoverer middleware deal
		if ok := Store.Has(k); !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		} else if err := Store.Erase(k); err != nil {
			Log.Printf("could not erase entry by key (=%v): %v", k, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
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
		w.WriteHeader(statusCode)
		return
	case http.StatusPartialContent:
		fallthrough
	case http.StatusOK:
		if err := json.NewEncoder(w).Encode(reports); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}