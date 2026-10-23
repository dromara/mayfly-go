package api

import (
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCheckImportFileSize 导入文件的字节护栏：
// CSV 流式解析给较大上限，Excel 因 excelize 整体读文件给较小上限；超限必须在解析前拒绝。
func TestCheckImportFileSize(t *testing.T) {
	// CSV 在 200MB 仍允许（上限 512MB）
	require.NotPanics(t, func() {
		checkImportFileSize(&multipart.FileHeader{Filename: "data.csv", Size: 200 << 20})
	})
	// Excel 超过 100MB 必须拒绝
	require.Panics(t, func() {
		checkImportFileSize(&multipart.FileHeader{Filename: "data.xlsx", Size: maxImportExcelBytes + 1})
	})
	// CSV 超过 512MB 拒绝
	require.Panics(t, func() {
		checkImportFileSize(&multipart.FileHeader{Filename: "data.csv", Size: maxImportCSVBytes + 1})
	})
	// 扩展名大小写不敏感
	require.Panics(t, func() {
		checkImportFileSize(&multipart.FileHeader{Filename: "DATA.XLSX", Size: maxImportExcelBytes + 1})
	})
}
