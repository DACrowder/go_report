package main

import (
	"flag"
	"fmt"
	"go_report/auth"
	"go_report/failure"
	"go_report/gh"
	"go_report/report"
	"log"
	"net/http"
	"os"
	"strconv"
)

var (
	logger *log.Logger
)

func main() {
	var (
		err error
		cfg Config
	)
	cfgPath := parseCL()
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
	logger.Print("logger started.")

	failure.Init(logger)
	logger.Println("initialized failure handler.")

	shh, err := auth.ReadSecrets(cfg.SecretsPath)
	if err != nil {
		logger.Fatalf("failed to read secrets: %v", err.Error())
		return
	}
	logger.Println("Initialized auth subservice.")

	ghs, err := gh.NewFromFile(cfg.GHServicePath)
	if err != nil {
		logger.Fatalf("could not read gh service configuration file: %v", err.Error())
		return
	}
	logger.Println("Initialized github subservice.")

	r := NewRouter(
		report.NewStore(cfg.StorageRoot, logger),
		auth.New(shh, ghs, logger),
		ghs,
	)
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

func parseCL() (cfgPath string) {
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.StringVar(&cfgPath, "c", "config.json", "Path to config.json")
	flag.Parse()
	return cfgPath
}
