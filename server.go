package main

import (
	aws "github.com/aws/aws-sdk-go/aws/session"
	"log"
	"net/http"
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

	r := NewRouter(store, shh, ghs, logger)
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
