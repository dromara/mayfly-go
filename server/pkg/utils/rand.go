package utils

import (
	"math/rand"
	"time"
)

const randChar = "0123456789abcdefghigklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// 生成随机字符串
func RandString(l int) string {
	strList := []byte(randChar)

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
