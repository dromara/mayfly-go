package sqlstmt

import "strings"

type QuoteChar struct {
	StartDelimiter string
	EndDelimiter   string
}

/**
 * Wrap value with quote character.
 *
 * @param value value to be wrapped
 * @return wrapped value
 */
func (qc *QuoteChar) Wrap(value string) string {
	return qc.StartDelimiter + value + qc.EndDelimiter
}

/**
 * Unwrap value with quote character.
 *
 * @param value value to be unwrapped
 * @return unwrapped value
 */
func (qc *QuoteChar) Unwrap(value string) string {
	if qc.IsWrapped(value) {
		return value[len(qc.StartDelimiter) : len(value)-len(qc.EndDelimiter)]
	}
	return value
}

/**
 * Is wrapped by quote character.
 *
 * @param value value to be judged
 * @return is wrapped or not
 */
func (qc *QuoteChar) IsWrapped(value string) bool {
	return strings.HasPrefix(value, qc.StartDelimiter) && strings.HasSuffix(value, qc.EndDelimiter)
}

func NewQuoteChar(startDelimiter, endDelimiter string) *QuoteChar {
	return &QuoteChar{
		StartDelimiter: startDelimiter,
		EndDelimiter:   endDelimiter,
	}
}

var (
	BACK_QUOTE   = NewQuoteChar("`", "`")
	SINGLE_QUOTE = NewQuoteChar("'", "'")
	QUOTE        = NewQuoteChar("\"", "\"")
	BRACKETS     = NewQuoteChar("[", "]")
	PARENTHESES  = NewQuoteChar("(", ")")
	NONE         = NewQuoteChar("", "")

	BY_FIRST_CHAR = map[string]*QuoteChar{
		BACK_QUOTE.StartDelimiter:   BACK_QUOTE,
		SINGLE_QUOTE.StartDelimiter: SINGLE_QUOTE,
		QUOTE.StartDelimiter:        QUOTE,
		BRACKETS.StartDelimiter:     BRACKETS,
		PARENTHESES.StartDelimiter:  PARENTHESES,
	}
)

func GetQuoteChar(value string) *QuoteChar {
	if value == "" {
		return NONE
	}
	if qc := BY_FIRST_CHAR[value[0:1]]; qc != nil {
		return qc
	} else {
		return NONE
	}
}
