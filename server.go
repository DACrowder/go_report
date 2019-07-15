package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	chiCors "github.com/go-chi/cors"
	"github.com/peterbourgon/diskv"
	"log"
	"net/http"
	"os"
	"strconv"
)

var Log *log.Logger
var Cfg Config
var Store *diskv.Diskv

func init() {
	var cfgPath string
	var err error
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.StringVar(&cfgPath, "c", "config.json", "Path to config.json")
	flag.Parse()
	if Cfg, err = ReadConfig(cfgPath); err != nil {
		_, err = fmt.Fprintf(os.Stderr, "Server failed to read configuration file: %+v", err.Error())
		panic(err)
	}
	if Log, err = StartLogger(Cfg.LogFile); err != nil {
		_, err = fmt.Fprintf(os.Stderr, "Server failed to start logger: %+v", err.Error())
		if err != nil {
			panic(err)
		}
	}
}

const (
	ReportKeyVar           = "reportsKey"
	ReportGIDVar           = "reportsGID"
	ReportSeverityLevelVar = "severityLevel"
)

func main() {
	Store = CreateStore(Cfg.StorageRoot)
	Log.Println("Storage initialized.")
	r := chi.NewRouter()
	// init cors middleware
	cors := chiCors.New(chiCors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
	// add middlewares
	r.Use(cors.Handler)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)
	r.Use(middleware.Logger)
	// Create router
	r.Route("/report", func(r chi.Router) {
		r.Get("/", GetAllHandler())
		r.Post("/", PostHandler())
		r.Route("/group/{"+ReportGIDVar+"}", func(r chi.Router) {
			r.Use(ReportGroupCtx)
			r.Get("/", GetGroupHandler())
			r.Delete("/", DeleteGroupHandler())
		})
		r.Route("/severity/{"+ReportSeverityLevelVar+"}", func(r chi.Router) {
			r.Use(ReportSeverityCtx)
			r.Get("/", GetBatchByTypeHandler())
		})
		r.Route("/key/{"+ReportKeyVar+"}", func(r chi.Router) {
			r.Use(ReportKeyCtx)
			r.Get("/", GetReportHandler())
			r.Delete("/", DeleteReportHandler())
		})
	})
	Log.Println("Router created, starting server...")
	// Start serving
	if err := http.ListenAndServe(":"+strconv.Itoa(Cfg.Port), r); err != nil {
		if err != http.ErrServerClosed {
			panic(err)
		} else {
			Log.Println("server shutdown complete.")
		}
	}
}

func ReportGroupCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rGID := chi.URLParam(r, ReportGIDVar)
		if rGID == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx := context.WithValue(r.Context(), ReportGIDVar, rGID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ReportKeyCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rKey := chi.URLParam(r, ReportKeyVar)
		if rKey == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx := context.WithValue(r.Context(), ReportKeyVar, rKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ReportSeverityCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slvl := chi.URLParam(r, ReportSeverityLevelVar)
		if slvl == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ctx := context.WithValue(r.Context(), ReportSeverityLevelVar, slvl)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
