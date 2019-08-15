package msscerts

import (
	"crypto/md5"
	"encoding/hex"
	"go_report/domain"
	"go_report/failure"
	"go_report/store/dynamo"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

var (
	initOnce            sync.Once
	man                 *Manager
	ErrWriteCertsFailed = failure.New(
		errors.New("Failed to write updated certificates"),
		http.StatusInternalServerError,
		"",
	)
)

const MssCertificateGid = "MSS_CERTIFICATE"

type Manager struct {
	*log.Logger
	*dynamo.Store
	lock sync.RWMutex
}

func Init(store *dynamo.Store, logger *log.Logger) {
	initOnce.Do(func() {
		man = &Manager{
			lock:   sync.RWMutex{},
			Logger: logger,
			Store:  store,
		}
	})
}

func GetManager() *Manager {
	return man
}

func (man *Manager) Verify(cert string) (bool, error) {
	man.lock.RLock()
	defer man.lock.RUnlock()
	cert = strings.TrimSpace(cert)
	// The query for the store
	r, err := man.Store.Select(domain.Receipt{GID: MssCertificateGid, Key: getMD5HashString([]byte(cert))})
	if err != nil {
		return false, errors.Wrap(err, "Could not retrieve entry from database")
	}
	return r != nil, nil
}

func (man *Manager) AddCertificate(cert string) error {
	man.lock.Lock()
	defer man.lock.Unlock()
	_, err := man.NewEntry(domain.Report{
		GID: MssCertificateGid,
		Key: getMD5HashString([]byte(cert)),
		Content: map[string]interface{}{"value":cert},
	})
	if err != nil {
		return errors.Wrap(err, "Could not write certificate to database")
	}
	return nil
}

func (man Manager) RemoveCertificate(needle string) error {
	man.lock.Lock()
	defer man.lock.Unlock()
	err := man.Store.RemoveEntry(domain.Receipt{GID: MssCertificateGid, Key: getMD5HashString([]byte(needle))})
	if err != nil {
		return errors.Wrap(err, "Could not remove certificate due to error")
	}
	return nil
}

func getMD5HashString(text []byte) string {
	hasher := md5.New()
	hasher.Write(text)
	return hex.EncodeToString(hasher.Sum(nil))
}