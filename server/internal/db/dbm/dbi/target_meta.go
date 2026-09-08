package dbi

import "strings"

// BuildTargetTableMeta 构建目标表元信息，用于生成 upsert/merge 类插入语句（保证后续生成SQL过程中不再查询数据库）
// 冲突检测列优先取表主键；无主键且仅存在一个唯一索引时取该索引列；否则 UniqueColumns 为空，GenInsert 会退化为直接插入
func BuildTargetTableMeta(targetConn *DbConn, tableName string, columns []Column) *TargetTableMeta {
	meta := &TargetTableMeta{}
	var uniqueCols []string
	var identityCols []string
	for _, column := range columns {
		if column.IsPrimaryKey {
			uniqueCols = append(uniqueCols, column.ColumnName)
		}
		if column.AutoIncrement {
			identityCols = append(identityCols, column.ColumnName)
		}
	}
	if len(uniqueCols) == 0 {
		// 无主键时尝试取唯一索引，且仅当只存在一个唯一索引时才可作为冲突检测列（多个唯一索引无法确定冲突语义）
		indexs, err := targetConn.GetMetadata().GetTableIndex(tableName)
		if err == nil {
			var uniqueIndexes []Index
			for _, index := range indexs {
				if index.IsUnique {
					uniqueIndexes = append(uniqueIndexes, index)
				}
			}
			if len(uniqueIndexes) == 1 {
				for _, col := range strings.Split(uniqueIndexes[0].ColumnName, ",") {
					trimmed := strings.TrimSpace(col)
					if trimmed != "" {
						uniqueCols = append(uniqueCols, trimmed)
					}
				}
			}
		}
	}
	meta.UniqueColumns = uniqueCols
	meta.IdentityColumns = identityCols
	return meta
}
