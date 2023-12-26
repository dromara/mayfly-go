package stringx

import (
	"math/rand"
	"time"
)

const Nums = "0123456789"
const LowerChars = "abcdefghigklmnopqrstuvwxyz"
const UpperChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// 生成随机字符串
func Rand(l int) string {
	return RandByChars(l, Nums+LowerChars+UpperChars)
}

// 根据传入的chars，随机生成指定位数的字符串
func RandByChars(l int, chars string) string {
	strList := []byte(chars)

	result := []byte{}
	i := 0

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	charLen := len(strList)
	for i < l {
		new := strList[r.Intn(charLen)]
		result = append(result, new)
		i = i + 1
	}
	return string(result)
}
