package main

import (
	"flag"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth"

	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/middleware"
	chiCors "github.com/go-chi/cors"
	"github.com/peterbourgon/diskv"
)

var (
	cfg     Config
	logger  *log.Logger
	store   *diskv.Diskv
	jwtAuth *jwtauth.JWTAuth
)

func init() {
	var cfgPath string
	var err error
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.StringVar(&cfgPath, "c", "config.json", "Path to config.json")
	flag.Parse()
	if cfg, err = ReadConfig(cfgPath); err != nil {
		_, err = fmt.Fprintf(os.Stderr, "Server failed to read configuration file: %+v", err.Error())
		panic(err)
	}
	if logger, err = StartLogger(cfg.LogFile); err != nil {
		_, err = fmt.Fprintf(os.Stderr, "Server failed to start logger: %+v", err.Error())
		if err != nil {
			panic(err)
		}
	}
	mssCertsMan = NewMSSCertificateManager()
}

func main() {
	jwtAuth = jwtauth.New(jwt.SigningMethodHS512.Name, []byte(cfg.Secrets.JWTKey), nil)
	store = CreateStore(cfg.StorageRoot)
	logger.Println("Storage initialized.")

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

	// public route for getting jwt
	r.Group(func(r chi.Router) {
		r.Route("/token", func(r chi.Router) {
			r.Put("/", TokenExchangeHandler())
			r.Post("/", TokenExchangeHandler())
		})
	})

	// Private routes for actual service -- requires JWT
	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(jwtAuth))
		r.Use(Authenticator)
		r.Route("/report", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				// Application authorization scheme
				r.Use(ReportCtx)
				r.Post("/", PostHandler())
			})
			r.Group(func(r chi.Router) {
				r.Use(OnlyDevsAuthenticator)
				// Require GitHub Repository access scope (developers only)
				r.Get("/", GetAllHandler())
				r.Route("/group/{"+string(ReportGIDVar)+"}", func(r chi.Router) {
					r.Use(ReportGroupCtx)
					r.Get("/", GetGroupHandler())
					r.Delete("/", DeleteGroupHandler())
				})
				r.Route("/severity/{"+string(ReportSeverityLevelVar)+"}", func(r chi.Router) {
					r.Use(ReportSeverityCtx)
					r.Get("/", GetBatchByTypeHandler())
				})
				r.Route("/key/{"+string(ReportKeyVar)+"}", func(r chi.Router) {
					r.Use(ReportKeyCtx)
					r.Get("/", GetReportHandler())
					r.Delete("/", DeleteReportHandler())
				})
			})
			r.Route("/certificate/{" + string(MSSCertificateCtxVar) + "}", func (r chi.Router) {
				r.Use(OnlyDevsAuthenticator)
				r.Post("/", AddCertificateHandler())
				r.Delete("/", RemoveCertificateHandler())
			})
		})
	})

	logger.Println("Router created, starting server...")
	// Start serving
	if err := http.ListenAndServe(":"+strconv.Itoa(cfg.Port), r); err != nil {
		if err != http.ErrServerClosed {
			logger.Panic(err)
		} else {
			logger.Println("server shutdown complete.")
		}
	}
}
