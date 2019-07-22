package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"go_report/failure"
	"go_report/report"
	"io/ioutil"
	"net/http"
)

type RequestContextKey string
const (
	// -------- CONTEXT KEYS --------
	// These constants are the context() keys to retrieve the Key/GID/Severity/Report from the request context
	// They are used by the middleware to place values in context predictably
	// Likewise, the handlers use them to retrieve values via Context().Values(ReportVar) -> value
	ReportKeyVar           RequestContextKey = "reportsKey"
	ReportGIDVar           RequestContextKey = "reportsGID"
	ReportSeverityLevelVar RequestContextKey = "severityLevel"
	ReportCtxVar           RequestContextKey = "reportFromRequestBody"
)

// ReportCtx returns a middleware which adds a *Report to POST request context
func ReportCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r.WithContext(r.Context()))
			return
		}
		b, _ := ioutil.ReadAll(r.Body)
		fmt.Println(string(b))
		bb := bytes.NewBuffer(b)
		rpt := new(report.Instance)
		if err := json.NewDecoder(bb).Decode(rpt); err != nil {
			failure.Fail(w, failure.New(err, http.StatusBadRequest, "Could not decode Report from request body"))
			return
		}
		ctx := context.WithValue(r.Context(), string(ReportCtxVar), *rpt)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ReportGroupCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rGID := chi.URLParam(r, string(ReportGIDVar))
		if rGID == "" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		ctx := context.WithValue(r.Context(), string(ReportGIDVar), rGID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ReportKeyCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rKey := chi.URLParam(r, string(ReportKeyVar))
		if rKey == "" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		ctx := context.WithValue(r.Context(), string(ReportKeyVar), rKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ReportSeverityCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slvl := chi.URLParam(r, string(ReportSeverityLevelVar))
		if slvl == "" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		ctx := context.WithValue(r.Context(), string(ReportSeverityLevelVar), slvl)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
