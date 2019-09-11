package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	awsesh "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/pkg/errors"
	"go_report/auth"
	"go_report/gh"
	"go_report/store/dynamo"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type Config struct {
	Port    string `json:"port" paramName:"BRS_PORT" paramDefault:"8080"`       // Port on which to connect the server
	LogFile string `json:"logFile" paramName:"BRS_LOGFILE" paramDefault:"stderr"` // File location for log
	TableName string `json:"tableName" paramName:"TABLE_NAME" paramDefault:"BugReports"`
	IssueCreationThreshold string `json:"issueCreationThreshold" paramName:"ISSUE_CREATION_THRESHOLD" paramDefault:"x"`
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
	var mm struct { // middleman between strings & typed gh.Secrets
		PrivateKeyFile string `json:"ghPrivateKeyFile" paramName:"GH_APP_KEY,secret"` // pem encoded rsa key
		AppID          string `json:"ghAppID" paramName:"GH_APP_ID,secret"`
		InstallID      string `json:"ghInstallID" paramName:"GH_INSTALL_ID,secret"`
		//WebhookSecret  string `json:"ghWebhookSecret" paramName:"GH_WEBHOOK,secret"` // not needed
		ClientID     string `json:"ghClientID" paramName:"GH_CLIENT_ID,secret"`
		ClientSecret string `json:"ghClientSecret" paramName:"GH_CLIENT_SECRET,secret"`
	}
	if err := LoadParams(svc, &mm); err != nil {
		return nil, err
	}
	ghshh := gh.Secrets{
		PrivateKeyFile: mm.PrivateKeyFile,
		ClientID:       mm.ClientID,
		ClientSecret:   mm.ClientSecret,
	}
	i, err := strconv.Atoi(mm.AppID)
	if err != nil {
		return nil, err
	}
	ghshh.AppID = i
	i, err = strconv.Atoi(mm.InstallID)
	if err != nil {
		return nil, err
	}
	ghshh.InstallID = i
	return gh.New(repo, ghshh), nil
}

func startAuthService(svc *ssm.SSM, store *dynamo.Store, ghs *gh.Service, logger *log.Logger) (*auth.Service, error) {
	var shh auth.Secrets
	if err := LoadParams(svc, &shh); err != nil {
		return nil, err
	}
	return auth.New(store, shh, ghs, logger), nil
}

func DescribeParametersAvailable(svc *ssm.SSM) {
	dpo, err := svc.DescribeParameters(&ssm.DescribeParametersInput{MaxResults: aws.Int64(15)})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Found Parameters: %+v", dpo.String())
}

func LoadFromParamStore(sesh *awsesh.Session) (cfg Config, auth *auth.Service, ghs *gh.Service, store *dynamo.Store, logger *log.Logger, err error) {
	svc := ssm.New(sesh)

	//DescribeParametersAvailable(svc)

	if err = LoadParams(svc, &cfg); err != nil {
		return
	}
	if logger, err = StartLogger(cfg.LogFile); err != nil {
		return
	}
	store = dynamo.New(sesh, cfg.TableName, logger)
	fmt.Printf("%+v %+v", store, cfg)
	if ghs, err = startGHService(svc); err != nil {
		return
	}
	if auth, err = startAuthService(svc, store, ghs, logger); err != nil {
		return
	}
	return
}

// structs with paramName & json tags can be autofilled from param store values
// to avoid writing own reflection for setting values, we require a json tag for
//  leveraging the stdlib unmarshal.
func LoadParams(svc *ssm.SSM, v interface{}) (err error) {
	const tagName = "paramName"
	const defaultValueTagName = "paramDefault"
	if ok := reflect.ValueOf(v).Kind() == reflect.Ptr; !ok {
		return errors.New("LoadParams requires a pointer to a tagged destination structure")
	}

	// Iterate over all available fields and read the tag value
	t, temp := reflect.ValueOf(v).Elem(), map[string]interface{}{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Type().Field(i)
		// get the json and param tags
		tag := field.Tag.Get(tagName)
		jt := field.Tag.Get("json")
		if jt == "" || tag == "" {
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
			Name:           aws.String(pn),
			WithDecryption: aws.Bool(isSecret),
		})
		// if err, set default value if provided; else fail with error
		if err != nil {
			dv := field.Tag.Get(defaultValueTagName)
			if dv == "" {
				fmt.Printf("param: %+v\tsecret: %+v", pn, isSecret)
				return err
			}
			// sets default value of param
			param = &ssm.GetParameterOutput{
				Parameter: &ssm.Parameter{Value: aws.String(dv)},
			}
		}
		// Now add the value from the param store, to the intermediate map
		if strings.Contains(jt, ",") {
			jt = strings.Split(jt, ",")[0]
		}
		temp[jt] = *param.Parameter.Value
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
