package transfer

// 本文件为数据库dump导出核心逻辑：将源库表结构/数据导出为目标方言SQL脚本。
// 供主包DbAppImpl的DumpDb功能与本包迁移/校验链路（集成测试直驱真实dump）共用，
// 与导入侧importDumpStream构成完整迁移引擎

import (
	"cmp"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/writerx"
)

// DumpDbScript dump核心逻辑：将dbConn指向的源库表结构/数据导出为目标方言SQL脚本。
//
// 源连接由参数注入而非内部按DbId获取，便于集成测试直接驱动真实dump链路（不经实例/权限体系），
// 也便于迁移等上层已在外部持有连接的场景复用
func DumpDbScript(ctx context.Context, dbConn *dbi.DbConn, reqParam *dto.DumpDb) error {
	log := dto.DefaultDumpLog
	if reqParam.Log != nil {
		log = reqParam.Log
	}
	progress := dto.DefaultDumpProgress
	if reqParam.Progress != nil {
		progress = reqParam.Progress
	}

	writer := writerx.NewStringWriter(reqParam.Writer)
	// writeDump写入dump文本。写入失败（如HTTP响应流中断、磁盘写满）必须显式失败，
	// 否则dump会在writer已不可用的状态下继续空转，产出不可用的部分文件且无任何报错，
	// 属备份场景下不可接受的静默损坏
	writeDump := func(text string) error {
		_, err := writer.WriteString(text)
		return err
	}

	dbName := reqParam.DbName
	tables := reqParam.Tables

	writeDumpHeader := strings.Join([]string{
		"\n-- ----------------------------",
		"\n-- Dump Platform: mayfly-go",
		fmt.Sprintf("\n-- Dump Time: %s ", time.Now().Format("2006-01-02 15:04:05")),
		fmt.Sprintf("\n-- Dump DB: %s ", dbi.SanitizeCommentText(dbName)),
		fmt.Sprintf("\n-- DB Dialect: %s ", cmp.Or(reqParam.TargetDbType, dbConn.Info.Type)),
		"\n-- ----------------------------\n\n",
	}, "")
	if err := writeDump(writeDumpHeader); err != nil {
		return err
	}

	// 获取目标元数据，仅生成sql，用于生成建表语句和插入数据，不能用于查询
	targetDialect := dbConn.GetDialect()
	if reqParam.TargetDbType != "" && dbConn.Info.Type != reqParam.TargetDbType {
		targetDialect = dbi.GetDialect(reqParam.TargetDbType)
	}

	srcMeta := dbConn.GetMetadata()
	srcDialect := dbConn.GetDialect()
	if len(tables) == 0 {
		log("gets the table information that can be export...")
		ti, err := srcMeta.GetTables()
		if err != nil {
			log(fmt.Sprintf("failed to get table info %s", err.Error()))
			return err
		}
		tables = make([]string, len(ti))
		for i, table := range ti {
			tables[i] = table.TableName
		}
		log(fmt.Sprintf("Get %d tables", len(tables)))
	}
	if len(tables) == 0 {
		log("no table to export. end export")
		return errorx.NewBiz("there is no table to export")
	}

	log("querying column information...")
	// 查询列信息，后面生成建表ddl和insert都需要列信息
	columns, err := srcMeta.GetColumns(tables...)
	if err != nil {
		log(fmt.Sprintf("failed to query column information: %s", err.Error()))
		return err
	}

	// 以表名分组，存放每个表的列信息
	columnMap := make(map[string][]dbi.Column)
	for _, column := range columns {
		if err := dbi.ConvToTargetDbColumn(dbConn.Info.Type, cmp.Or(reqParam.TargetDbType, dbConn.Info.Type), targetDialect, &column); err != nil {
			return err
		}
		columnMap[column.TableName] = append(columnMap[column.TableName], column)
	}

	// 按表名排序
	sort.Strings(tables)
	targetDumpHelper := targetDialect.GetDumpHelper()
	targetSqlGenerator := targetDialect.GetSQLGenerator()

	srcDialectQuote := srcDialect.Quoter().QuoteIdent
	// 遍历获取每个表的信息
	for _, tableName := range tables {
		log(fmt.Sprintf("get table [%s] information...", tableName))
		// targetQuoteTableName := targetDialectQuote(tableName)
		srcQuoteTableName := srcDialectQuote(tableName)

		// 查询表信息，主要是为了查询表注释
		tbs, err := srcMeta.GetTables(tableName)
		if err != nil {
			log(fmt.Sprintf("failed to get table [%s] information: %s", tableName, err.Error()))
			return err
		}
		if len(tbs) <= 0 {
			log(fmt.Sprintf("failed to get table [%s] information: No table information was retrieved", tableName))
			return errorx.NewBizf("Failed to get table information: %s", tableName)
		}

		tableInfo := tbs[0]
		columns := columnMap[tableName]
		// 列信息缺失时生成的DDL/INSERT必然为空表结构或空值元组，属静默丢表丢数据，必须显式失败
		if len(columns) == 0 {
			log(fmt.Sprintf("failed to get column information for table [%s]: no columns retrieved", tableName))
			return errorx.NewBizf("failed to get the columns information of the table [%s]", tableName)
		}

		// 生成表结构信息
		if reqParam.DumpDDL {
			log(fmt.Sprintf("generate table [%s] DDL...", tableName))
			if err := writeDump(fmt.Sprintf("\n-- ----------------------------\n-- Table structure: %s \n-- ----------------------------\n", dbi.SanitizeCommentText(tableName))); err != nil {
				return err
			}
			tbDdlArr := targetSqlGenerator.GenTableDDL(tableInfo, columns, true)
			for _, ddl := range tbDdlArr {
				if _, err := writer.WriteString(ddl + ";\n"); err != nil {
					return err
				}
			}
			progress(tableName, dbi.StmtTypeDDL, len(tbDdlArr), true)
		}

		// 生成insert sql，数据在索引前，加速insert
		if reqParam.DumpData {
			log(fmt.Sprintf("generate table [%s] DML...", tableName))
			if err := writeDump(fmt.Sprintf("\n-- ----------------------------\n-- Data: %s \n-- ----------------------------\n", dbi.SanitizeCommentText(tableName))); err != nil {
				return err
			}

			// 导出场景无需处理冲突，直接生成插入语句
			if err := targetDumpHelper.BeforeInsert(writer, tableName); err != nil {
				return err
			}

			dataCount := 0
			rows := make([][]any, 0)
			// 行数与字节数双预算：固定100行分批在单行大值（如大blob/text）场景下会
			// 生成超大单条INSERT（100行×1MB=100MB），导致内存峰值暴涨，且导入时超
			// mysql max_allowed_packet等单包上限直接失败；累计字节超预算时提前flush
			pendingBytes := 0
			// 分片过滤：TableFilter含该表时仅导出满足where条件的数据（大表主键分片）
			walkSql := fmt.Sprintf("SELECT * FROM %s", srcQuoteTableName)
			if where := reqParam.TableFilter[tableName]; where != "" {
				walkSql += " WHERE " + where
			}
			flushRows := func() error {
				if len(rows) == 0 {
					return nil
				}
				// BeforeInsertSql由目标方言helper自行quote表名并判断自增列，
				// 不能传入源库schema（源schema在目标库中通常不存在）
				beforeInsert := targetDumpHelper.BeforeInsertSql(tableName, columns)
				if beforeInsert != "" {
					writer.WriteString(beforeInsert)
				}
				insertSql := targetSqlGenerator.GenInsert(tableName, columns, rows, dbi.DuplicateStrategyNone, nil)
				if _, err := writer.WriteString(strings.Join(insertSql, ";\n") + ";\n"); err != nil {
					return err
				}
				rows = make([][]any, 0)
				pendingBytes = 0
				return nil
			}
			// colKeys为元数据列在查询结果row中的key，首行时解析（见resolveDumpColumnKeys）；
			// 未提前解析时按元数据列名直接索引row，列名大小写/驱动命名差异会静默取到nil并导出NULL（丢数据）
			var colKeys []string
			_, err = dbConn.WalkQueryRows(ctx, walkSql, func(row map[string]any, queryCols []*dbi.QueryColumn) error {
				if colKeys == nil {
					resolvedKeys, err := resolveDumpColumnKeys(columns, queryCols)
					if err != nil {
						return err
					}
					colKeys = resolvedKeys
				}
				rowValues := make([]any, len(columns))
				rowBytes := 0
				for i, key := range colKeys {
					rowValues[i] = row[key]
					switch v := rowValues[i].(type) {
					case string:
						rowBytes += len(v)
					case []byte:
						rowBytes += len(v)
					default:
						rowBytes += 16
					}
				}
				rows = append(rows, rowValues)
				dataCount++
				pendingBytes += rowBytes
				if dataCount%dbi.DumpInsertBatchRows != 0 && pendingBytes < dbi.DumpInsertBatchBytes {
					return nil
				}

				if err := flushRows(); err != nil {
					return err
				}
				progress(tableName, dbi.StmtTypeInsert, dataCount, false)
				return nil
			})

			if err != nil {
				return err
			}

			if err := flushRows(); err != nil {
				return err
			}

			if err := targetDumpHelper.AfterInsert(writer, tableName, columns); err != nil {
				return err
			}
			progress(tableName, dbi.StmtTypeInsert, dataCount, true)
		}

		log(fmt.Sprintf("get table [%s] index information...", tableName))
		indexs, err := srcMeta.GetTableIndex(tableName)
		if err != nil {
			log(fmt.Sprintf("failed to get table [%s] index information: %s", tableName, err.Error()))
			return err
		}

		if len(indexs) > 0 {
			// 最后添加索引
			log(fmt.Sprintf("generate table [%s] index...", tableName))
			if err := writeDump(fmt.Sprintf("\n-- ----------------------------\n-- Table Index: %s \n-- ----------------------------\n", dbi.SanitizeCommentText(tableName))); err != nil {
				return err
			}
			sqlArr := targetSqlGenerator.GenIndexDDL(tableInfo, indexs)
			for _, sqlStr := range sqlArr {
				if _, err := writer.WriteString(sqlStr + ";\n"); err != nil {
					return err
				}
			}
			progress(tableName, dbi.StmtTypeDDL, len(sqlArr), true)
		}
	}

	return nil
}

