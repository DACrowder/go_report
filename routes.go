package main

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	chiCors "github.com/go-chi/cors"
	"go_report/auth"
	"go_report/gh"
	"go_report/report"
)

func NewRouter(s *report.Store, a *auth.Service, ghs *gh.Service) *chi.Mux {
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
				r.Post("/", PostHandler(s, ghs))
			})
			r.Group(func(r chi.Router) {
				r.Use(a.OnlyDevsAuthenticate)
				// Require GitHub Repository access scope (developers only)
				r.Get("/", GetAllHandler(s))
				r.Route("/group/{"+string(ReportGIDVar)+"}", func(r chi.Router) {
					r.Use(ReportGroupCtx)
					r.Get("/", GetGroupHandler(s))
					r.Delete("/", DeleteGroupHandler(s))
				})
				r.Route("/severity/{"+string(ReportSeverityLevelVar)+"}", func(r chi.Router) {
					r.Use(ReportSeverityCtx)
					r.Get("/", GetBatchByTypeHandler(s))
				})
				r.Route("/key/{"+string(ReportKeyVar)+"}", func(r chi.Router) {
					r.Use(ReportKeyCtx)
					r.Get("/", GetReportHandler(s))
					r.Delete("/", DeleteReportHandler(s))
				})
			})
		})
	})
	return r
}
