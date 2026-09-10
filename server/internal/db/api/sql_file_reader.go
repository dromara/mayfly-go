package api

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"sort"
	"strings"
)

const (
	// sqlZipMaxBytes 压缩包原始字节上限：zip 需整体缓冲后才能随机访问，不加上限会被超大包打满内存
	sqlZipMaxBytes = 10 << 20
	// sqlUnzipMaxBytes 解压后内容总上限（防 zip 炸弹）；SQL文本压缩比通常在20:1以内，此上限足够宽松
	sqlUnzipMaxBytes = 200 << 20
)

// newSqlFileReader 按文件名后缀还原出真正的 SQL 文本流（SQL文件执行与备份文件导入共用）：
//   - .zip：按文件名升序导入包内全部 SQL 文件，文件之间补换行，避免上一条语句与下一条粘连
//   - .gz/.gzip：gunzip 后返回（平台下载的备份即 gzip 流，可直接回导）
//   - 其他：原样返回，保持流式读取（百MB级脚本不进内存）
//
// 任何异常（非法压缩包、超出大小上限）都必须返回 error：
// 静默返回空内容或截断内容，导入会表现为「执行成功但没导全数据」，事后无法追溯，属数据安全事故
func newSqlFileReader(filename string, body io.Reader) (io.Reader, error) {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return zipSqlReader(body)
	case strings.HasSuffix(lower, ".gz"), strings.HasSuffix(lower, ".gzip"):
		gr, err := gzip.NewReader(body)
		if err != nil {
			return nil, fmt.Errorf("invalid gz sql file: %w", err)
		}
		budget := &unzipBudget{remain: sqlUnzipMaxBytes}
		return budget.wrap(gr, lower), nil
	default:
		return body, nil
	}
}

// zipSqlReader 读取压缩包内的 SQL 文件内容并按序拼接
func zipSqlReader(body io.Reader) (io.Reader, error) {
	// 多读1字节用于判断是否超限（超限必须报错，不能把截断后的包当成完整包）
	data, err := io.ReadAll(io.LimitReader(body, sqlZipMaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read zip file error: %w", err)
	}
	if len(data) > sqlZipMaxBytes {
		return nil, fmt.Errorf("zip file exceeds the %dMB limit, please import the sql file directly", sqlZipMaxBytes>>20)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid zip file: %w", err)
	}
	entries, err := zipSqlEntries(zr)
	if err != nil {
		return nil, err
	}

	// 包内数据已全部缓冲在内存中，这里打开的是内存流，无需显式关闭
	budget := &unzipBudget{remain: sqlUnzipMaxBytes}
	readers := make([]io.Reader, 0, len(entries)*2)
	for _, f := range entries {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open zip entry error: %w", err)
		}
		var inner io.Reader = rc
		if lower := strings.ToLower(f.Name); strings.HasSuffix(lower, ".gz") || strings.HasSuffix(lower, ".gzip") {
			gr, err := gzip.NewReader(rc)
			if err != nil {
				return nil, fmt.Errorf("invalid gz sql file [%s] in zip: %w", f.Name, err)
			}
			inner = gr
		}
		readers = append(readers, budget.wrap(inner, f.Name))
		// 每个条目后补换行：避免前一个文件末尾无分号/无换行的内容与后一个文件首行粘连成语义错误的语句
		readers = append(readers, strings.NewReader("\n"))
	}
	return io.MultiReader(readers...), nil
}

// zipSqlEntries 选出压缩包内要导入的条目并按名称升序排序（排序保证多次导入顺序一致、结果可复现）：
//   - 跳过目录与打包产生的垃圾条目（__MACOSX/、._*、.DS_Store 等，这些是二进制元数据，执行只会报错）
//   - 优先只取 .sql / .sql.gz 文件；包内没有此类命名时回退为全部文件（兼容既有压缩包）
func zipSqlEntries(zr *zip.Reader) ([]*zip.File, error) {
	sqlFiles := make([]*zip.File, 0, len(zr.File))
	otherFiles := make([]*zip.File, 0, len(zr.File))
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || isZipJunkEntry(f.Name) {
			continue
		}
		lower := strings.ToLower(f.Name)
		if strings.HasSuffix(lower, ".sql") || strings.HasSuffix(lower, ".gz") || strings.HasSuffix(lower, ".gzip") {
			sqlFiles = append(sqlFiles, f)
			continue
		}
		otherFiles = append(otherFiles, f)
	}
	entries := sqlFiles
	if len(entries) == 0 {
		entries = otherFiles
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("zip file is empty")
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}

// isZipJunkEntry 判断是否为压缩工具/系统打包时附带的非内容条目
func isZipJunkEntry(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, "__macosx/") {
		return true
	}
	base := lower
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	// AppleDouble 元数据（._xxx）与 .DS_Store 等隐藏文件
	return strings.HasPrefix(base, "._") || strings.HasPrefix(base, ".")
}

// unzipBudget 解压内容大小预算。超限以 error 终止读取而非返回 EOF：
// 截断后的 SQL 若被当作完整内容导入，会在缺失尾部语句的情况下“成功”，属静默丢数据
type unzipBudget struct {
	remain int64
}

func (b *unzipBudget) wrap(r io.Reader, name string) io.Reader {
	return &budgetReader{reader: r, budget: b, name: name}
}

type budgetReader struct {
	reader io.Reader
	budget *unzipBudget
	name   string
}

func (c *budgetReader) Read(p []byte) (int, error) {
	if c.budget.remain <= 0 {
		return 0, fmt.Errorf("uncompressed sql content exceeds the %dMB limit at [%s]", sqlUnzipMaxBytes>>20, c.name)
	}
	if int64(len(p)) > c.budget.remain {
		p = p[:c.budget.remain]
	}
	n, err := c.reader.Read(p)
	c.budget.remain -= int64(n)
	return n, err
}
