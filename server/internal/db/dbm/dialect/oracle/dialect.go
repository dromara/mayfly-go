package oracle

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/oracle"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
	"strings"
	"time"
)

var _ dbi.Dialect = (*OracleDialect)(nil)

type OracleDialect struct {
	dbi.DefaultDialect

	di *dbi.DbInfo
}

func (od *OracleDialect) CopyTable(copy *dbi.DbCopyTable) error {
	quote := od.Quoter().QuoteIdent
	// 生成新表名,为老表明+_copy_时间戳
	newTableName := strings.ToUpper(copy.TableName + "_copy_" + time.Now().Format("20060102150405"))
	condition := ""
	if !copy.CopyData {
		condition = " where 1 = 2"
	}
	_, err := od.di.Exec(fmt.Sprintf("create table %s as select * from %s%s", quote(newTableName), quote(copy.TableName), condition))
	return err
}

func (od *OracleDialect) GetSQLGenerator() dbi.SQLGenerator {
	return &SQLGenerator{
		DefaultSQLGenerator: dbi.DefaultSQLGenerator{QuoterFn: od.Quoter},
		Dialect:             od,
		version:             od.di.Version,
	}
}

func (od *OracleDialect) GetSQLParser() sqlparser.SQLParser {
	return new(oracle.OracleParser)
}

// GetSQLSplitter oracle切割器：q'[..]' 替代引用字面量 + PL-SQL 过程块感知（反斜杠为普通字符）
func (od *OracleDialect) GetSQLSplitter() sqlparser.SQLSplitter {
	return sqlparser.NewSplitter(tokenizer.OracleConfig)
}

// 编译期契约：解析 / 分页改写 / 语句分类三者缺一不可。
// 历史实现把它们做成「可选能力 + 类型断言 + 默认 LIMIT 回退」，漏实现时运行期会静默
// 生成非法 SQL（Oracle/MSSQL 不支持 LIMIT）；现收敛为必需方法，缺实现直接编译失败。
var _ sqlparser.SQLParser = (*oracle.OracleParser)(nil)
