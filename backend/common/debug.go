package common

import (
	"encoding/json"
	"fmt"
)

func DumpJSON(value any) {
	formattedBytes, stdErr := json.MarshalIndent(value, "", "  ")
	if stdErr != nil {
		//nolint:forbidigo
		fmt.Printf("DEBUG: warning: couldn't dump JSON")
		return
	}

	//nolint:forbidigo
	fmt.Printf("DEBUG: dumped JSON: %v\n", string(formattedBytes))
}
