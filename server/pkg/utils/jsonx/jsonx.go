package jsonx

import (
	"encoding/json"
	"mayfly-go/pkg/global"
)

// json字符串转map
func ToMap(jsonStr string) map[string]any {
	var res map[string]any
	if jsonStr == "" {
		return res
	}
	_ = json.Unmarshal([]byte(jsonStr), &res)
	return res
}

// 转换为json字符串
func ToStr(val any) string {
	if strBytes, err := json.Marshal(val); err != nil {
		global.Log.Error("toJsonStr error: ", err)
		return ""
	} else {
		return string(strBytes)
	}
}
