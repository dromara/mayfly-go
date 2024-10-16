package sqlstmt

import (
	"testing"
)

func TestQuoteCharUnwrap(t *testing.T) {
	value := "`hehehe`"
	qc := GetQuoteChar(value)
	if qc != BACK_QUOTE {
		t.Fatal("quote char should be BACK_QUOTE")
	}
	oriValue := qc.Unwrap(value)
	t.Log(oriValue)
}
