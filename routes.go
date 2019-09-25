package main

import (
	"go_report/auth"
	"go_report/domain"
	"go_report/gh"
	"log"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	chiCors "github.com/go-chi/cors"
)

func NewRouter(issCreateThreshold int, s domain.Storer, a *auth.Service, ghs *gh.Service, logger *log.Logger) *chi.Mux {
	r := chi.NewRouter()
	// init cors middleware
	cors := chiCors.New(chiCors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-ReportType", "X-CSRF-Token"},
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

	r.Route("/ping", func(r chi.Router) {
		r.Get("/", PingHandler())
	})

	// public route for getting jwt
	r.Route("/token", func(r chi.Router) {
		r.Put("/", a.TokenExchangeHandler())
		r.Post("/", a.TokenExchangeHandler())
	})

	// private (dev only) routes for add/remove certificates
	r.Group(func(r chi.Router) {
		r.Use(a.Verifier)
		r.Use(a.Authenticate)
		r.Group(func(r chi.Router) {
			r.Use(a.MSSCertificateCtx)
			r.Use(a.OnlyDevsAuthenticate)
			r.Route("/certificate/{"+string(auth.CertCtxVar)+"}", func(r chi.Router) {
				r.Post("/", a.AddCertificateHandler())
				r.Delete("/", a.RemoveCertificateHandler())
			})
		})
	})

	// Private routes for actual service -- requires JWT
	r.Group(func(r chi.Router) {
		r.Use(a.Verifier)
		r.Use(a.Authenticate)
		r.Route("/report", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				// Application authorization scheme
				r.Use(ReportCtx)
				r.Post("/", PostHandler(issCreateThreshold, s, ghs, logger))
			})
			r.Group(func(r chi.Router) {
				r.Use(a.OnlyDevsAuthenticate)
				// Require GitHub Repository access scope (developers only)
				r.Get("/", GetAllHandler(s))
				r.Route("/group/{"+string(ReportGIDVar)+"}", func(r chi.Router) {
					r.Use(ReportGroupCtx)
					r.Get("/", GetGroupHandler(s))
				})
				r.Route("/group/{"+string(ReportGIDVar)+"}"+"/key/{"+string(ReportKeyVar)+"}", func(r chi.Router) { // "/group/{gid}/key/{key}/...
					r.Use(ReportGroupCtx)
					r.Use(ReportKeyCtx)
					r.Get("/", GetReportHandler(s))
					r.Delete("/", DeleteReportHandler(s))
				})
			})
		})
	})
	return r
}
