# Go Report (bugs & crashes)
A simple server for automated crash reporting, written in Go

Welcome to the chi/_examples/rest generated docs.

## Routes

<details>
<summary>`/report/*`</summary>

- [(*Cors).Handler-fm]()
- [RequestID]()
- [Recoverer]()
- [URLFormat]()
- **/report/***
	- **/**
		- _POST_
			- [main.PostHandler.func1]()
		- _GET_
			- [main.GetAllHandler.func1]()

</details>
<details>
<summary>`/report/*/group/{reportsGID}/*`</summary>

- [(*Cors).Handler-fm]()
- [RequestID]()
- [Recoverer]()
- [URLFormat]()
- **/report/***
	- **/group/{reportsGID}/***
		- [main.ReportGroupCtx]()
		- **/**
			- _GET_
				- [main.GetGroupHandler.func1]()
			- _DELETE_
				- [main.DeleteGroupHandler.func1]()

</details>
<details>
<summary>`/report/*/severity/{severityLevel}/*`</summary>

- [(*Cors).Handler-fm]()
- [RequestID]()
- [Recoverer]()
- [URLFormat]()
- **/report/***
	- **/severity/{severityLevel}/***
		- [main.ReportSeverityCtx]()
		- **/**
			- _GET_
				- [main.GetBatchByTypeHandler.func1]()

</details>
<details>
<summary>`/report/*/{reportsKey}/*`</summary>

- [(*Cors).Handler-fm]()
- [RequestID]()
- [Recoverer]()
- [URLFormat]()
- **/report/***
	- **/{reportsKey}/***
		- [main.ReportKeyCtx]()
		- **/**
			- _GET_
				- [main.GetReportHandler.func1]()
			- _DELETE_
				- [main.DeleteReportHandler.func1]()

</details>

Total # of routes: 4