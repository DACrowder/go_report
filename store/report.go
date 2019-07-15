package store

import (
	"time"
)

type ReportType int
const (
	Crash ReportType = iota
	Bug              = iota
)

type Report struct {
	// The report creation request will contain these three fields
	Id       string                 `json:"id"`
	Severity ReportType             `json:"severity"`
	Content  map[string]interface{} `json:"content"`
}

// For sending responses to queries regarding report creation confirmation, and lookup help
type ReportReceipt struct {
	ID string // the id corresponding to the hash -- for quick lookup of a/groups of reports
	TimeSlot time.Time // the time of report arrival -- for looking up report by time
	FileName string // a string representation of the Report hash -- for specific lookup
}
