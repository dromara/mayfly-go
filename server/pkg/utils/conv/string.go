package conv

import (
	"mayfly-go/pkg/logx"
	"strconv"
)

// 将字符串值转为int值, 若value为空或者转换失败，则返回默认值
func Str2Int(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	if intV, err := strconv.Atoi(value); err != nil {
		logx.ErrorTrace("str conv int error: ", err)
		return defaultValue
	} else {
		return intV
	}
}
