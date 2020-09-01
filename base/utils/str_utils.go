package utils

import (
	"bytes"
	"strings"
	"text/template"
)

// 可判断中文
func StrLen(str string) int {
	return len([]rune(str))
}

// 去除字符串左右空字符
func StrTrim(str string) string {
	return strings.Trim(str, " ")
}

func SubString(str string, begin, end int) (substr string) {
	// 将字符串的转换成[]rune
	rs := []rune(str)
	lth := len(rs)

	// 简单的越界判断
	if begin < 0 {
		begin = 0
	}
	if begin >= lth {
		begin = lth
	}
	if end > lth {
		end = lth
	}

	// 返回子串
	return string(rs[begin:end])
}

func UnicodeIndex(str, substr string) int {
	// 子串在字符串的字节位置
	result := strings.Index(str, substr)
	if result >= 0 {
		// 获得子串之前的字符串并转换成[]byte
		prefix := []byte(str)[0:result]
		// 将子串之前的字符串转换成[]rune
		rs := []rune(string(prefix))
		// 获得子串之前的字符串的长度，便是子串在字符串的字符位置
		result = len(rs)
	}

	return result
}

// 字符串模板解析
func TemplateResolve(temp string, data interface{}) string {
	t, _ := template.New("string-temp").Parse(temp)
	var tmplBytes bytes.Buffer

	err := t.Execute(&tmplBytes, data)
	if err != nil {
		panic(err)
	}
	return tmplBytes.String()
}

func ReverStrTemplate(temp, str string, res map[string]interface{}) {
	index := UnicodeIndex(temp, "{")
	ei := UnicodeIndex(temp, "}") + 1
	next := StrTrim(temp[ei:])
	nextContain := UnicodeIndex(next, "{")
	nextIndexValue := next
	if nextContain != -1 {
		nextIndexValue = SubString(next, 0, nextContain)
	}
	key := temp[index+1 : ei-1]
	// 如果后面没有内容了，则取字符串的长度即可
	var valueLastIndex int
	if nextIndexValue == "" {
		valueLastIndex = StrLen(str)
	} else {
		valueLastIndex = UnicodeIndex(str, nextIndexValue)
	}
	value := StrTrim(SubString(str, index, valueLastIndex))
	res[key] = value
	// 如果后面的还有需要解析的，则递归调用解析
	if nextContain != -1 {
		ReverStrTemplate(next, StrTrim(SubString(str, UnicodeIndex(str, value)+StrLen(value), StrLen(str))), res)
	}
}
