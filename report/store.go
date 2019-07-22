package report

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/peterbourgon/diskv"
	"log"
	"net/http"
	"strings"
)

type Store struct {
	*diskv.Diskv
	log *log.Logger
}

func NewStore(rootDir string, logger *log.Logger) *Store {
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
	return &Store{
		log: logger,
		Diskv: diskv.New(
			diskv.Options{
				BasePath:          rootDir,
				AdvancedTransform: transformer,
				InverseTransform:  invTransformer,
				CacheSizeMax:      1024 * 1024,
			}),
	}
}

func getMD5HashString(text []byte) string {
	hasher := md5.New()
	hasher.Write(text)
	return hex.EncodeToString(hasher.Sum(nil))
}

func (s *Store) CreateEntry(r Instance) (key string, reportJson []byte, err error) {
	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(r)
	if err != nil {
		s.log.Fatal(err)
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

func (s *Store) GetKeysByGID(gid string) []string {
	keysChannel := s.KeysPrefix(gid, nil)
	keys := make([]string, 0, 16)
	for k := range keysChannel {
		keys = append(keys, k)
	}
	return keys
}

func (s *Store) GetReportsWithKeys(keys ...string) (reports map[string]Instance, statusCode int) {
	reports, statusCode = map[string]Instance{}, http.StatusOK
	errs, buf := make([]error, 0, 16), new(bytes.Buffer)
	dec := json.NewDecoder(buf)
	for _, k := range keys {
		rbytes, err := s.Read(k)
		if err != nil {
			s.log.Printf("failed to retrieve report (k=%v): %v", k, err.Error())
			errs = append(errs, err)
			continue
		}
		buf.Write(rbytes) // will not fail without panic for ENOMEM || ErrWriteTooLarge https://golang.org/pkg/bytes/#Buffer.Write thus ignoring error is ok!
		rprt := new(Instance)
		if err := dec.Decode(rprt); err != nil {
			s.log.Printf("failed to decode entry (k=%v): %v", k, err.Error())
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
