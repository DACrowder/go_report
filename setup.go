package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	Port             int    `json:"port"` // Port on which to connect the server
	DBConnection 	 string `json:"dbConnectionString"`
	LogFile          string `json:"logFile"`     // File location for log
	GithubToken 	 string `json:"githubToken"`
	StorageRoot 	 string `json:"storageRootDir"`
}

//ReadConfig reads a config.json file into a Config struct
func ReadConfig(fp string) (c Config, err error) {
	cfg := Config{}
	fd, err := os.Open(fp)
	if err != nil {
		return cfg, err
	}
	if err = json.NewDecoder(fd).Decode(&cfg); err != nil {
		return cfg, err
	}
	return cfg, fd.Close()
}

func StartLogger(fp string) (*log.Logger, error) {
	logFile, err := os.OpenFile(fp, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	logger := log.New(logFile, filepath.Base(fp)+": ", log.LstdFlags)
	logger.Println("Logger started successfully.")
	return logger, nil
}

func StartDB(cfg Config) error {
	return nil
}