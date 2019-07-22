package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Port          int    `json:"port"`                // Port on which to connect the server
	LogFile       string `json:"logFile"`             // File location for log
	SecretsPath   string `json:"secretsPath"`         // path to file containing secrets
	GHServicePath string `json:"ghServiceConfigPath"` // path to file containing github repo and secrets information
	StorageRoot   string `json:"storageRootDir"`
}

//ReadConfig reads a cfg.json file into a Config struct
func ReadConfig(fp string) (c Config, err error) {
	cfg := Config{}
	fd, err := os.Open(fp)
	if err != nil {
		return cfg, err
	}
	if err = json.NewDecoder(fd).Decode(&cfg); err != nil {
		return cfg, err
	}
	if err := fd.Close(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func StartLogger(fp string) (*log.Logger, error) {
	var logger *log.Logger
	if strings.ToLower(fp) == "stderr" || fp == "2" {
		logger = log.New(os.Stderr, "Status: ", log.LstdFlags|log.Lshortfile)
	} else if strings.ToLower(fp) == "stdout" || fp == "1" {
		logger = log.New(os.Stdout, "Status: ", log.LstdFlags|log.Lshortfile)
	} else {
		logFile, err := os.OpenFile(fp, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		logger = log.New(logFile, filepath.Base(fp)+": ", log.LstdFlags|log.Lshortfile)
	}
	logger.Println("Logger started successfully.")
	return logger, nil
}
