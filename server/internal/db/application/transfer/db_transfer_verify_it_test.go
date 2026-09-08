//go:build it

package transfer

// 数据校验真实链路集成测试：verifyTable对真实库做count比对与头/尾抽样内容比对。
//
// 运行方式：cd server && go test -tags it -count=1 -v ./internal/db/application/
//
// 注意：dbm.Conn对相同连接信息复用同一连接池，故制造"目标与源存在差异"的场景
// 必须使用两个不同数据库（mysql源 + pg目标）。

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

const verifyItTable = "it_verify_src"

func verifyMysqlConn(t *testing.T) *dbi.DbConn {
	return transferTestConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
}

func verifyPgConn(t *testing.T) *dbi.DbConn {
	return transferTestConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
}

// setupVerifyPair 在mysql源与pg目标建同名表并各插入n行
func setupVerifyPair(t *testing.T, n int) (srcConn, tgtConn *dbi.DbConn) {
	t.Helper()
	srcConn = verifyMysqlConn(t)
	tgtConn = verifyPgConn(t)

	require.NoError(t, (&DbTransferAppImpl{}).importDumpStream(context.Background(), 0, srcConn,
		strings.NewReader(buildDumpScript(srcConn.GetDialect().Quoter().Quote, verifyItTable, n))))
	require.NoError(t, (&DbTransferAppImpl{}).importDumpStream(context.Background(), 0, tgtConn,
		strings.NewReader(buildDumpScript(tgtConn.GetDialect().Quoter().Quote, verifyItTable, n))))
	return srcConn, tgtConn
}

// TestITVerifyTable_AllMatch 数据一致 → count匹配且抽样无差异（mysql→pg跨方言）
func TestITVerifyTable_AllMatch(t *testing.T) {
	srcConn, tgtConn := setupVerifyPair(t, 600)
	defer srcConn.Close()
	defer tgtConn.Close()

	res := (&DbTransferAppImpl{}).verifyTable(context.Background(), srcConn, tgtConn, verifyItTable)
	require.Empty(t, res.Err)
	assert.True(t, res.CountMatch)
	assert.Equal(t, int64(600), res.SrcCount)
	assert.Equal(t, int64(600), res.TargetCount)
	assert.Greater(t, res.Sampled, 0)
	assert.Empty(t, res.MismatchPk, "跨方言等值行经规范化后不应误报差异")
}

// TestITVerifyTable_CountMismatch 目标删一行+改一个值 → count不一致且mismatchPk命中
func TestITVerifyTable_CountMismatch(t *testing.T) {
	srcConn, tgtConn := setupVerifyPair(t, 600)
	defer srcConn.Close()
	defer tgtConn.Close()

	quote := tgtConn.GetDialect().Quoter().Quote
	// 删除id=1行（头部抽样覆盖）+ 篡改id=599的val（尾部抽样覆盖，头部300行不含）
	_, err := tgtConn.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = 1", quote(verifyItTable)))
	require.NoError(t, err)
	_, err = tgtConn.Exec(fmt.Sprintf("UPDATE %s SET val = 'tampered' WHERE id = 599", quote(verifyItTable)))
	require.NoError(t, err)

	res := (&DbTransferAppImpl{}).verifyTable(context.Background(), srcConn, tgtConn, verifyItTable)
	require.Empty(t, res.Err)
	assert.False(t, res.CountMatch, "删除一行后count应不一致")
	assert.Equal(t, int64(600), res.SrcCount)
	assert.Equal(t, int64(599), res.TargetCount)
	assert.Contains(t, res.MismatchPk, "1", "被删行应被头部抽样检出")
	assert.Contains(t, res.MismatchPk, "599", "被篡改行应被尾部抽样检出")
}

// TestITVerifyTable_NoPkTable 无主键表 → count比对照常、内容抽样跳过并记录原因
func TestITVerifyTable_NoPkTable(t *testing.T) {
	conn := verifyPgConn(t)
	defer conn.Close()
	quote := conn.GetDialect().Quoter().Quote

	table := "it_verify_nopk"
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	require.NoError(t, err)
	_, err = conn.Exec(fmt.Sprintf("CREATE TABLE %s (code varchar(32), val varchar(32))", quote(table)))
	require.NoError(t, err)
	_, err = conn.Exec(fmt.Sprintf("INSERT INTO %s VALUES ('c1', 'v1'), ('c2', 'v2')", quote(table)))
	require.NoError(t, err)

	res := (&DbTransferAppImpl{}).verifyTable(context.Background(), conn, conn, table)
	require.Empty(t, res.Err)
	assert.True(t, res.CountMatch)
	assert.Equal(t, int64(2), res.SrcCount)
	assert.Empty(t, res.MismatchPk)
	assert.NotEmpty(t, res.SampleErr, "无主键应记录抽样跳过原因")
}
