package itest

// 库名/schema 枚举的系统库排除回归。
//
// 系统库一旦混入枚举结果，前端资源树会把它们当成业务库/schema 展开（mysql 的 sys.sys_config、
// mssql 的 master.sys.databases、oracle 的 SYS 数据字典），用户既无法在此建业务对象，
// 又会被大量只读系统视图淹没。这里的断言落在真实实例返回的清单上，而不是 SQL 文本。

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestITDbNamesExcludeSystemSchemas(t *testing.T) {
	t.Run("mysql", func(t *testing.T) {
		conn := mysqlConn(t)
		defer conn.Close()

		names, err := conn.Metadata().GetDbNames()
		require.NoError(t, err)
		require.NotEmpty(t, names, "实例上必然存在业务库，空清单说明枚举本身失效")
		for _, sys := range []string{"mysql", "information_schema", "performance_schema", "sys"} {
			assert.NotContains(t, names, sys, "系统库 %q 不得出现在库名列表", sys)
		}
	})

	t.Run("mssql", func(t *testing.T) {
		conn := gcMssqlConn(t)
		defer conn.Close()

		names, err := conn.Metadata().GetDbNames()
		require.NoError(t, err)
		require.NotEmpty(t, names, "实例上必然存在业务库，空清单说明枚举本身失效")
		for _, sys := range []string{"master", "tempdb", "model", "msdb"} {
			assert.NotContains(t, names, sys, "系统库 %q 不得出现在库名列表", sys)
		}
	})

	t.Run("oracle", func(t *testing.T) {
		conn := oracleConn(t)
		defer conn.Close()

		schemas, err := conn.Metadata().GetSchemas()
		require.NoError(t, err)
		require.NotEmpty(t, schemas, "schema 列表不得为空")
		// SYS/SYSTEM 由 catalog.sql 创建，ORACLE_MAINTAINED='Y'，只应作为数据字典所在 schema 存在
		for _, sys := range []string{"SYS", "SYSTEM", "CTXSYS", "MDSYS", "XDB"} {
			assert.NotContains(t, schemas, sys, "内置账号 %q 不得出现在 schema 列表", sys)
		}
		assert.Contains(t, schemas, strings.ToUpper(itOracleUser), "业务 schema 必须保留")
	})
}
