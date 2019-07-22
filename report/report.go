package report

type Type int

const (
	UnknownType Type = iota
	BugType
	CrashType
)

type Instance struct {
	// The report creation request will contain these three fields
	GID      string                 `json:"gid"`
	Severity Type                   `json:"severity"`
	Content  map[string]interface{} `json:"content"`
	Key      string
}

// For sending responses to queries regarding report creation confirmation, and lookup help
type Receipt struct {
	GID      string // the id of the report (directory)
	FileName string // the filename (string representation of its md5 hash)
}
