package stringx

import (
	"bytes"
	"strings"
	"text/template"
	"unicode"
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

// Camel2Snake 将驼峰命名转换为下划线命名。
//
// 连续大写视为一个首字母缩略词，只在该词的末尾划词："HTTPServer" -> "http_server"、
// "userID" -> "user_id"，与 GORM NamingStrategy 的列名推导同口径。本身就是蛇形的
// 物理列名（"id"、"is_deleted"）走零分配快路径原样返回
func Camel2Snake(name string) string {
	// 先扫一遍：纯 ASCII 且不含大写字母（本就是蛇形的物理列名，最常见的输入）
	// 原样返回，避免白分配一次
	needConvert := false
	for i := 0; i < len(name); i++ {
		if name[i] >= utf8.RuneSelf || (name[i] >= 'A' && name[i] <= 'Z') {
			needConvert = true
			break
		}
	}
	if !needConvert {
		return name
	}

	var result strings.Builder
	result.Grow(len(name) + 4)
	prevIsUpper, first := false, true
	for i := 0; i < len(name); {
		r, size := utf8.DecodeRuneInString(name[i:])
		if !unicode.IsUpper(r) {
			result.WriteRune(r)
			prevIsUpper = false
		} else {
			nextIsLower := false
			if n, sz := utf8.DecodeRuneInString(name[i+size:]); sz > 0 && unicode.IsLower(n) {
				nextIsLower = true
			}
			// 连续大写视为一个首字母缩略词：只在前一字符不是大写（单词边界）
			// 或后一字符是小写（缩略词末字母属于下一个单词，如 HTTPServer 的 S）时划词
			if !first && (!prevIsUpper || nextIsLower) {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
			prevIsUpper = true
		}
		first = false
		i += size
	}
	return result.String()
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

// Truncate 截断字符串并在中间部分显示指定的替换字符串。
// 该函数基于 Unicode 字符（rune）进行计算，支持中文等多字节字符。
//
// 参数说明：
//   - s: 原始字符串
//   - length: 截断后字符串的总最大长度（包含前缀、替换串和后缀）。
//     如果原字符串长度小于等于 length，则直接返回原字符串。
//   - prefixLen: 保留的前缀字符数量。
//   - replace: 用于替换中间被截断部分的字符串（如 "..."）。
//
// 返回：
//   - 格式化后的字符串。
//
// 示例：
//
//	Truncate("Hello, World!", 10, 5, "...") -> "Hello...ld!"
//	Truncate("你好世界", 3, 1, "..") -> "你..界"
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
