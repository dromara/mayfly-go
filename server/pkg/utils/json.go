package utils

import (
	"encoding/json"
	"mayfly-go/pkg/global"
)

func Json2Map(jsonStr string) map[string]any {
	var res map[string]any
	if jsonStr == "" {
		return res
	}
	_ = json.Unmarshal([]byte(jsonStr), &res)
	return res
}

func ToJsonStr(val any) string {
	if strBytes, err := json.Marshal(val); err != nil {
		global.Log.Error("toJsonStr error: ", err)
		return ""
	} else {
		return string(strBytes)
	}
}
