package utils

import (
	"encoding/json"
)

func Json2Map(jsonStr string) map[string]interface{} {
	var res map[string]interface{}
	if jsonStr == "" {
		return res
	}
	_ = json.Unmarshal([]byte(jsonStr), &res)
	return res
}
