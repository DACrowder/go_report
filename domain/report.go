package domain

import "strings"

type ReportType int

type Storer interface {
	NewEntry(r Report) (Receipt, error)                                        // Create a new entry in the store, return receipt
	Select(lookup Receipt) (*Report, error)                                      // Select one record by its key
	SelectAll() ([]Report, error)
	SelectGroup(gid string) ([]Report, error)
	RemoveEntry(lookup Receipt) error                                              // Erase a record from the store, or a group of records by GID
}

const (
	UnknownType ReportType = iota
	BugType
	CrashType
)

type Report struct {
	// The report creation request will contain these three fields
	GID      string                 `json:"gid"`
	Severity ReportType             `json:"severity"`
	Content  map[string]interface{} `json:"content"`
	Key      string `json:"key"`
}

// For sending responses to queries regarding report creation confirmation, and lookup help
type Receipt struct {
	GID string   `json:"gid"`// the id of the report - PARTITION KEY
	Key string   `json:"key"` // the report's md5 hash - SORT KEY
}

func ConvertSeverityLevelString(slvl string) ReportType {
	switch strings.ToLower(slvl) {
	case "1", "bug":
		return BugType
	case "2", "crash":
		return CrashType
	default:
		return UnknownType
	}
}
