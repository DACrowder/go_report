package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/peterbourgon/diskv"
	"net/http"
	"strings"
)

type Report struct {
	// The report creation request will contain these three fields
	GID      string                 `json:"gid"`
	Severity ReportType             `json:"severity"`
	Content  map[string]interface{} `json:"content"`
	key      string
}

// For sending responses to queries regarding report creation confirmation, and lookup help
type ReportReceipt struct {
	GID      string // the id of the report (directory)
	FileName string // the filename (string representation of its md5 hash)
}

func CreateStore(root string) *diskv.Diskv {
	transformer := func(key string) *diskv.PathKey {
		path := strings.Split(key, "/")
		last := len(path) - 1
		return &diskv.PathKey{
			Path:     path[:last],
			FileName: path[last],
		}
	}
	invTransformer := func(pathKey *diskv.PathKey) (key string) {
		return strings.Join(pathKey.Path, "/") + "/" + pathKey.FileName
	}
	return diskv.New(
		diskv.Options{
			BasePath:          root,
			AdvancedTransform: transformer,
			InverseTransform:  invTransformer,
			CacheSizeMax:      1024 * 1024,
		})
}

func getMD5HashString(text []byte) string {
	hasher := md5.New()
	hasher.Write(text)
	return hex.EncodeToString(hasher.Sum(nil))
}

func CreateEntry(r Report) (key string, reportJson []byte, err error) {
	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(r)
	if err != nil {
		logger.Fatal(err)
		return "", nil, err
	}
	reportJson = buf.Bytes()
	h := getMD5HashString(reportJson)
	key = strings.Join(
		[]string{r.GID, h + ".txt"},
		"/",
	)
	return key, reportJson, nil
}

func GetKeysByGID(gid string) []string {
	keysChannel := store.KeysPrefix(gid, nil)
	keys := make([]string, 0, 16)
	for k := range keysChannel {
		keys = append(keys, k)
	}
	return keys
}

func GetReportsWithKeys(keys ...string) (reports map[string]Report, statusCode int) {
	reports, statusCode = map[string]Report{}, http.StatusOK
	errs, buf := make([]error, 0, 16), new(bytes.Buffer)
	dec := json.NewDecoder(buf)
	for _, k := range keys {
		rbytes, err := store.Read(k)
		if err != nil {
			logger.Printf("failed to retrieve report (k=%v): %v", k, err.Error())
			errs = append(errs, err)
			continue
		}
		buf.Write(rbytes) // will not fail without panic for ENOMEM || ErrWriteTooLarge https://golang.org/pkg/bytes/#Buffer.Write thus ignoring error is ok!
		rprt := new(Report)
		if err := dec.Decode(rprt); err != nil {
			logger.Printf("failed to decode entry (k=%v): %v", k, err.Error())
			errs = append(errs, err)
			continue
		}
		reports[k] = *rprt
		buf.Reset()
	}
	if len(errs) > 0 {
		if len(errs) < len(reports) {
			return reports, http.StatusPartialContent
		} else {
			return nil, http.StatusInternalServerError // all retrievals failed -> something is wrong.
		}
	}
	return reports, statusCode
}
