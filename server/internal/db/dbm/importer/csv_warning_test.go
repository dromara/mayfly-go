package importer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCSVFieldCountMismatchReportsWarning 字段数与首行不一致必须回调告警。
//
// 典型成因是单元格内出现未转义的裸引号（LazyQuotes 会重新断词），该行后续列整体位移；
// 导入按列下标映射，错位会静默写进相邻列。解析层保持宽松（历史文件依赖它），
// 但必须把这个风险交给上层提示操作者。
func TestCSVFieldCountMismatchReportsWarning(t *testing.T) {
	const csvData = "a,b,c\n1,2,3\n4,5\n6,7,8\n"
	var warns []string
	err := (&CSVImporter{}).Stream(strings.NewReader(csvData), &Options{HasHeader: true}, &StreamHandler{
		OnHeader:     func([]string, []string) error { return nil },
		OnRow:        func([]string) error { return nil },
		OnRowWarning: func(rowNum int, msg string) { warns = append(warns, msg) },
	})
	require.NoError(t, err)
	require.Len(t, warns, 1, "只有列数不一致的那一行应告警")
	require.Contains(t, warns[0], "column count mismatch")

	// 对齐的文件不得告警
	warns = nil
	err = (&CSVImporter{}).Stream(strings.NewReader("a,b\n1,2\n3,4\n"), &Options{HasHeader: true}, &StreamHandler{
		OnHeader:     func([]string, []string) error { return nil },
		OnRow:        func([]string) error { return nil },
		OnRowWarning: func(int, string) { warns = append(warns, "unexpected") },
	})
	require.NoError(t, err)
	require.Empty(t, warns)
}
