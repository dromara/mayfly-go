package utils

import (
	"encoding/json"
	"mayfly-go/pkg/global"
)

func Json2Map(jsonStr string) map[string]interface{} {
	var res map[string]interface{}
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
