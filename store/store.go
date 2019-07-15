package store

import (
	"crypto"
	"errors"
	"os"
	"strings"
	"time"
)

var ErrNotDirectory = errors.New("A file exists at the given path, but it does not point to a directory.")
var ErrIsDirectory = errors.New("A directory exists at the path given for storing the index file")

type index struct {
	ReportReceipt
	hash     crypto.Hash // must correspond to filename once confirmed
	toa      time.Time
}

type Store struct {
	RootPath string
	IndexIDMapFile string // name of index storage file -- to save/read from
	IDIndexMap map[string][]index // in memory recollection
}

// Initialize the storage dir without clobbering an existing one
func New(rootDir string, indexFile string) (Store, error) {
	store := Store{}
	fi, err := os.Stat(rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(rootDir, 0644); err != nil {
				return store, err
			}
		} else {
			return store, err
		}
	}
	if ok := fi.IsDir(); !ok {
		return store, ErrNotDirectory
	}
	store.RootPath = rootDir
	// Create index file if does not exist
	fp := strings.Join([]string{store.RootPath, indexFile}, os.PathSeparator)
	fi, err = os.Stat()
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.Create(fp); err != nil {
				return store, err
			}
		} else {
			return store, err
		}
	}
	if fi.IsDir() {
		return store, ErrIsDirectory
	}
	store.IndexIDMapFile = fp
	return store, nil
}

// Add report to storage
func (s Store) Add(r Report) (ReportReceipt, error) {

}