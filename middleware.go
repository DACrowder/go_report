package main

import (
	"context"
	"encoding/json"
	"github.com/go-chi/chi"
	"net/http"
)

// ReportCtx returns a middleware which adds a *Report to POST request context
func ReportCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r.WithContext(r.Context()))
			return
		}
		rpt := new(Report)
		if err := json.NewDecoder(r.Body).Decode(rpt); err != nil {
			logger.Printf("Could not decode Report from request body: %v", err.Error())
			err := json.NewEncoder(w).Encode(
				map[string]string{"error": "No malformed report json in request body"},
			)
			if err != nil {
				logger.Printf("Error occurred while responding to client with err: malformed request body, in ReportCtx: %v", err.Error())
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), ReportCtxVar, *rpt)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ReportGroupCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rGID := chi.URLParam(r, string(ReportGIDVar))
		if rGID == "" {
			w.WriteHeader(http.StatusNotFound)
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
			w.WriteHeader(http.StatusNotFound)
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
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx := context.WithValue(r.Context(), ReportSeverityLevelVar, slvl)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
