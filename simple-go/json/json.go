package json

import (
	"encoding/json"
	"fmt"
)

func ToJson(v interface{}) string {
	// Format as JSON array
	result, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("Error encoding result: %v", err))
	}

	return string(result)
}

func ToPrettyJson(v interface{}) string {
	// Format as JSON array
	result, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("Error encoding result: %v", err))
	}

	return string(result)
}
