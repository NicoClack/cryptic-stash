package testdata

import _ "embed"

//go:embed existingClientCredential.json
var ExistingClientCredentialJSON string

//go:embed existingServerCredential.json
var ExistingServerCredentialJSON string
