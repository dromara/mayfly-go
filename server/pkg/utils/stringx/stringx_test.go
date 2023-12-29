package stringx

import (
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestTruncateStr(t *testing.T) {
	testCases := []struct {
		data   string
		length int
		want   string
	}{
		{"123一二三", 0, ""},
		{"123一二三", 1, "1"},
		{"123一二三", 3, "123"},
		{"123一二三", 4, "123"},
		{"123一二三", 5, "123"},
		{"123一二三", 6, "123一"},
		{"123一二三", 7, "123一"},
		{"123一二三", 11, "123一二"},
		{"123一二三", 12, "123一二三"},
		{"123一二三", 13, "123一二三"},
	}
	for _, tc := range testCases {
		t.Run(strconv.Itoa(tc.length), func(t *testing.T) {
			got := TruncateStr(tc.data, tc.length)
			require.Equal(t, tc.want, got)
		})
	}
}
