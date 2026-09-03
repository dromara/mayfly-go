package ansiterm

import (
	"bytes"
	"unicode"
	"unicode/utf8"
)

func WidthOfRune(r rune) int {
	if r < 32 || (r >= 0x7f && r < 0xa0) {
		return 0
	}
	if unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r) {
		return 2
	}
	return 1
}

func DecodeUTF8WithReplacement(data []byte) (string, error) {
	var output bytes.Buffer
	for len(data) > 0 {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			output.WriteRune('\uFFFD')
			data = data[size:]
		} else {
			output.WriteRune(r)
			data = data[size:]
		}
	}
	return output.String(), nil
}

func BytesToString(data []byte) string {
	var result string
	for _, b := range data {
		result += string(b)
	}
	return result
}

func Range1(x, y int) []int {
	var res []int
	for i := x; i < y; i++ {
		res = append(res, i)
	}
	return res
}

func Contains[T int | string](slice []T, element T) bool {
	for _, v := range slice {
		if v == element {
			return true
		}
	}
	return false
}

func ReverseSlice(s []int) []int {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func Pop[T int | Savepoint](slice *[]T) (T, bool) {
	s := *slice
	if len(s) == 0 {
		var zeroVal T
		return zeroVal, false
	}
	val := s[len(s)-1]
	*slice = s[:len(s)-1]
	return val, true
}
