package main

import (
	"go_report/domain"
	"log"
	"net/http"
	"strconv"

	aws "github.com/aws/aws-sdk-go/aws/session"
)

func main() {
	sesh, err := aws.NewSessionWithOptions(aws.Options{
		SharedConfigState: aws.SharedConfigEnable,
	})
	if err != nil {
		panic(err)
		return
	}
	cfg, shh, ghs, store, logger, err := LoadFromParamStore(sesh)
	if err != nil {
		if logger != nil {
			log.Fatal(err.Error())
		} else {
			panic(err)
		}
	}
	ict, err := strconv.Atoi(cfg.IssueCreationThreshold)
	if err != nil {
		ict = domain.DisableIssueCreation// default to disabling issue creation if non-int passed
	}
	r := NewRouter(ict, store, shh, ghs, logger)
	logger.Println("Router created, starting server...")

	// Start serving
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		if err != http.ErrServerClosed {
			logger.Panic(err)
		} else {
			logger.Println("server shutdown complete.")
		}
	}
}
