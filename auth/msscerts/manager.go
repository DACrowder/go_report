package msscerts

import (
	"bytes"
	"github.com/pkg/errors"
	"go_report/failure"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

var (
	initOnce            sync.Once
	man                 *Manager
	ErrWriteCertsFailed = failure.New(
		errors.New("Failed to write updated certificates file"),
		http.StatusInternalServerError,
		"",
	)
)

type Manager struct {
	*log.Logger
	lock         sync.RWMutex
	MSSCertsFile string `json:"jwtMSSCertsFile"`
}

func Init(certsFilePath string, logger *log.Logger) {
	initOnce.Do(func() {
		man = &Manager{
			lock:         sync.RWMutex{},
			Logger:       logger,
			MSSCertsFile: certsFilePath,
		}
	})
}

func GetManager() *Manager {
	return man
}

func (man *Manager) Verify(cert string) (bool, error) {
	cert = strings.TrimSpace(cert)
	ok := false
	b, err := man.read()
	if err != nil {
		return false, err
	}
	for _, ln := range bytes.Split(b, []byte("\n")) {
		ln = bytes.TrimSpace(ln)
		if cert == string(ln) {
			ok = true
		}
	}
	return ok, nil
}

func (man *Manager) AddCertificate(cert string) error {
	man.lock.Lock()
	defer man.lock.Unlock()
	f, err := os.OpenFile(man.MSSCertsFile, os.O_APPEND|os.O_WRONLY, 0644)
	defer func() {
		if err := f.Close(); err != nil {
			man.Printf("failed to close certificates file: %v", err.Error())
		}
	}()
	if err != nil {
		return errors.Wrap(err, "failed to open cert registry")
	}
	if _, err := f.WriteString("\n" + cert + "\n"); err != nil {
		return errors.Wrap(err, "failed to write new cert to registry")
	}
	return nil
}

func (man Manager) RemoveCertificate(needle string) error {
	hs, err := man.read()
	if err != nil {
		return errors.Wrapf(err, "cannot remove %v failed to read cert registry", needle)
	}
	remain := make([][]byte, 0, 128)
	for _, ln := range bytes.Split(hs, []byte("\n")) {
		ln = bytes.TrimSpace(ln)
		if needle == string(ln) {
			continue // do not add to new haystack
		}
		remain = append(remain, ln)
	}
	if _, err := man.write(bytes.Join(remain, []byte("\n"))); err != nil {
		return errors.Wrap(err, "failed to write replacement cert registry")
	}
	return nil
}

func (man Manager) read() ([]byte, error) {
	man.lock.RLock()
	b, err := ioutil.ReadFile(man.MSSCertsFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not read certificates file")
	}
	man.lock.RUnlock()
	return b, nil
}

func (man Manager) write(certs []byte) (int, error) {
	man.lock.Lock()
	if err := ioutil.WriteFile(man.MSSCertsFile, certs, 0660); err != nil {
		return 0, ErrWriteCertsFailed
	}
	man.lock.Unlock()
	return len(certs), nil
}
