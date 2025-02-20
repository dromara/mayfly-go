package oracle

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/may-fly/cast"
)

const (
	ORACLE11_COLUMN_MA_KEY = "ORACLE11_COLUMN_MA"
)

type OracleMetadata11 struct {
	OracleMetadata
}

// 获取列元信息, 如列名等
func (od *OracleMetadata11) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	dialect := od.dc.GetDialect()
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dialect.Quoter().Trim(val))
	}), ",")

	// 如果表数量超过了1000，需要分批查询
	if len(tableNames) > 1000 {
		columns := make([]dbi.Column, 0)
		for i := 0; i < len(tableNames); i += 1000 {
			end := i + 1000
			if end > len(tableNames) {
				end = len(tableNames)
			}
			tables := tableNames[i:end]
			cols, err := od.GetColumns(tables...)
			if err != nil {
				return nil, err
			}
			columns = append(columns, cols...)
		}
		return columns, nil
	}

	_, res, err := od.dc.Query(fmt.Sprintf(dbi.GetLocalSql(ORACLE_META_FILE, ORACLE11_COLUMN_MA_KEY), tableName))
	if err != nil {
		return nil, err
	}

	columns := make([]dbi.Column, 0)
	for _, re := range res {
		column := dbi.Column{
			TableName:     cast.ToString(re["TABLE_NAME"]),
			ColumnName:    cast.ToString(re["COLUMN_NAME"]),
			DataType:      cast.ToString(re["DATA_TYPE"]),
			CharMaxLength: cast.ToInt(re["CHAR_MAX_LENGTH"]),
			ColumnComment: cast.ToString(re["COLUMN_COMMENT"]),
			Nullable:      cast.ToString(re["NULLABLE"]) == "YES",
			IsPrimaryKey:  cast.ToInt(re["IS_PRIMARY_KEY"]) == 1,
			AutoIncrement: cast.ToInt(re["IS_IDENTITY"]) == 1,
			ColumnDefault: cast.ToString(re["COLUMN_DEFAULT"]),
			NumPrecision:  cast.ToInt(re["NUM_PRECISION"]),
			NumScale:      cast.ToInt(re["NUM_SCALE"]),
		}

		od.dc.GetDbDataType(column.DataType).FixColumn(&column)
		columns = append(columns, column)
	}
	return columns, nil
}

func (od *OracleMetadata11) genColumnBasicSql(column dbi.Column) string {
	dialect := od.dc.GetDialect()
	colName := dialect.Quoter().Quote(column.ColumnName)

	if column.AutoIncrement {
		// 11g以前的版本 如果是自增，自增列数据类型必须是number，不需要设置默认值和空值，建表后设置自增序列
		return fmt.Sprintf(" %s NUMBER", colName)
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	defVal := ""
	if column.ColumnDefault != "" {
		defVal = fmt.Sprintf(" DEFAULT %v", column.ColumnDefault)
	}

	columnSql := fmt.Sprintf(" %s %s%s%s", colName, column.GetColumnType(), defVal, nullAble)
	return columnSql
}

// 11g及以下版本会设置自增序列和触发器
func (od *OracleMetadata11) GenerateTableOtherDDL(tableInfo dbi.Table, quoteTableName string, columns []dbi.Column) []string {
	result := make([]string, 0)
	for _, col := range columns {
		if col.AutoIncrement {
			seqName := fmt.Sprintf("%s_%s_seq", tableInfo.TableName, col.ColumnName)
			trgName := fmt.Sprintf("%s_%s_trg", tableInfo.TableName, col.ColumnName)
			result = append(result, fmt.Sprintf("CREATE SEQUENCE %s START WITH 1 INCREMENT BY 1", seqName))
			result = append(result, fmt.Sprintf("CREATE OR REPLACE TRIGGER %s BEFORE INSERT ON %s FOR EACH ROW WHEN (NEW.%s IS NULL) BEGIN SELECT %s.nextval INTO :new.%s FROM dual; END", trgName, quoteTableName, col.ColumnName, seqName, col.ColumnName))
		}
	}

	return result
}
