package stringx

import (
	mrand "math/rand"
	"strings"
	"time"
	"uuid"
)

const Nums = "0123456789"
const LowerChars = "abcdefghigklmnopqrstuvwxyz"
const UpperChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// 生成随机字符串
func Rand(l int) string {
	return RandByChars(l, Nums+LowerChars+UpperChars)
}

// RandUUID 生成随机 UUID（32 位，无序）
func RandUUID() string {
	return strings.Replace(uuid.New().String(), "-", "", -1)
}

// SortableUUID 生成有序 UUID v7（48bit 毫秒时间戳 + 随机，字典序 == 时间序）
// 去掉连字符后为 32 位 hex 字符串，可直接用于 ORDER BY 排序
func SortableUUID() string {
	return strings.Replace(uuid.NewV7().String(), "-", "", -1)
}

// 根据传入的chars，随机生成指定位数的字符串
func RandByChars(l int, chars string) string {
	strList := []byte(chars)

	result := []byte{}
	i := 0

	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	charLen := len(strList)
	for i < l {
		new := strList[r.Intn(charLen)]
		result = append(result, new)
		i = i + 1
	}
	return string(result)
}
