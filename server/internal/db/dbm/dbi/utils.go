package dbi

import (
	"strings"
)

func QuoteEscape(str string) string {
	return strings.Replace(str, `'`, `''`, -1)
}
