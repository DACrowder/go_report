package main

import (
	"encoding/json"
	"os"
)

type Secrets struct {
	GHPrivateKeyFile string `json:"ghPrivateKeyFile"` // pem encoded rsa key
	GHAppID          int    `json:"ghAppID"`
	GHInstallID      int    `json:"ghInstallID"`
	GHWebhookSecret  string `json:"ghWebhookSecret"`
	GHClientID       string `json:"ghClientID"`
	GHClientSecret   string `json:"ghClientSecret"`
}

//ReadConfig reads a _secrets.json file into a Config struct
func ReadSecrets(fp string) (s Secrets, err error) {
	shh := Secrets{}
	fd, err := os.Open(fp)
	if err != nil {
		return shh, err
	}
	if err = json.NewDecoder(fd).Decode(&shh); err != nil {
		return shh, err
	}
	return shh, fd.Close()
}
