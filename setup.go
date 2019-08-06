package main

import (
	"encoding/json"
	aws "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/pkg/errors"
	"go_report/auth"
	"go_report/gh"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type Config struct {
	Port          int    `json:"port" paramName:"BRS_PORT"`                // Port on which to connect the server
	LogFile       string `json:"logFile" paramName:"BRS_LOGFILE"`             // File location for log
	InProduction bool 	 `json:"inProduction"`
}

//ReadConfigFromFile reads a cfg.json file into a Config struct
func ReadConfigFromFile(fp string) (c Config, err error) {
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

func startGHService(svc *ssm.SSM) (*gh.Service, error) {
	var repo gh.Repo
	if err := LoadParams(svc, &repo); err != nil {
		return nil, err
	}
	var ghshh gh.Secrets
	if err := LoadParams(svc, &ghshh); err != nil {
		return nil, err
	}
	return gh.New(repo, ghshh), nil
}

func startAuthService(svc *ssm.SSM, ghs *gh.Service, logger *log.Logger) (*auth.Service, error) {
	var shh auth.Secrets
	if err := LoadParams(svc, &shh); err != nil {
		return nil, err
	}
	return auth.New(shh, ghs, logger), nil
}

func LoadFromParamStore(sesh *aws.Session) (cfg Config, auth *auth.Service, ghs *gh.Service, logger *log.Logger, err error) {
	svc := ssm.New(sesh)
	if err = LoadParams(svc, &cfg); err != nil {
		return
	}
	if logger, err = StartLogger(cfg.LogFile); err != nil {
		return
	}
	if ghs, err = startGHService(svc); err != nil {
		return
	}
	if auth, err = startAuthService(svc, ghs, logger); err != nil {
		return
	}
	return
}

// structs with paramName & json tags can be autofilled from param store values
// to avoid writing own reflection for setting values, we require a json tag for
//  leveraging the stdlib unmarshal.
func LoadParams(svc *ssm.SSM, v interface{}) (err error) {
	const tagName = "paramName"
	if ok := reflect.ValueOf(v).Kind() == reflect.Ptr; !ok {
		return errors.New("LoadParams requires a pointer to a tagged destination structure")
	}
	// Iterate over all available fields and read the tag value
	t, temp := reflect.TypeOf(v), map[string]interface{}{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Get the field tag value
		tag := field.Tag.Get(tagName)
		if tag == "" {
			continue
		}
		var pn string
		var isSecret bool
		if strings.Contains(tag, ",secret") {
			pn, isSecret = strings.Split(tag, ",")[0], true
		} else {
			pn, isSecret = tag, false
		}
		param, err := svc.GetParameter(&ssm.GetParameterInput{
			Name:           &pn,
			WithDecryption: &isSecret,
		})
		if err != nil {
			return err
		}
		// Now add the value from the param store, to the intermediate map
		tag = field.Tag.Get("json")
		if tag == "" {
			continue
		}
		if strings.Contains(tag, ",") {
			tag = strings.Split(tag, ",")[0]
		}
		temp[tag] = *param.Parameter.Value
	}
	b, err := json.Marshal(temp)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return err
	}
	return nil
}