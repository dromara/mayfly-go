package importer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestExcelImporterNeverReadsCellStyle Excel 解析不得调用 GetCellStyle。
//
// excelize 的 GetCellStyle 会触发 workSheetReader：把整张 sheetN.xml 反序列化并常驻内存。
// 一旦按"单元格是否套了日期格式"做归一，解析层"不聚合整表"的内存保证就不成立
// （100MB xlsx 峰值可达 GB 级，属进程级不可 recover）。
//
// 日期序列数还原改由目标列类型驱动（Options.DateSerialColumns），本用例锁住"不再回退到按样式判断"。
func TestExcelImporterNeverReadsCellStyle(t *testing.T) {
	content, err := os.ReadFile("excel_importer.go")
	require.NoError(t, err)
	source := string(content)

	// 注释里可以提到 GetCellStyle（解释为什么不用），但不得出现真实调用
	require.NotContains(t, source, "f.GetCellStyle(",
		"解析层不得读取单元格样式：它会触发 excelize 全量解析整张 sheet")
	require.NotContains(t, source, "builtinDateTimeNumFmts",
		"按样式判断日期的死代码必须一并移除，避免被再次启用")
	require.NotContains(t, source, "looksLikeDateTimeFormat",
		"格式启发式判定也必须移除：留着就会被再次用来触发 GetCellStyle")
	require.Contains(t, source, "opts.DateSerialColumns",
		"日期归一必须由目标列驱动")
}
