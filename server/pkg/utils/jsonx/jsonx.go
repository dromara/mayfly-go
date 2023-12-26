package jsonx

import (
	"encoding/json"
	"mayfly-go/pkg/logx"
	"strings"

	"github.com/buger/jsonparser"
)

// json字符串转map
func ToMap(jsonStr string) map[string]any {
	return ToMapByBytes([]byte(jsonStr))
}

// json字符串转结构体
func To[T any](jsonStr string, res T) (T, error) {
	return res, json.Unmarshal([]byte(jsonStr), &res)
}

// json字节数组转map
func ToMapByBytes(bytes []byte) map[string]any {
	var res map[string]any
	err := json.Unmarshal(bytes, &res)
	if err != nil {
		logx.Errorf("json字符串转map失败: %s", err.Error())
	}
	return res
}

// 转换为json字符串
func ToStr(val any) string {
	if strBytes, err := json.Marshal(val); err != nil {
		logx.ErrorTrace("toJsonStr error: ", err)
		return ""
	} else {
		return string(strBytes)
	}
}

// 根据json字节数组获取对应字段路径的string类型值
//
// @param fieldPath字段路径。如user.username等
func GetStringByBytes(bytes []byte, fieldPath string) (string, error) {
	return jsonparser.GetString(bytes, strings.Split(fieldPath, ".")...)
}

// 根据json字符串获取对应字段路径的string类型值
//
// @param fieldPath字段路径。如user.username等
func GetString(jsonStr string, fieldPath string) (string, error) {
	return GetStringByBytes([]byte(jsonStr), fieldPath)
}

// 根据json字节数组获取对应字段路径的int类型值
//
// @param fieldPath字段路径。如user.age等
func GetIntByBytes(bytes []byte, fieldPath string) (int64, error) {
	return jsonparser.GetInt(bytes, strings.Split(fieldPath, ".")...)
}

// 根据json字符串获取对应字段路径的int类型值
//
// @param fieldPath字段路径。如user.age等
func GetInt(jsonStr string, fieldPath string) (int64, error) {
	return GetIntByBytes([]byte(jsonStr), fieldPath)
}

// 根据json字节数组获取对应字段路径的bool类型值
//
// @param fieldPath字段路径。如user.isDeleted等
func GetBoolByBytes(bytes []byte, fieldPath string) (bool, error) {
	return jsonparser.GetBoolean(bytes, strings.Split(fieldPath, ".")...)
}

// 根据json字符串获取对应字段路径的bool类型值
//
// @param fieldPath字段路径。如user.isDeleted等
func GetBool(jsonStr string, fieldPath string) (bool, error) {
	return GetBoolByBytes([]byte(jsonStr), fieldPath)
}
