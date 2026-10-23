//go:build it

package application

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/importer"
)

const importItAutoIncTable = "it_data_import_autoinc"

// TestITDataImportAutoIncrementRealDialects 真实 mysql/postgres 上验证两遍流式导入的自增列判定：
//   - 整列为空：必须从 INSERT 列清单剔除，由数据库生成主键。PG 的 serial 显式写 NULL 会直接报错，
//     因此这条用例能真正区分「剔除」与「未剔除」；sqlite 的 INTEGER PRIMARY KEY 即使写 NULL 也自增，无法区分。
//   - 显式 id 非空：必须保留，10/20/30 与自增生成的 1/2/3 可区分。
//
// batchSize=2 覆盖跨批次 flush。scratchclean 会为 mysql/pg 建独占临时库并在 TestMain 统一清理。
func TestITDataImportAutoIncrementRealDialects(t *testing.T) {
	run := func(t *testing.T, dialect string, conn *dbi.DbConn, columnsSQL string) {
		t.Helper()
		defer conn.Close()
		quote := conn.GetDialect().Quoter().QuoteIdent

		prepare := func() {
			_, err := conn.Exec("DROP TABLE IF EXISTS " + quote(importItAutoIncTable))
			require.NoError(t, err, dialect)
			// 表名/列定义均为包内常量（非请求输入），表名仍统一经 Quoter 加引号
			_, err = conn.Exec("CREATE TABLE " + quote(importItAutoIncTable) + " (" + columnsSQL + ")")
			require.NoError(t, err, dialect)
		}
		importCSV := func(csv string) *dto.DataImportRes {
			app := &dbDataImportAppImpl{}
			res, err := app.Import(context.Background(), &dto.DataImportReq{
				DbConn:    conn,
				TableName: importItAutoIncTable,
				Reader:    strings.NewReader(csv),
				Filename:  "data.csv",
				Options:   &importer.Options{HasHeader: true},
				Columns: []dto.ImportColumn{
					{Source: "0", Target: "id"},
					{Source: "1", Target: "name"},
				},
				EmptyAsNull: true,
				BatchSize:   2,
			})
			require.NoError(t, err, dialect)
			return res
		}

		// 1) 整列空：库生成 1/2/3（PG 若未剔除会因 serial 显式 NULL 直接报错）
		prepare()
		res := importCSV("id,name\n,alice\n,bob\n,carol\n")
		assert.Equal(t, 3, res.TotalRows, dialect)
		assert.Equal(t, 3, res.Imported, dialect)
		_, rows, err := conn.Query("SELECT id, name FROM " + quote(importItAutoIncTable) + " ORDER BY id")
		require.NoError(t, err, dialect)
		require.Len(t, rows, 3, dialect)
		assert.Equal(t, "alice", importItText(rows[0]["name"]), dialect)
		assert.Equal(t, "carol", importItText(rows[2]["name"]), dialect)
		assert.Equal(t, "1", importItText(rows[0]["id"]), dialect)
		assert.Equal(t, "3", importItText(rows[2]["id"]), dialect)

		// 2) 显式 id：必须原样保留（证明 anyNonEmpty 未误判整列为空）
		prepare()
		res = importCSV("id,name\n10,alice\n20,bob\n30,carol\n")
		assert.Equal(t, 3, res.Imported, dialect)
		_, rows, err = conn.Query("SELECT id, name FROM " + quote(importItAutoIncTable) + " ORDER BY id")
		require.NoError(t, err, dialect)
		require.Len(t, rows, 3, dialect)
		assert.Equal(t, "10", importItText(rows[0]["id"]), dialect)
		assert.Equal(t, "30", importItText(rows[2]["id"]), dialect)
	}

	t.Run("mysql", func(t *testing.T) {
		run(t, "mysql", appMysqlConn(t), "id INT NOT NULL AUTO_INCREMENT PRIMARY KEY, name VARCHAR(64) NOT NULL")
	})
	t.Run("postgres", func(t *testing.T) {
		run(t, "postgres", appPgConn(t), "id SERIAL PRIMARY KEY, name TEXT NOT NULL")
	})
}
