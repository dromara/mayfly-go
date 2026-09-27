// Package importer 表格文件导入解析：把 CSV/Excel 等表格文件解析为「表头 + 行数据」的二维文本结构，
// 供上层按目标表列类型转换后生成插入语句。与 export 包（数据库→文件）方向相反，二者共同构成文件与表的双向通道。
//
// 本包只负责「文件 → 结构化文本」，不含任何 SQL/方言逻辑：值到列类型的转换、插入语句生成交由
// application 层复用 dbi.SQLGenerator，保持解析层与执行层解耦，新增文件格式无需改动导入执行代码。
package importer

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Options 文件解析选项。零值即可用于最常见场景（含表头的逗号分隔 CSV / 取首个 sheet 的 Excel）。
type Options struct {
	// HasHeader 首行是否为列名行（true=首行作为表头且不参与数据导入；false=无表头，列以位置索引标识）
	HasHeader bool

	// FieldSeparator CSV 字段分隔符，空则默认 ","；Excel 忽略此项
	FieldSeparator string

	// Sheet Excel 工作表名，空则取第一个工作表；CSV 忽略此项
	Sheet string

	// PreviewLimit 仅解析用于预览的前 N 行数据（不含表头），<=0 表示不限制。全量导入时由调用方传 0
	PreviewLimit int
}

// Table 解析结果：文件表头（无表头时为空）与数据行。所有单元格以文本形态返回，类型转换由上层负责。
type Table struct {
	// HasHeader 解析时是否按首行识别了表头（供上层判断列标识是列名还是列下标）
	HasHeader bool

	// Headers 表头列名；HasHeader=false 时为空切片
	Headers []string

	// Rows 数据行（已剔除表头行）；每行是单元格文本切片
	Rows [][]string

	// Sheets 多工作表格式（如 Excel）内的全部工作表名，供上层让用户选择导入哪个 sheet；CSV 等为 nil
	Sheets []string
}

// ColumnCount 返回文件的列数：有表头取表头宽度，否则取数据行的最大宽度。
// 表格导入按「列位置」映射，上层据此校验映射下标不越界，避免越界列被静默写成 NULL。
func (t *Table) ColumnCount() int {
	if n := len(t.Headers); n > 0 {
		return n
	}
	count := 0
	for _, row := range t.Rows {
		if len(row) > count {
			count = len(row)
		}
	}
	return count
}

// Importer 单个表格文件格式的解析器契约。
type Importer interface {
	// Extensions 该解析器支持的文件扩展名（小写、不含点），如 ["csv"]
	Extensions() []string

	// Parse 把文件流解析为 Table。解析失败必须返回 error，不得静默丢行/截断
	Parse(r io.Reader, opts *Options) (*Table, error)
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Importer)
)

// Register 注册解析器，按其支持的扩展名建立索引。同名扩展名后注册者覆盖前者。
// 新增文件格式只需实现本接口并在 init 注册，编排/应用/接口层无需改动（开闭原则）。
func Register(imp Importer) {
	if imp == nil {
		panic("importer: register nil importer")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	for _, ext := range imp.Extensions() {
		registry[strings.ToLower(strings.TrimPrefix(ext, "."))] = imp
	}
}

// ForFilename 按文件名后缀选择解析器。不支持的后缀返回 error（上层据此给出「仅支持 csv/excel」提示）。
func ForFilename(filename string) (Importer, error) {
	// filepath.Ext 取最后一段扩展名（含点），去点后小写归一即为注册键；无扩展名时返回空串
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	registryMu.RLock()
	imp := registry[ext]
	registryMu.RUnlock()
	if imp == nil {
		return nil, fmt.Errorf("unsupported import file type [%s], supported: %s", ext, strings.Join(SupportedExtensions(), ", "))
	}
	return imp, nil
}

// SupportedExtensions 返回全部已注册扩展名（字典序，输出稳定，供 API 能力协商与前端提示）。
func SupportedExtensions() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	exts := make([]string, 0, len(registry))
	for e := range registry {
		exts = append(exts, e)
	}
	sort.Strings(exts)
	return exts
}

// isBlankRow 判断一行是否全为空/纯空白单元格。Excel 尾随/中间的空白行、CSV 连续空行都不是有效数据，
// 导入时应跳过——否则会被写成整行 NULL（可空列）或触发非空约束报错，并使行数统计虚高。
func isBlankRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}
