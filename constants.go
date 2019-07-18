package main

import "time"

type ReportType int

type JwtClaimKey string
type JwtAudience string
type RequestContextKey string

const ExpiresOneYear = (time.Minute * 60 * 24 * 365)

const (
	GHAudience  JwtAudience = "github"
	MSSAudience JwtAudience = "mss"
)

const (
	GHUser         JwtClaimKey = "ghuname"
	GHToken        JwtClaimKey = "ghtkn"
	MSSCertificate JwtClaimKey = "mssCert"
)

const (
	// -------- CONTEXT KEYS --------
	// These constants are the context() keys to retrieve the Key/GID/Severity/Report from the request context
	// They are used by the middleware to place values in context predictably
	// Likewise, the handlers use them to retrieve values via Context().Values(ReportVar) -> value
	ReportKeyVar           RequestContextKey = "reportsKey"
	ReportGIDVar           RequestContextKey = "reportsGID"
	ReportSeverityLevelVar RequestContextKey = "severityLevel"
	ReportCtxVar           RequestContextKey = "reportFromRequestBody"

	MSSCertificateCtxVar		RequestContextKey = "mssCertificate"
)

const (
	Unknown ReportType = iota
	Bug
	Crash
)
