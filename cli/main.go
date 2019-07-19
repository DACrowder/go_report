package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/kr/pretty"
	"golang.org/x/oauth2"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const baseurl = "http://127.0.0.1:3333"

var (
	 key, gid, slvl, ghUser, ghToken, jwt, cert string
	stype                                = -1
	ALL = false
	delReq                          = false
	err                                  error
)

func init() {
	flag.Usage = func() {
		flag.PrintDefaults()
		fmt.Println("Precedence of report qualifiers: key > group > severity = type")
		fmt.Println("i.e. if group and severity given, severity will be ignored")
	}

	flag.BoolVar(&ALL, "ALL", false, "must be present to get/delete all records")

	// short flags
	flag.StringVar(&ghUser, "u", "", "your github username (for token requests)")
	flag.StringVar(&ghToken, "o", "", "your github oauth token (for token requests)")
	flag.StringVar(&jwt, "j", "", "the report server auth jwt to use")
	flag.StringVar(&key, "k", "", "the report key to lookup")
	flag.StringVar(&gid, "g", "", "the report group id to lookup")
	flag.StringVar(&slvl, "s", "", "the report severity string (bug | unknown | crash)")
	flag.IntVar(&stype, "t", -1, "the report type (severity level in numeric form: 0|1|2)")
	flag.StringVar(&cert, "c", "", "the mss application certificate to add/remove")

	// corresponding long flags
	flag.StringVar(&ghUser, "user", "", "your github username (for token requests)")
	flag.StringVar(&ghToken, "oauth", "", "your github oauth token (for token requests)")
	flag.StringVar(&jwt, "jwt", "", "the report server auth jwt to use")
	flag.StringVar(&key, "key", "", "the report key to lookup")
	flag.StringVar(&gid, "gid", "", "the report group id to lookup")
	flag.StringVar(&slvl, "severity", "", "the report severity string (bug | unknown | crash)")
	flag.IntVar(&stype, "type", -1, "the report type (severity level in numeric form: 0|1|2)")
	flag.BoolVar(&delReq, "Delete", false, "when set, the reports found will be deleted")
	flag.StringVar(&cert, "certificate", "", "the mss application certificate to add/remove")

	flag.Parse()
}

func main() {
	if (ghUser == "" || ghToken == "") && jwt == "" {
		_, _ = fmt.Fprint(os.Stderr, "Cannot make request: no auth token (nor github for retrieval) provided\n")
		flag.Usage()
		os.Exit(http.StatusUnauthorized)
		return
	}
	if jwt == "" {
		if jwt, err = getJWT(); err != nil {
			_ = fmt.Errorf("could not retrieve auth jwt: %v\n", err.Error())
			os.Exit(1)
			return
		}
		fmt.Printf("JWT: %v\n", jwt)
	}
	tc := oauth2.NewClient(
		context.Background(),
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: jwt}),
	)
	if cert != "" {
		if err := certRequest(tc); err != nil {
			_ = fmt.Errorf("could not complete certificate request: %v\n", err.Error())
			os.Exit(2)
			return
		}
	}
	url := url()
	if (url == "") {
		os.Exit(0)
		return
	}
	b, _ := body(map[string]interface{}{}) // did not give data thus no error possible
	req, err := http.NewRequest(method(),url, b)
	if err != nil {
		_ = fmt.Errorf("failed to create request: %v\n", err.Error())
		os.Exit(3)
		return
	}
	resp, err := tc.Do(req)
	if err != nil {
		_ = fmt.Errorf("request failed: %v\n", err.Error())
		os.Exit(4)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err.Error())
		}
	}()
	data := map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		_ = fmt.Errorf("Could not decode response from server: %v\n", err.Error())
		os.Exit(5)
		return
	}
	_, _ = pretty.Println(data)
}

// getJWT from server using github credentials (uname + oauth2 token)
func getJWT() (string, error) {
	body, err := body(map[string]interface{}{
		"ghUser":  ghUser,
		"ghToken": ghToken,
	})
	if err != nil {
		return "", err
	}
	resp, err := http.Post(baseurl+"/token", "application/json", body)
	if err != nil {
		return "", err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func certRequest(tc *http.Client) (err error) {
	var req *http.Request
	var expected int
	url := baseurl+"/certificate/"+cert+"/"
	if delReq {
		req, err = http.NewRequest(http.MethodDelete, url, nil)
		expected = http.StatusNoContent
	} else {
		req, err =  http.NewRequest(http.MethodPost, url, nil)
		expected = http.StatusCreated
	}
	if err != nil {
		return err
	}
	resp, err := tc.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != expected {
		return errors.New(fmt.Sprintf("unexpected response code %v in certificate response", resp.StatusCode))
	}
	return nil
}

func body(data map[string]interface{}) (io.Reader, error) {
	b := new(bytes.Buffer)
	if len(data) == 0 {
		return b, nil
	}
	err := json.NewEncoder(b).Encode(data)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func method() string {
	if delReq {
		return http.MethodDelete
	}
	return http.MethodGet
}

func url() string {
	url := baseurl + "/report"
	if key != "" {
		url += "/key/" + key + "/"
	} else if gid != "" {
		url += "/group/" + gid + "/"
	} else if slvl != "" {
		url += "/severity/" + slvl + "/"
	} else if stype >= 0 {
		url += "/severity/" + strconv.Itoa(stype) + "/"
	} else if ALL {
		url += "/"
	} else {
		return ""
	}
	return url
}
