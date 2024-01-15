package stringx

import (
	"strconv"
)

// 将字符串值转为int值, 若value为空或者转换失败，则返回默认值
func ConvInt(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	if intV, err := strconv.Atoi(value); err != nil {
		return defaultValue
	} else {
		return intV
	}
}
