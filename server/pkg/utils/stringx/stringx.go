package stringx

import (
	"bytes"
	"strings"
	"text/template"
	"unicode/utf8"
)

// 可判断中文
func Len(str string) int {
	return len([]rune(str))
}

// 去除字符串左右空字符
func Trim(str string) string {
	return strings.Trim(str, " ")
}

// 去除字符串左右空字符与\n\r换行回车符
func TrimSpaceAndBr(str string) string {
	return strings.TrimFunc(str, func(r rune) bool {
		s := string(r)
		return s == " " || s == "\n" || s == "\r"
	})
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

func Camel2Underline(name string) string {
	if name == "" {
		return ""
	}

	temp := strings.Split(name, "_")
	var s string
	for _, v := range temp {
		vv := []rune(v)
		if len(vv) > 0 {
			if bool(vv[0] >= 'a' && vv[0] <= 'z') { //首字母大写
				vv[0] -= 32
			}
			s += string(vv)
		}
	}

	return s
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
func TemplateResolve(temp string, data any) (string, error) {
	t, err := template.New("string-temp").Parse(temp)
	if err != nil {
		return "", err
	}
	var tmplBytes bytes.Buffer

	err = t.Execute(&tmplBytes, data)
	if err != nil {
		return "", err
	}
	return tmplBytes.String(), nil
}

func ReverStrTemplate(temp, str string, res map[string]any) {
	index := UnicodeIndex(temp, "{")
	ei := UnicodeIndex(temp, "}") + 1
	next := Trim(temp[ei:])
	nextContain := UnicodeIndex(next, "{")
	nextIndexValue := next
	if nextContain != -1 {
		nextIndexValue = SubString(next, 0, nextContain)
	}
	key := temp[index+1 : ei-1]
	// 如果后面没有内容了，则取字符串的长度即可
	var valueLastIndex int
	if nextIndexValue == "" {
		valueLastIndex = Len(str)
	} else {
		valueLastIndex = UnicodeIndex(str, nextIndexValue)
	}
	value := Trim(SubString(str, index, valueLastIndex))
	res[key] = value
	// 如果后面的还有需要解析的，则递归调用解析
	if nextContain != -1 {
		ReverStrTemplate(next, Trim(SubString(str, UnicodeIndex(str, value)+Len(value), Len(str))), res)
	}
}

// Truncate 截断字符串并在中间部分显示指定的替换字符串
func Truncate(s string, length int, prefixLen int, replace string) string {
	totalRunes := utf8.RuneCountInString(s)

	// 如果字符串长度小于或等于指定的 length，直接返回原字符串
	if totalRunes <= length {
		return s
	}

	// 如果字符串长度小于或等于 prefixLen，直接返回原字符串
	if totalRunes <= prefixLen {
		return s
	}

	// 计算 suffixLen
	suffixLen := length - prefixLen

	// 确保 suffixLen 不会越界
	if suffixLen <= 0 {
		runes := []rune(s)
		return string(runes[:length]) + replace
	}

	// 获取前 prefixLen 个字符
	runes := []rune(s)
	prefix := string(runes[:prefixLen])

	// 获取后 suffixLen 个字符
	suffix := string(runes[len(runes)-suffixLen:])

	// 返回格式化后的字符串
	return prefix + replace + suffix
}