// resolveDumpColumnKeys 解析元数据列与查询结果列的对应关系，返回每个元数据列在row map中的key。
//
// dump按列顺序组装INSERT的VALUES，若仅以元数据列名直接索引row，当驱动返回的列标签与
// 元数据中的列名存在大小写/命名差异（如部分驱动将列名转大写）时，索引不命中会静默得到nil，
// 导出为NULL——属不可逆的数据丢失，故此处先精确匹配、再大小写不敏感匹配，仍不命中则报错终止
func resolveDumpColumnKeys(columns []dbi.Column, queryCols []*dbi.QueryColumn) ([]string, error) {
	exact := make(map[string]string, len(queryCols))
	folded := make(map[string]string, len(queryCols))
	for _, qc := range queryCols {
		exact[qc.Name] = qc.Key
		folded[strings.ToLower(qc.Name)] = qc.Key
	}

	keys := make([]string, len(columns))
	for i, column := range columns {
		key, ok := exact[column.ColumnName]
		if !ok {
			key, ok = folded[strings.ToLower(column.ColumnName)]
		}
		if !ok {
			return nil, errorx.NewBizf("column [%s] of table [%s] not found in the query result columns, refuse to dump with unknown values",
				column.ColumnName, dbi.SanitizeCommentText(column.TableName))
		}
		keys[i] = key
	}
	return keys, nil
}
