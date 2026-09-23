package oracle

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/spf13/cast"
)

const (
	ORACLE11_COLUMN_MA_KEY = "ORACLE11_COLUMN_MA"
)

type OracleMetadata11 struct {
	OracleMetadata
}

// 获取列元信息, 如列名等
func (od *OracleMetadata11) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	dialect := od.di.GetDialect()
	tableName := strings.Join(collx.ArrayMap[string, string](tableNames, func(val string) string {
		return fmt.Sprintf("'%s'", dbi.QuoteEscape(dialect.Quoter().Trim(val)))
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

	_, res, err := od.di.Query(fmt.Sprintf(metaSQL.Get(ORACLE11_COLUMN_MA_KEY), tableName))
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

		od.di.GetDbDataType(column.DataType).FixColumn(&column)
		// Oracle的DATA_DEFAULT对字面量默认值恒带引号，不带引号的函数/运算形态即表达式默认值
		dbi.MarkExprDefault(&column)
		columns = append(columns, column)
	}
	return columns, nil
}
