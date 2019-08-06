package auth

import (
	"encoding/json"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
)

type Secrets struct {
	MSSCertsFile string `json:"jwtMSSCertsFile" paramName:"MSS_CERTS_LIST,list"`
	JWTKey       string `json:"jwtKey" paramName:"JWT_SECRET_KEY,secret"`
}

//ReadConfig reads a _secrets.json file into a Config struct
func ReadSecrets(fp string) (s Secrets, err error) {
	shh := Secrets{}
	if ok := filepath.IsAbs(fp); !ok {
		return shh, errors.New("path to secrets must be an absolute path")
	}
	fd, err := os.Open(fp)
	if err != nil {
		return shh, err
	}
	if err = json.NewDecoder(fd).Decode(&shh); err != nil {
		return shh, err
	}
	return shh, fd.Close()
}
