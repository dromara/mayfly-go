package schema

// 结构指纹：把一张表的结构压成一个可比较的哈希，供漂移检测与告警使用。
//
// 指纹覆盖的方面必须与差异比对**严格一致**，否则会出现「比对说有差异、指纹说没变」
// 这种自相矛盾的输出，而用户无法判断该信哪个。因此这里直接复用比对用的那几个取值
// （列类型文本、可空、归一化后的默认值、自增、有序主键、按列组合归并的索引），
// 不另立一套判据；注释与索引名刻意排除在外——差异比对也不比它们（见 diffIndexes 注释）。

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
)

// TableSummary 一张表的结构摘要，用于让漂移说明说得出具体的变化而不是「结构变了」
type TableSummary struct {
	// Fingerprint 结构指纹（sha256 十六进制）
	Fingerprint string
	// ColumnCount 列数
	ColumnCount int
	// PrimaryKeys 有序主键列名，逗号连接；主键读不到时为 nil 而非空串（见 TableSnapshot）
	PrimaryKeys string
	// PrimaryKeysAvailable 主键是否真的读到了
	PrimaryKeysAvailable bool
	// IndexCount 索引个数（按列组合归并后的数量，与比对同口径）
	IndexCount int
	// IndexesAvailable 索引内省是否成功
	IndexesAvailable bool
}

// SummarizeTable 计算一张表的结构摘要。
//
// 采集失败的面（主键/索引读不到）在指纹里显式编码：不能把「读不到」哈希成与
// 「确实没有」相同的值，否则一次权限受限的内省会被记成基线，之后真的读到时又报成漂移
func SummarizeTable(table TableSnapshot) TableSummary {
	builder := strings.Builder{}
	dbType := dbi.DbType(table.DbType)
	for _, column := range table.Columns {
		builder.WriteString(canonicalColumn(dbType, column))
		builder.WriteByte('\n')
	}

	summary := TableSummary{
		ColumnCount:          len(table.Columns),
		PrimaryKeysAvailable: table.PrimaryKeysAvailable,
		IndexesAvailable:     table.IndexesAvailable,
	}

	if table.PrimaryKeysAvailable {
		summary.PrimaryKeys = strings.ToLower(strings.Join(table.PrimaryKeys, ","))
		builder.WriteString("pk?:1 pk=" + summary.PrimaryKeys + "\n")
	} else {
		builder.WriteString("pk?:0\n")
	}

	if table.IndexesAvailable {
		indexes := indexByKey(table.Indexes)
		keys := make([]string, 0, len(indexes))
		for key := range indexes {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		summary.IndexCount = len(keys)
		for _, key := range keys {
			builder.WriteString(fmt.Sprintf("idx %s unique=%v\n", key, firstUnique(indexes[key])))
		}
	} else {
		builder.WriteString("idx?:0\n")
	}

	sum := sha256.Sum256([]byte(builder.String()))
	summary.Fingerprint = hex.EncodeToString(sum[:])
	return summary
}

// canonicalColumn 单列的可比文本：与 diffColumnPair 逐项比的那几个属性一一对应
func canonicalColumn(dbType dbi.DbType, column dbi.Column) string {
	// 表达式默认值跨方言不可还原，比对不比它，指纹也不记它
	defaultValue := ""
	if !column.IsExprDefault {
		defaultValue = normalizeDefault(column.ColumnDefault)
	}
	return strings.Join([]string{
		strings.ToLower(column.ColumnName),
		strings.ToLower(describeColumn(dbType, column)),
		fmt.Sprint(column.Nullable),
		defaultValue,
		fmt.Sprint(column.AutoIncrement),
	}, "|")
}

// 漂移变化的种类标识，由调用方（应用层）按 i18n 渲染成文案：
// 纯函数层不拼人读文本，否则文案就进了单测不覆盖的层，且存进库的会是当时语言的句子
const (
	// DriftKindColumnCount 列数变化（From/To 为数量）
	DriftKindColumnCount = "columnCount"
	// DriftKindPrimaryKeys 主键列变化（From/To 为有序列名）
	DriftKindPrimaryKeys = "primaryKeys"
	// DriftKindIndexCount 索引数量变化（From/To 为数量）
	DriftKindIndexCount = "indexCount"
	// DriftKindColumnDefs 列定义或列顺序变化（指纹变了但上几项都没变时的归因，无 From/To）
	DriftKindColumnDefs = "columnDefs"
)

// DriftChange 一处结构变化
type DriftChange struct {
	Kind string
	From string
	To   string
}

// DriftChanges 列出两个摘要之间的具体变化。
//
// 指纹不一致而这里只能给出 DriftKindColumnDefs 一项时，说变的是列定义/列顺序：
// 不能因为拼不出更细的明细就返回空列表，那等于把已确认的漂移说成没变
func DriftChanges(baseline, current TableSummary) []DriftChange {
	changes := make([]DriftChange, 0, 4)
	if baseline.ColumnCount != current.ColumnCount {
		changes = append(changes, DriftChange{Kind: DriftKindColumnCount, From: fmt.Sprint(baseline.ColumnCount), To: fmt.Sprint(current.ColumnCount)})
	}
	if baseline.PrimaryKeysAvailable && current.PrimaryKeysAvailable && baseline.PrimaryKeys != current.PrimaryKeys {
		changes = append(changes, DriftChange{Kind: DriftKindPrimaryKeys, From: baseline.PrimaryKeys, To: current.PrimaryKeys})
	}
	if baseline.IndexesAvailable && current.IndexesAvailable && baseline.IndexCount != current.IndexCount {
		changes = append(changes, DriftChange{Kind: DriftKindIndexCount, From: fmt.Sprint(baseline.IndexCount), To: fmt.Sprint(current.IndexCount)})
	}
	if len(changes) == 0 {
		changes = append(changes, DriftChange{Kind: DriftKindColumnDefs})
	}
	return changes
}
