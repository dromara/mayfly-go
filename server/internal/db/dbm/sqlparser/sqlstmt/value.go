package sqlstmt

import "strings"

type IdentifierValue struct {
	Value     string
	QuoteChar *QuoteChar
}

func NewIdentifierValue(value string) *IdentifierValue {
	value = strings.TrimPrefix(value, ".")
	qc := GetQuoteChar(value)
	if qc == NONE {
		return &IdentifierValue{
			Value:     value,
			QuoteChar: qc,
		}
	}
	return &IdentifierValue{
		Value:     qc.Unwrap(value),
		QuoteChar: qc,
	}
}
