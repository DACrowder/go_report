package main

import (
	"flag"
	"fmt"

	"github.com/go-chi/chi"

	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/middleware"
	chiCors "github.com/go-chi/cors"
	"github.com/peterbourgon/diskv"
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

// Typing these makes more headaches than it solves.
// These constants are the context() keys to retrieve the Key/GID/Severity/Report from the request context
// They are used by the middleware to place values in context predictably
// Likewise, the handlers use them to retrieve values via Context().Values(ReportVar) -> value
const (
	ReportKeyVar           = "reportsKey"
	ReportGIDVar           = "reportsGID"
	ReportSeverityLevelVar = "severityLevel"
	ReportCtxVar           = "reportFromRequestBody"
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

	// Private routes for actual service
	r.Group(func(r chi.Router) {
		r.Route("/report", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				// Application authorization scheme
				r.Use(ReportCtx)
				r.Post("/", PostHandler())
			})
			r.Group(func(r chi.Router) {
				// Require GitHub Repository access scope (developers only)
				r.Get("/", GetAllHandler())
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
