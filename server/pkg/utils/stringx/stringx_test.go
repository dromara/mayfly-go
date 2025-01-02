package stringx

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTruncateStr(t *testing.T) {
	testCases := []struct {
		data   string
		length int
		want   string
	}{
		{"123一二三", 4, "123...三"},
		{"123一二三", 5, "123...二三"},
	}
	for _, tc := range testCases {
		t.Run(strconv.Itoa(tc.length), func(t *testing.T) {
			got := Truncate(tc.data, tc.length, 3, "...")
			require.Equal(t, tc.want, got)
		})
	}
}
