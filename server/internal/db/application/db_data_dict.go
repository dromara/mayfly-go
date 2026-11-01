package application

// 数据字典导出：把「这个库有哪些表、每张表有哪些列/索引、注释是什么」导出成
// 可直接交付的文本（CSV / Markdown）。
//
// 用途是合规与交接：评审、迁移评估、数据资产盘点都需要一份不依赖平台界面的清单。
// 刻意只导出**结构事实**，不导出数据；敏感列标注以「敏感数据发现」与「脱敏规则」
// 面板为准——那两处的判定带有角色与实例维度，静态文件里给不出同口径的结论。

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
)

// DictTable 表级事实
type DictTable struct {
	Name    string `json:"name"`
	Comment string `json:"comment"`
	Rows    int    `json:"rows"`
	Bytes   int64  `json:"bytes"`
}

// DictColumn 列级事实
type DictColumn struct {
	Table         string `json:"table"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Comment       string `json:"comment"`
	Nullable      bool   `json:"nullable"`
	PrimaryKey    bool   `json:"primaryKey"`
	AutoIncrement bool   `json:"autoIncrement"`
	Generated     bool   `json:"generated"`
	Default       string `json:"default"`
}

// DictIndex 索引事实
type DictIndex struct {
	Table   string `json:"table"`
	Name    string `json:"name"`
	Columns string `json:"columns"`
	Unique  bool   `json:"unique"`
	Primary bool   `json:"primary"`
}

// DataDict 整个库的结构清单
type DataDict struct {
	Db      string       `json:"db"`
	Tables  []DictTable  `json:"tables"`
	Columns []DictColumn `json:"columns"`
	Indexes []DictIndex  `json:"indexes"`
}

// DataDictApp 数据字典
type DataDictApp interface {
	// Build 读取库结构；tables 为空表示整库
	Build(ctx context.Context, dbId uint64, dbName string, tables []string) (*DataDict, error)
	// RenderCSV 一行一列，便于进 Excel / 评审表
	RenderCSV(dict *DataDict) []byte
	// RenderMarkdown 一表一节，便于贴进文档或工单
	RenderMarkdown(dict *DataDict) []byte
}

func GetDataDictApp() DataDictApp {
	return ioc.Get[DataDictApp]()
}

type dataDictAppImpl struct {
	dbApp Db `inject:"T"`
}

var _ DataDictApp = (*dataDictAppImpl)(nil)

// Build 读取结构
func (a *dataDictAppImpl) Build(ctx context.Context, dbId uint64, dbName string, tables []string) (*DataDict, error) {
	conn, err := a.dbApp.GetDbConn(ctx, dbId, dbName)
	if err != nil {
		return nil, err
	}
	reader := conn.Metadata()

	allTables, err := reader.GetTables()
	if err != nil {
		return nil, err
	}
	wanted := make(map[string]bool, len(tables))
	for _, table := range tables {
		wanted[table] = true
	}
	dict := &DataDict{Db: dbName}
	tableNames := make([]string, 0, len(allTables))
	for _, table := range allTables {
		if len(wanted) > 0 && !wanted[table.TableName] {
			continue
		}
		tableNames = append(tableNames, table.TableName)
		dict.Tables = append(dict.Tables, DictTable{
			Name: table.TableName, Comment: table.TableComment, Rows: table.TableRows, Bytes: table.DataLength,
		})
	}
	biz.IsTrue(len(dict.Tables) > 0, "no table matched")

	// 列：一次批量取（逐表取在大库上是 N 次往返）
	columns, err := reader.GetColumns(tableNames...)
	if err != nil {
		return nil, err
	}
	for _, column := range columns {
		dict.Columns = append(dict.Columns, DictColumn{
			Table:         column.TableName,
			Name:          column.ColumnName,
			Type:          column.ColumnType,
			Comment:       column.ColumnComment,
			Nullable:      column.Nullable,
			PrimaryKey:    column.IsPrimaryKey,
			AutoIncrement: column.AutoIncrement,
			Generated:     column.IsGenerated,
			Default:       column.ColumnDefault,
		})
	}
	sort.SliceStable(dict.Columns, func(i, j int) bool {
		if dict.Columns[i].Table != dict.Columns[j].Table {
			return dict.Columns[i].Table < dict.Columns[j].Table
		}
		return false
	})

	// 索引：方言只提供按表查询，这里按表逐个取；取不到的库跳过（不阻断整份字典）
	for _, tableName := range tableNames {
		indexes, idxErr := reader.GetTableIndex(tableName)
		if idxErr != nil {
			continue
		}
		grouped := dbi.GroupIndexColumns(indexes)
		for _, index := range grouped {
			dict.Indexes = append(dict.Indexes, DictIndex{
				Table:   tableName,
				Name:    index.IndexName,
				Columns: index.ColumnName,
				Unique:  index.IsUnique,
				Primary: index.IsPrimaryKey,
			})
		}
	}
	return dict, nil
}

// RenderCSV 一行一列；表注释在每行重复，便于按表筛选
func (a *dataDictAppImpl) RenderCSV(dict *DataDict) []byte {
	var builder strings.Builder
	// 带 BOM：Excel 直接双击打开中文不乱码
	builder.WriteString("\ufeff")
	builder.WriteString("库,表名,表注释,列名,类型,可空,主键,自增,生成列,默认值,列注释\r\n")
	tableComments := make(map[string]string, len(dict.Tables))
	for _, table := range dict.Tables {
		tableComments[table.Name] = table.Comment
	}
	for _, column := range dict.Columns {
		fmt.Fprintf(&builder, "%s,%s,%s,%s,%s,%t,%t,%t,%t,%s,%s\r\n",
			csvCell(dict.Db), csvCell(column.Table), csvCell(tableComments[column.Table]), csvCell(column.Name),
			csvCell(column.Type), column.Nullable, column.PrimaryKey, column.AutoIncrement, column.Generated,
			csvCell(column.Default), csvCell(column.Comment))
	}
	return []byte(builder.String())
}

// RenderMarkdown 一表一节：表信息 + 列表格 + 索引表格
func (a *dataDictAppImpl) RenderMarkdown(dict *DataDict) []byte {
	var builder strings.Builder
	builder.WriteString("# 数据字典：" + dict.Db + "\n\n")
	fmt.Fprintf(&builder, "共 %d 张表、%d 个列、%d 个索引\n\n", len(dict.Tables), len(dict.Columns), len(dict.Indexes))

	columnsByTable := make(map[string][]DictColumn, len(dict.Tables))
	for _, column := range dict.Columns {
		columnsByTable[column.Table] = append(columnsByTable[column.Table], column)
	}
	indexesByTable := make(map[string][]DictIndex, len(dict.Tables))
	for _, index := range dict.Indexes {
		indexesByTable[index.Table] = append(indexesByTable[index.Table], index)
	}

	for _, table := range dict.Tables {
		builder.WriteString("## " + table.Name + "\n\n")
		if table.Comment != "" {
			builder.WriteString("- 注释：" + table.Comment + "\n")
		}
		fmt.Fprintf(&builder, "- 行数：%d\n- 数据大小：%d 字节\n\n", table.Rows, table.Bytes)
		builder.WriteString("| 列 | 类型 | 可空 | 主键 | 默认值 | 注释 |\n|---|---|---|---|---|---|\n")
		for _, column := range columnsByTable[table.Name] {
			nullable := "否"
			if column.Nullable {
				nullable = "是"
			}
			primary := ""
			if column.PrimaryKey {
				primary = "是"
			}
			fmt.Fprintf(&builder, "| %s | %s | %s | %s | %s | %s |\n",
				column.Name, column.Type, nullable, primary, mdCell(column.Default), mdCell(column.Comment))
		}
		if indexes := indexesByTable[table.Name]; len(indexes) > 0 {
			builder.WriteString("\n| 索引 | 列 | 唯一 | 主键 |\n|---|---|---|---|\n")
			for _, index := range indexes {
				unique, primary := "", ""
				if index.Unique {
					unique = "是"
				}
				if index.Primary {
					primary = "是"
				}
				fmt.Fprintf(&builder, "| %s | %s | %s | %s |\n", index.Name, index.Columns, unique, primary)
			}
		}
		builder.WriteString("\n")
	}
	return []byte(builder.String())
}

// csvCell 转义：逗号/引号/换行都会破坏行列结构
func csvCell(value string) string {
	if !strings.ContainsAny(value, ",\"\n\r") {
		return value
	}
	return "\"" + strings.ReplaceAll(value, "\"", "\"\"") + "\""
}

// mdCell 转义：竖线会破坏表格，换行压成空格
func mdCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " ")
}
