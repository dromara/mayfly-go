package stringx

import (
	"sort"
	"strconv"
	"testing"
	"time"

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

func TestSortableUUID(t *testing.T) {
	// 生成多个 ID，验证长度和唯一性
	ids := make([]string, 100)
	for i := range ids {
		ids[i] = SortableUUID()
		require.Len(t, ids[i], 32, "SortableUUID should be 32 chars")
	}

	// 验证唯一性
	unique := make(map[string]bool)
	for _, id := range ids {
		require.False(t, unique[id], "SortableUUID should be unique")
		unique[id] = true
	}

	// 验证有序性：间隔生成，后生成的应该字典序更大
	id1 := SortableUUID()
	time.Sleep(2 * time.Millisecond)
	id2 := SortableUUID()
	require.Greater(t, id2, id1, "later SortableUUID should be lexicographically greater")

	// 验证排序后与生成顺序一致
	shuffled := make([]string, len(ids))
	copy(shuffled, ids)
	sort.Strings(shuffled)
	require.Equal(t, ids, shuffled, "SortableUUIDs should already be in sorted order")
}
