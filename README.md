## Go Report (bugs & crashes)
A simple server for automated crash reporting, which stores reports using dynamodb as a backing store.
Keys are md5 hashes of the report content, used to uniquely identify each file within a group. 
Group IDs are human-readable identifiers for reports (i.e. from a particular catch block) 
which enable grouping of reports which occurred in a specific set of circumstances. 

# Preliminary Server Setup Details:
	1. Create a GitHub App with permission to read the repository, and read/create issues
	2. Generate the App Secrets (CLIENT_ID, CLIENT_SECRET, APP_ID) + the private key file
		- Webhook is required for creating an app, but we do not use it and the secret should be excluded
	3. Move the private key file to a secure location of which the instance has read access.
	4. Install the App onto the repository, record the install ID.
	5. Create the necessary params (See gh/service.go -> Secrets, Repo structs for param names)
	6. Specify the JWT and MSS Certificate Parameters (see auth/setup.go -> Config)
	7. Setup aws credentials, yada yada, ready to go.

# Application Side Details:
	. On build => generate & add certificate for release version
	. On first bug report => exchange certificate for jwt
	. Use JWT to send report

# CLI Usage
	. Provide GH user & token
	. Recieve JWT
	. Use JWT to make queries

# AWS Parameter Store:
	. Config, gh.Secrets, auth.Secrets => all have tagged fields (tag="paramName")
	. The tagged fields are extracted from the param store using the tag value as the param name
	. tags with values of form: "{value},secret" should be stored/transmitted as a SecureString


## Routes

<details>
<summary>`/certificate/{mssCertificate}/*`</summary>

- [(*Cors).Handler-fm]()
- [RequestID]()
- [Recoverer]()
- [URLFormat]()
- [Logger]()
- **/certificate/{mssCertificate}/***
	- **/**
		- _DELETE_
			- [(*Service).RemoveCertificateHandler.func1]()
		- _POST_
			- [(*Service).AddCertificateHandler.func1]()

</details>
<details>
<summary>`/report/*`</summary>

- [(*Cors).Handler-fm]()
- [RequestID]()
- [Recoverer]()
- [URLFormat]()
- [Logger]()
- **/report/***
	- **/**
		- _POST_
			- [main.ReportCtx]()
			- [main.PostHandler.func1]()
		- _GET_
			- [(*Service).OnlyDevsAuthenticate-fm]()
			- [main.GetAllHandler.func1]()

</details>
<details>
<summary>`/report/*/group/{reportsGID}/*`</summary>

- [(*Cors).Handler-fm]()
- [RequestID]()
- [Recoverer]()
- [URLFormat]()
- [Logger]()
- **/report/***
	- **/group/{reportsGID}/***
		- [main.ReportGroupCtx]()
		- **/**
			- _GET_
				- [main.GetGroupHandler.func1]()

</details>
<details>
<summary>`/report/*/group/{reportsGID}/key/{reportsKey}/*`</summary>

- [(*Cors).Handler-fm]()
- [RequestID]()
- [Recoverer]()
- [URLFormat]()
- [Logger]()
- **/report/***
	- **/group/{reportsGID}/key/{reportsKey}/***
		- [main.ReportGroupCtx]()
		- [main.ReportKeyCtx]()
		- **/**
			- _GET_
				- [main.GetReportHandler.func1]()
			- _DELETE_
				- [main.DeleteReportHandler.func1]()

</details>
<details>
<summary>`/token/*`</summary>

- [(*Cors).Handler-fm]()
- [RequestID]()
- [Recoverer]()
- [URLFormat]()
- [Logger]()
- **/token/***
	- **/**
		- _PUT_
			- [(*Service).TokenExchangeHandler.func1]()
		- _POST_
			- [(*Service).TokenExchangeHandler.func1]()

</details>

Total # of routes: 5

