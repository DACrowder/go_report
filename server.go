package main

import (
	"flag"
	"fmt"
	"github.com/go-chi/chi"
	chiCors "github.com/go-chi/cors"
	"log"
	"net/http"
	"os"
)

var Log *log.Logger
var Cfg Config

func init()  {
	var cfgPath string
	var err error
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.StringVar(&cfgPath, "c", "stderr", "Path to config.json")
	flag.Parse()
	if Cfg, err = ReadConfig(cfgPath); err != nil {
		_, err = fmt.Fprintf(os.Stderr, "Server failed to read configuration file: %+v", err.Error())
		panic(err)
	}
	if Log, err = StartLogger(Cfg.LogFile); err != nil {
		_, err = fmt.Fprintf(os.Stderr, "Server failed to start logger: %+v", err.Error())
		if err != nil{
			panic(err)
		}
	}
}

func main() {
	r := chi.NewRouter()

	cors := chiCors.New(chiCors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
	r.Use(cors.Handler)

	r.Route("/report/", func(r chi.Router) {
		r.Get("/", GetHandler())
		r.Post("/", PostHandler())
		r.Put("/", PostHandler())
		r.Delete("/", DeleteHandler())
	})

	if err := http.ListenAndServe(":3000", r); err != nil {
		panic(err)
	}
}


