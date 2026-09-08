//go:build it

package mask

// 脱敏功能跨方言运行时验证（集成测试）：基于本地 MySQL(3306) 与 PostgreSQL(5432) 真实执行查询，
// 验证 真实方言SQL解析 → walkQueryRows真实列元数据(含重名列改名) → buildRowMaskerFromPlan → MaskRow → Masked标识与json透传 的完整链路。
//
// 覆盖场景：基础命中 / 别名列(AS p) / 反引号别名 / 双引号别名 / 大写列名 / JOIN重名列 / 表限定列 / 表达式列(COUNT(*)) / 豁免标签 / 绑定标签算法覆盖 / masked json透传
//
// 运行方式：cd server && go test -tags it -count=1 -v -run TestMaskIT ./internal/db/application/
//
// 环境依赖：docker mysql:8.0(localhost:3306, root/111049) 与 mayfly-pg-it 容器(postgres:16, localhost:5432, postgres/postgres)

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	_ "mayfly-go/internal/db/dbm/mysql" // 注册mysql方言
	_ "mayfly-go/internal/db/dbm/postgres"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	masksvc "mayfly-go/internal/db/domain/mask"
)

const maskItMysqlDatabase = "mayfly_dbm_it"

const maskItPgDatabase = "mayfly_pg_it"

func maskItCtx() context.Context {
	return context.Background()
}

func maskMysqlConn(t *testing.T) *dbi.DbConn {
	t.Helper()
	// 先连server确保库存在
	serverConn, err := dbm.Conn(maskItCtx(), &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "information_schema"})
	require.NoError(t, err)
	defer serverConn.Close()
	_, err = serverConn.Exec("CREATE DATABASE IF NOT EXISTS `" + maskItMysqlDatabase + "` DEFAULT CHARSET utf8mb4")
	require.NoError(t, err)

	conn, err := dbm.Conn(maskItCtx(), &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: maskItMysqlDatabase})
	require.NoError(t, err)
	return conn
}

func maskPgConn(t *testing.T) *dbi.DbConn {
	t.Helper()
	conn, err := dbm.Conn(maskItCtx(), &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: maskItPgDatabase})
	require.NoError(t, err)
	require.NoError(t, conn.Ping())
	return conn
}

func maskItExec(t *testing.T, conn *dbi.DbConn, sql string) {
	t.Helper()
	if _, err := conn.Exec(sql); err != nil {
		t.Fatalf("exec failed [%s]: %s", sql, err.Error())
	}
}

func setupMaskMysqlTables(t *testing.T, conn *dbi.DbConn) {
	t.Helper()
	for _, ddl := range []string{
		"DROP TABLE IF EXISTS t_user",
		"CREATE TABLE t_user (id int NOT NULL PRIMARY KEY, phone varchar(20), email varchar(64), address varchar(128)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		"INSERT INTO t_user (id, phone, email, address) VALUES (1, '13800001234', 'zhangsan@example.com', '北京市朝阳区')",
		"DROP TABLE IF EXISTS t_order",
		"CREATE TABLE t_order (id int NOT NULL PRIMARY KEY, user_id int, phone varchar(20), amount decimal(10,2))",
		"INSERT INTO t_order (id, user_id, phone, amount) VALUES (1, 1, '13900002345', 99.50)",
		"DROP TABLE IF EXISTS t_log",
		"CREATE TABLE t_log (id int NOT NULL PRIMARY KEY, phone varchar(20))",
		"INSERT INTO t_log (id, phone) VALUES (1, '13700003456')",
	} {
		maskItExec(t, conn, ddl)
	}
}

func setupMaskPgTables(t *testing.T, conn *dbi.DbConn) {
	t.Helper()
	for _, ddl := range []string{
		"DROP TABLE IF EXISTS t_mask_user",
		`CREATE TABLE t_mask_user ("ID" serial PRIMARY KEY, "PHONE" varchar(20), "USER_NAME" varchar(64), "ADDRESS" varchar(128))`,
		`INSERT INTO t_mask_user ("PHONE", "USER_NAME", "ADDRESS") VALUES ('13800001234', 'zhangsan', '北京市朝阳区')`,
		"DROP TABLE IF EXISTS t_log",
		`CREATE TABLE t_log ("ID" int PRIMARY KEY, "PHONE" varchar(20))`,
		`INSERT INTO t_log ("ID", "PHONE") VALUES (1, '13700003456')`,
	} {
		maskItExec(t, conn, ddl)
	}
}

// maskItPlan 构建真实脱敏计划：全局规则（正则手机号+精确姓名）与列标签（t_user.phone绑定hash、t_log整表豁免）
func maskItPlan(t *testing.T) *masksvc.Plan {
	t.Helper()
	mustAlg := func(name string) masksvc.Algorithm {
		alg, err := masksvc.Get(name)
		require.NoError(t, err)
		return alg
	}
	plan, err := masksvc.NewPlan([]*masksvc.Rule{
		{Name: "手机号", MatchType: masksvc.MatchTypeRegex, Pattern: `(?i)(phone|mobile|tel)`, Algorithm: mustAlg(masksvc.AlgoPhone)},
		{Name: "邮箱", MatchType: masksvc.MatchTypeRegex, Pattern: `(?i)email`, Algorithm: mustAlg(masksvc.AlgoEmail)},
		{Name: "姓名", MatchType: masksvc.MatchTypeExact, Pattern: "user_name", Algorithm: mustAlg(masksvc.AlgoFull)},
	}, []*masksvc.ColumnTag{
		{TableName: "t_user", ColumnName: "phone", Action: masksvc.TagActionBind, Algorithm: mustAlg(masksvc.AlgoHash)},
		{TableName: "t_log", Action: masksvc.TagActionExempt},
	})
	require.NoError(t, err)
	return plan
}

// maskItHash 期望的绑定标签hash脱敏值（hashAlgo默认md5/keepLen=8）
func maskItHash(v string) string {
	sum := md5.Sum([]byte(v))
	return hex.EncodeToString(sum[:])[:8]
}

// runMaskQuery 模拟doQuery的真实链路：WalkQueryRows逐行回调内构建脱敏器并脱敏，返回行数据与列元数据
func runMaskQuery(t *testing.T, conn *dbi.DbConn, plan *masksvc.Plan, sql string) ([]map[string]any, []*dbi.QueryColumn) {
	t.Helper()
	var sel *sqlstmt.SelectStmt
	if stmt, err := conn.GetDialect().GetSQLParser().Parse(sql); err == nil {
		if s, ok := stmt.(*sqlstmt.SelectStmt); ok {
			sel = s
		}
	}
	rows := make([]map[string]any, 0)
	cols, err := conn.WalkQueryRows(maskItCtx(), sql, func(row map[string]any, columns []*dbi.QueryColumn) error {
		m := buildRowMaskerFromPlan(plan, conn.Info.GetDatabase(), sel, columns, conn.GetDialect().GetSQLParser())
		if m != nil {
			m.MaskRow(row)
		}
		rows = append(rows, row)
		return nil
	})
	require.NoError(t, err, "walk query rows failed: %s", sql)
	return rows, cols
}

// assertColMasked 断言指定列的Masked标识
func assertColMasked(t *testing.T, cols []*dbi.QueryColumn, expect map[string]bool) {
	t.Helper()
	for _, col := range cols {
		if e, ok := expect[col.Key]; ok {
			assert.Equal(t, e, col.Masked, "列[%s](Key=%s)的Masked标识不符", col.Name, col.Key)
		}
	}
}

// checkMaskedJson 断言masked标识的json透传行为：脱敏列携带"masked":true，未脱敏列因omitempty不携带
func checkMaskedJson(t *testing.T, cols []*dbi.QueryColumn) {
	t.Helper()
	for _, col := range cols {
		data, err := json.Marshal(col)
		require.NoError(t, err)
		if col.Masked {
			assert.Contains(t, string(data), `"masked":true`, "脱敏列json应携带masked标识")
		} else {
			assert.NotContains(t, string(data), `"masked"`, "未脱敏列json不应携带masked字段")
		}
	}
}

// TestMaskITMysql MySQL方言运行时验证：基础命中/别名/反引号别名/表限定列/JOIN重名列/表达式列/豁免标签
func TestMaskITMysql(t *testing.T) {
	conn := maskMysqlConn(t)
	defer conn.Close()
	setupMaskMysqlTables(t, conn)
	plan := maskItPlan(t)

	// 1. 基础命中：phone命中t_user绑定标签(hash)，email命中正则规则(phone算法)，address不脱敏
	rows, cols := runMaskQuery(t, conn, plan, "SELECT phone, email, address FROM t_user")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"], "phone应命中绑定标签使用hash算法")
	assert.NotEqual(t, "zhangsan@example.com", rows[0]["email"], "email应被脱敏")
	assert.Equal(t, "北京市朝阳区", rows[0]["address"], "address不应脱敏")
	assertColMasked(t, cols, map[string]bool{"phone": true, "email": true, "address": false})
	checkMaskedJson(t, cols)

	// 2. 别名列 SELECT phone AS p：通过别名映射脱敏
	rows, cols = runMaskQuery(t, conn, plan, "SELECT phone AS p FROM t_user")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["p"], "别名列p应通过别名映射命中绑定标签")
	assertColMasked(t, cols, map[string]bool{"p": true})
	checkMaskedJson(t, cols)

	// 3. 反引号中文别名：别名需去除引用符后命中映射
	rows, cols = runMaskQuery(t, conn, plan, "SELECT phone AS `手机号` FROM t_user")
	require.Len(t, rows, 1)
	require.Contains(t, rows[0], "手机号")
	assert.Equal(t, maskItHash("13800001234"), rows[0]["手机号"], "反引号别名应去除引用符后命中映射")
	assertColMasked(t, cols, map[string]bool{"手机号": true})

	// 4. 表限定列（无别名）SELECT u.phone FROM t_user u：按结果列名兜底匹配
	rows, cols = runMaskQuery(t, conn, plan, "SELECT u.phone FROM t_user u")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"])
	assertColMasked(t, cols, map[string]bool{"phone": true})

	// 5. 表别名+反引号限定列 SELECT u.`phone` FROM t_user u：别名映射尾段需去引号
	rows, cols = runMaskQuery(t, conn, plan, "SELECT u.`phone` AS `p` FROM t_user u")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["p"], "反引号限定列+反引号别名应正确映射")
	assertColMasked(t, cols, map[string]bool{"p": true})

	// 6. JOIN重名列（无AS别名）：u.phone与o.phone结果同名，walkQueryRows将第二列Key改名为phoneN；
	// 位置对应模式下两列均精确归属：第一列命中t_user标签(hash)，第二列归属t_order走全局phone算法
	rows, cols = runMaskQuery(t, conn, plan, "SELECT u.phone, o.phone, o.amount FROM t_user u JOIN t_order o ON o.user_id = u.id")
	require.Len(t, rows, 1)
	require.Len(t, cols, 3)
	require.NotEqual(t, cols[0].Key, cols[1].Key, "重名列应有唯一Key")
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"], "第一列phone应精确归属t_user命中hash标签")
	assert.Equal(t, "139****2345", rows[0][cols[1].Key], "第二列phone应精确归属t_order走全局phone算法，而非t_user的hash标签")
	assertColMasked(t, cols, map[string]bool{"phone": true, cols[1].Key: true, "amount": false})
	checkMaskedJson(t, cols)

	// 6.1 JOIN限定名+别名：逐表精确归属——up命中t_user标签(hash)，op归属t_order无标签
	// 但命中全局phone正则规则(phone算法)，不应错用t_user的hash标签；amt不脱敏
	rows, cols = runMaskQuery(t, conn, plan,
		"SELECT u.phone AS up, o.phone AS op, o.amount AS amt FROM t_user u JOIN t_order o ON o.user_id = u.id")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["up"], "up应精确归属t_user命中hash标签")
	assert.Equal(t, "139****2345", rows[0]["op"], "op应精确归属t_order走全局phone算法，而非t_user的hash标签")
	assert.NotEqual(t, maskItHash("13900002345"), rows[0]["op"], "op不应错误归属到t_user的hash标签")
	assertColMasked(t, cols, map[string]bool{"up": true, "op": true, "amt": false})
	checkMaskedJson(t, cols)

	// 7. 表达式列COUNT(*)：无法溯源源列，按结果列名兜底不命中
	rows, cols = runMaskQuery(t, conn, plan, "SELECT COUNT(*) AS cnt FROM t_user")
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0]["cnt"])
	assertColMasked(t, cols, map[string]bool{"cnt": false})
	checkMaskedJson(t, cols)

	// 8. 豁免标签：t_log整表豁免，phone不脱敏
	rows, cols = runMaskQuery(t, conn, plan, "SELECT phone FROM t_log")
	require.Len(t, rows, 1)
	assert.Equal(t, "13700003456", rows[0]["phone"], "t_log命中整表豁免标签不应脱敏")
	assertColMasked(t, cols, map[string]bool{"phone": false})

	// 9. 子查询表名：TableRef.Name为原始文本，应跳过且不影响列名兜底匹配
	rows, cols = runMaskQuery(t, conn, plan, "SELECT phone FROM (SELECT phone FROM t_user) tmp")
	require.Len(t, rows, 1)
	assert.NotEqual(t, "13800001234", rows[0]["phone"], "子查询结果列phone应按列名正则兜底脱敏")
	assertColMasked(t, cols, map[string]bool{"phone": true})
}

// TestMaskITPg PostgreSQL方言运行时验证：大写列名/双引号别名/双引号表别名限定列
func TestMaskITPg(t *testing.T) {
	conn := maskPgConn(t)
	defer conn.Close()
	setupMaskPgTables(t, conn)
	plan := maskItPlan(t)

	// 1. 大写列名：正则(?i)命中PHONE；精确规则user_name小写化命中USER_NAME；ADDRESS不脱敏
	rows, cols := runMaskQuery(t, conn, plan, `SELECT "PHONE", "USER_NAME", "ADDRESS" FROM t_mask_user`)
	require.Len(t, rows, 1)
	assert.NotEqual(t, "13800001234", rows[0]["PHONE"], "大写列名PHONE应命中正则规则")
	assert.NotEqual(t, "zhangsan", rows[0]["USER_NAME"], "大写列名USER_NAME应命中精确规则(大小写不敏感)")
	assert.Equal(t, "北京市朝阳区", rows[0]["ADDRESS"], "ADDRESS不应脱敏")
	assertColMasked(t, cols, map[string]bool{"PHONE": true, "USER_NAME": true, "ADDRESS": false})
	checkMaskedJson(t, cols)

	// 2. 双引号别名：别名需去除引用符后命中映射
	rows, cols = runMaskQuery(t, conn, plan, `SELECT "PHONE" AS "p" FROM t_mask_user`)
	require.Len(t, rows, 1)
	assert.NotEqual(t, "13800001234", rows[0]["p"], "双引号别名应去除引用符后命中映射")
	assertColMasked(t, cols, map[string]bool{"p": true})

	// 3. 双引号表别名+限定列：尾段去引号后按列名兜底匹配
	rows, cols = runMaskQuery(t, conn, plan, `SELECT u."PHONE" FROM t_mask_user u`)
	require.Len(t, rows, 1)
	assert.NotEqual(t, "13800001234", rows[0]["PHONE"], "表别名限定列应正确脱敏")
	assertColMasked(t, cols, map[string]bool{"PHONE": true})

	// 4. 限定列+双引号别名：别名映射应覆盖限定列尾段去引号
	rows, cols = runMaskQuery(t, conn, plan, `SELECT u."PHONE" AS "p" FROM t_mask_user u`)
	require.Len(t, rows, 1)
	assert.NotEqual(t, "13800001234", rows[0]["p"], "限定列+双引号别名应正确映射脱敏")
	assertColMasked(t, cols, map[string]bool{"p": true})

	// 5. 豁免标签跨库生效：t_log整表豁免（标签未限定库名）
	rows, cols = runMaskQuery(t, conn, plan, `SELECT "PHONE" FROM t_log`)
	require.Len(t, rows, 1)
	assert.Equal(t, "13700003456", rows[0]["PHONE"], "PG下t_log命中整表豁免标签不应脱敏")
	assertColMasked(t, cols, map[string]bool{"PHONE": false})
}

// TestMaskITMysqlComplex MySQL复杂查询运行时验证：子查询/多表JOIN/聚合表达式/UNION/DISTINCT/星号展开等
func TestMaskITMysqlComplex(t *testing.T) {
	conn := maskMysqlConn(t)
	defer conn.Close()
	setupMaskMysqlTables(t, conn)
	plan := maskItPlan(t)

	// 1. IN子查询：WHERE中的子查询不影响select项精确归属
	rows, cols := runMaskQuery(t, conn, plan, "SELECT phone FROM t_user WHERE id IN (SELECT user_id FROM t_order)")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"], "IN子查询场景phone应命中hash标签")
	assertColMasked(t, cols, map[string]bool{"phone": true})

	// 2. EXISTS子查询：email走全局正则规则
	rows, cols = runMaskQuery(t, conn, plan, "SELECT email FROM t_user WHERE EXISTS (SELECT 1 FROM t_order o WHERE o.user_id = t_user.id)")
	require.Len(t, rows, 1)
	assert.NotEqual(t, "zhangsan@example.com", rows[0]["email"], "EXISTS子查询场景email应被脱敏")
	assertColMasked(t, cols, map[string]bool{"email": true})

	// 3. 标量子查询混合：phone命中hash标签；max_amt(amount非敏感列)不脱敏不误命中
	rows, cols = runMaskQuery(t, conn, plan, "SELECT phone, (SELECT MAX(amount) FROM t_order) AS max_amt FROM t_user WHERE id = 1")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"])
	assertColMasked(t, cols, map[string]bool{"phone": true, "max_amt": false})
	checkMaskedJson(t, cols)

	// 4. 派生表（子查询结果集）：递归解析内层SQL建立血缘，t.phone精确归属t_user命中hash标签
	rows, cols = runMaskQuery(t, conn, plan, "SELECT t.phone FROM (SELECT phone FROM t_user) t")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"], "派生表phone应通过血缘精确归属t_user命中hash标签")
	assertColMasked(t, cols, map[string]bool{"phone": true})

	// 5. LEFT JOIN混合敏感列与普通列
	rows, cols = runMaskQuery(t, conn, plan, "SELECT u.phone AS up, o.amount FROM t_user u LEFT JOIN t_order o ON o.user_id = u.id")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["up"])
	assertColMasked(t, cols, map[string]bool{"up": true, "amount": false})

	// 6. 三表JOIN：t_user绑定hash、t_order走全局规则、t_log整表豁免——三列三种结果，标签归属精确
	rows, cols = runMaskQuery(t, conn, plan,
		"SELECT u.phone AS up, o.phone AS op, l.phone AS lp FROM t_user u JOIN t_order o ON o.user_id = u.id LEFT JOIN t_log l ON l.id = o.user_id")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["up"], "up应精确归属t_user命中hash标签")
	assert.Equal(t, "139****2345", rows[0]["op"], "op应精确归属t_order走全局phone算法")
	assert.Equal(t, "13700003456", rows[0]["lp"], "lp应精确归属t_log命中整表豁免标签不脱敏")
	assertColMasked(t, cols, map[string]bool{"up": true, "op": true, "lp": false})
	checkMaskedJson(t, cols)

	// 7. 自JOIN同表：两个别名都归属t_user，均命中hash标签
	rows, cols = runMaskQuery(t, conn, plan, "SELECT a.phone AS ap, b.phone AS bp FROM t_user a JOIN t_user b ON a.id = b.id")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["ap"])
	assert.Equal(t, maskItHash("13800001234"), rows[0]["bp"])
	assertColMasked(t, cols, map[string]bool{"ap": true, "bp": true})

	// 8. 逗号隐式JOIN（parser归入From）：限定名归属精确
	rows, cols = runMaskQuery(t, conn, plan, "SELECT u.phone AS up, o.phone AS op FROM t_user u, t_order o WHERE o.user_id = u.id")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["up"])
	assert.Equal(t, "139****2345", rows[0]["op"], "逗号JOIN的o.phone应精确归属t_order")
	assertColMasked(t, cols, map[string]bool{"up": true, "op": true})

	// 9. 聚合敏感列（无限定名）：表达式token匹配到phone，表上下文未知走全局规则，仍精确脱敏
	rows, cols = runMaskQuery(t, conn, plan, "SELECT MAX(phone) AS mp FROM t_user")
	require.Len(t, rows, 1)
	assert.Equal(t, "138****1234", rows[0]["mp"], "MAX(phone)应通过表达式token匹配脱敏")
	assertColMasked(t, cols, map[string]bool{"mp": true})

	// 10. 聚合敏感列（限定名）：token+限定符确定t_user，命中hash标签
	rows, cols = runMaskQuery(t, conn, plan, "SELECT MAX(u.phone) AS mp FROM t_user u")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["mp"], "MAX(u.phone)应通过限定符归属t_user命中hash标签")
	assertColMasked(t, cols, map[string]bool{"mp": true})

	// 11. CONCAT表达式包含敏感列：整体结果脱敏；结果非纯手机号，phone算法按partial位置处理
	// （前3后4：13800001234- 共12字符 → 138 + 5星 + 后4字符"234-"）
	rows, cols = runMaskQuery(t, conn, plan, "SELECT CONCAT(phone, '-') AS cp FROM t_user")
	require.Len(t, rows, 1)
	assert.Equal(t, "138*****234-", rows[0]["cp"], "CONCAT(phone,..)应通过表达式token匹配脱敏")
	assertColMasked(t, cols, map[string]bool{"cp": true})

	// 12. 字符串字面量含列名不误命中：'phone'被剔除，address非敏感列不脱敏
	rows, cols = runMaskQuery(t, conn, plan, "SELECT CONCAT('phone', address) AS c2 FROM t_user WHERE id = 1")
	require.Len(t, rows, 1)
	assert.Equal(t, "phone北京市朝阳区", rows[0]["c2"], "字符串字面量中的列名不应触发脱敏")
	assertColMasked(t, cols, map[string]bool{"c2": false})

	// 13. GROUP BY聚合：分组列精确脱敏，计数列不脱敏
	rows, cols = runMaskQuery(t, conn, plan, "SELECT phone, COUNT(*) AS cnt FROM t_user GROUP BY phone")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"])
	assertColMasked(t, cols, map[string]bool{"phone": true, "cnt": false})

	// 14. UNION：列级脱敏一致，两个来源的phone均被hash
	rows, cols = runMaskQuery(t, conn, plan, "SELECT phone FROM t_user UNION SELECT phone FROM t_log")
	require.Len(t, rows, 2)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"], "UNION主查询phone应命中hash标签")
	assert.NotEqual(t, "13700003456", rows[1]["phone"], "UNION第二来源phone不应泄露原文")
	assertColMasked(t, cols, map[string]bool{"phone": true})

	// 15. DISTINCT+ORDER BY+LIMIT：不影响精确归属
	rows, cols = runMaskQuery(t, conn, plan, "SELECT DISTINCT phone FROM t_user ORDER BY phone DESC LIMIT 1")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"])
	assertColMasked(t, cols, map[string]bool{"phone": true})

	// 16. 星号展开：无法位置对应，退化为按结果列名兜底（单表无歧义，脱敏正确）
	rows, cols = runMaskQuery(t, conn, plan, "SELECT u.* FROM t_user u")
	require.Len(t, rows, 1)
	require.Len(t, cols, 4)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"], "星号展开的phone应按列名命中hash标签")
	assert.NotEqual(t, "zhangsan@example.com", rows[0]["email"])
	assert.Equal(t, "北京市朝阳区", rows[0]["address"])
	assertColMasked(t, cols, map[string]bool{"phone": true, "email": true, "id": false, "address": false})
}

// TestMaskITPgComplex PostgreSQL复杂查询运行时验证：大写列名表达式/JOIN豁免精确归属/星号展开
func TestMaskITPgComplex(t *testing.T) {
	conn := maskPgConn(t)
	defer conn.Close()
	setupMaskPgTables(t, conn)
	plan := maskItPlan(t)

	// 1. 大写列名函数表达式：token PHONE命中全局规则（函数名UPPER不误命中）
	rows, cols := runMaskQuery(t, conn, plan, `SELECT UPPER("PHONE") AS "UP" FROM t_mask_user m`)
	require.Len(t, rows, 1)
	assert.Equal(t, "138****1234", rows[0]["UP"], "UPPER(PHONE)应通过表达式token匹配脱敏")
	assertColMasked(t, cols, map[string]bool{"UP": true})

	// 2. JOIN跨表豁免精确归属：m的PHONE脱敏，t2的PHONE命中t_log整表豁免不脱敏
	rows, cols = runMaskQuery(t, conn, plan, `SELECT m."PHONE" AS p1, t2."PHONE" AS p2 FROM t_mask_user m JOIN t_log t2 ON t2."ID" = m."ID"`)
	require.Len(t, rows, 1)
	assert.Equal(t, "138****1234", rows[0]["p1"], "p1应精确归属t_mask_user走全局规则")
	assert.Equal(t, "13700003456", rows[0]["p2"], "p2应精确归属t_log命中豁免标签")
	assertColMasked(t, cols, map[string]bool{"p1": true, "p2": false})
	checkMaskedJson(t, cols)

	// 3. PG拼接表达式包含敏感列；结果非纯手机号，phone算法按partial位置处理
	// （前3后4：13800001234-x 共13字符 → 138 + 6星 + 后4字符"34-x"）
	rows, cols = runMaskQuery(t, conn, plan, `SELECT "PHONE" || '-x' AS cat FROM t_mask_user`)
	require.Len(t, rows, 1)
	assert.Equal(t, "138******34-x", rows[0]["cat"], "拼接表达式应通过token匹配脱敏")
	assertColMasked(t, cols, map[string]bool{"cat": true})

	// 4. 星号展开：按结果列名兜底（大写列名大小写不敏感命中）
	rows, cols = runMaskQuery(t, conn, plan, `SELECT m.* FROM t_mask_user m`)
	require.Len(t, rows, 1)
	require.Len(t, cols, 4)
	assert.NotEqual(t, "13800001234", rows[0]["PHONE"])
	assert.NotEqual(t, "zhangsan", rows[0]["USER_NAME"])
	assert.Equal(t, "北京市朝阳区", rows[0]["ADDRESS"])
	assertColMasked(t, cols, map[string]bool{"PHONE": true, "USER_NAME": true, "ID": false, "ADDRESS": false})

	// 5. 表限定+别名+ORDER BY+LIMIT
	rows, cols = runMaskQuery(t, conn, plan, `SELECT m."PHONE" AS "P" FROM t_mask_user m ORDER BY m."PHONE" LIMIT 1`)
	require.Len(t, rows, 1)
	assert.Equal(t, "138****1234", rows[0]["P"])
	assertColMasked(t, cols, map[string]bool{"P": true})
}

// TestMaskITMysqlComplex2 MySQL复杂查询补充：反引号列名函数/库名限定/CASE WHEN/UNION ALL混合/多表达式
func TestMaskITMysqlComplex2(t *testing.T) {
	conn := maskMysqlConn(t)
	defer conn.Close()
	setupMaskMysqlTables(t, conn)
	plan := maskItPlan(t)

	// 1. 反引号列名+函数表达式：MAX(`phone`) token匹配脱敏（反引号不阻碍标识符提取）
	rows, cols := runMaskQuery(t, conn, plan, "SELECT MAX(`phone`) AS mp FROM t_user")
	require.Len(t, rows, 1)
	assert.Equal(t, "138****1234", rows[0]["mp"], "MAX(`phone`)应通过token匹配走全局规则脱敏")
	assertColMasked(t, cols, map[string]bool{"mp": true})

	// 2. CASE WHEN表达式包含敏感列：结果分支均脱敏
	rows, cols = runMaskQuery(t, conn, plan, "SELECT CASE WHEN id = 1 THEN phone ELSE 'nophone' END AS c FROM t_user WHERE id = 1")
	require.Len(t, rows, 1)
	assert.Equal(t, "138****1234", rows[0]["c"], "CASE WHEN的phone分支应脱敏")
	assertColMasked(t, cols, map[string]bool{"c": true})

	// 3. UNION ALL混合敏感与豁免来源：主查询items决定映射，两行phone均不泄露原文；
	// DESC使t_user行（138>137）在前，断言第一行hash命中
	rows, cols = runMaskQuery(t, conn, plan, "SELECT phone FROM t_user UNION ALL SELECT phone FROM t_log ORDER BY phone DESC")
	require.Len(t, rows, 2)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"], "UNION ALL第一来源phone应命中hash标签")
	assert.NotEqual(t, "13700003456", rows[1]["phone"], "UNION ALL第二来源phone不应泄露原文")
	assertColMasked(t, cols, map[string]bool{"phone": true})

	// 4. 同一查询多个表达式混合：限定名表达式各归属各表，敏感与普通列互不干扰
	rows, cols = runMaskQuery(t, conn, plan, "SELECT MAX(t_user.phone) AS mp, MAX(o.amount) AS ma FROM t_user JOIN t_order o ON o.user_id = t_user.id")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["mp"], "MAX(t_user.phone)应通过限定名对归属t_user命中hash标签")
	assert.NotEmpty(t, rows[0]["ma"], "MAX(o.amount)不应脱敏（若命中即误判）")
	assertColMasked(t, cols, map[string]bool{"mp": true, "ma": false})

	// 5. 表达式与普通列混合：普通列精确归属hash标签，表达式走全局规则，互不影响；
	// UPPER(phone)结果仍为原手机号（数字不变），phone算法输出可预测
	rows, cols = runMaskQuery(t, conn, plan, "SELECT phone, UPPER(phone) AS pu FROM t_user WHERE id = 1")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"], "普通列phone应精确归属hash标签")
	assert.Equal(t, "138****1234", rows[0]["pu"], "UPPER(phone)应通过token匹配走全局规则脱敏")
	assertColMasked(t, cols, map[string]bool{"phone": true, "pu": true})

	// 6. 派生表内层别名：血缘p->phone->t_user，精确命中hash标签
	rows, cols = runMaskQuery(t, conn, plan, "SELECT t.p FROM (SELECT phone AS p FROM t_user) t")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["p"], "派生表内层别名列应通过血缘精确归属t_user")
	assertColMasked(t, cols, map[string]bool{"p": true})

	// 7. 派生表JOIN真实表：派生表列与真实表列各自精确归属
	rows, cols = runMaskQuery(t, conn, plan, "SELECT y.uphone, a.phone FROM t_user a JOIN (SELECT id, phone AS uphone FROM t_user) y ON y.id = a.id")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["uphone"], "派生表列uphone应通过血缘归属t_user命中hash标签")
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"], "真实表限定列phone应精确归属t_user")
	assertColMasked(t, cols, map[string]bool{"uphone": true, "phone": true})

	// 8. UNION派生表：多来源不误归属单表，走全局规则两行均不泄露
	rows, cols = runMaskQuery(t, conn, plan, "SELECT x.phone FROM (SELECT phone FROM t_user UNION SELECT phone FROM t_log) x")
	require.Len(t, rows, 2)
	assert.NotEqual(t, "13800001234", rows[0]["phone"], "UNION派生表第一行不应泄露原文")
	assert.NotEqual(t, "13700003456", rows[1]["phone"], "UNION派生表第二行不应泄露原文")
	assertColMasked(t, cols, map[string]bool{"phone": true})

	// 9. 内层SELECT*星号展开：输出列=来源表同名列，通配血缘精确归属t_user命中hash标签
	rows, cols = runMaskQuery(t, conn, plan, "SELECT d.phone FROM (SELECT * FROM t_user) d")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"], "内层星号展开应通过通配血缘精确归属t_user命中hash标签")
	assertColMasked(t, cols, map[string]bool{"phone": true})

	// 9.1 内层SELECT*+UNION ALL多来源展开（派生表要求输出列名唯一，JOIN展开重名列非法，
	// UNION ALL输出列名取第一分支故合法）：phone来源歧义，锁定仅全局规则，不误归属t_user的hash
	rows, cols = runMaskQuery(t, conn, plan, "SELECT d.phone FROM (SELECT * FROM t_user UNION ALL SELECT * FROM t_order) d")
	require.Len(t, rows, 2)
	assert.Equal(t, "138****1234", rows[0]["phone"], "多来源星号展开的同名列歧义应仅走全局规则")
	assert.NotEqual(t, maskItHash("13800001234"), rows[0]["phone"], "不应误归属t_user的hash标签")
	assertColMasked(t, cols, map[string]bool{"phone": true})

	// 9.2 内层SELECT*嵌套两层派生表：通配血缘跨层传播精确归属
	rows, cols = runMaskQuery(t, conn, plan, "SELECT x.phone FROM (SELECT * FROM (SELECT * FROM t_user) m) x")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"], "嵌套星号派生表应跨层传播通配血缘")
	assertColMasked(t, cols, map[string]bool{"phone": true})

	// 9.3 外层SELECT*：输出列名即内层输出列名，经通配来源表按名兜底精确归属
	rows, cols = runMaskQuery(t, conn, plan, "SELECT * FROM (SELECT * FROM t_user) d")
	require.Len(t, rows, 1)
	require.Len(t, cols, 4)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"], "外层星号展开的phone应经通配血缘归属t_user")
	assert.Equal(t, "北京市朝阳区", rows[0]["address"], "address不应脱敏")
	assertColMasked(t, cols, map[string]bool{"phone": true, "email": true, "id": false, "address": false})

	// 9.4 内层部分星号u.*：单来源通配血缘支撑限定表达式精确归属
	rows, cols = runMaskQuery(t, conn, plan, "SELECT MAX(d.phone) AS mp FROM (SELECT u.* FROM t_user u) d")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["mp"], "部分星号通配血缘应支撑限定表达式命中hash标签")
	assertColMasked(t, cols, map[string]bool{"mp": true})

	// 9.5 内层SELECT*+豁免表：通配血缘精确归属t_log命中整表豁免不脱敏
	// （若血缘未建立则退化为全局规则误脱敏，以此实证精确归属）
	rows, cols = runMaskQuery(t, conn, plan, "SELECT d.phone FROM (SELECT * FROM t_log) d")
	require.Len(t, rows, 1)
	assert.Equal(t, "13700003456", rows[0]["phone"], "星号派生表的t_log列应通过通配血缘命中整表豁免不脱敏")
	assertColMasked(t, cols, map[string]bool{"phone": false})

	// 10. 嵌套派生表：跨两层血缘传播精确归属
	rows, cols = runMaskQuery(t, conn, plan, "SELECT tt.phone FROM (SELECT t.phone FROM (SELECT phone FROM t_user) t) tt")
	require.Len(t, rows, 1)
	assert.Equal(t, maskItHash("13800001234"), rows[0]["phone"], "嵌套派生表应跨层传播血缘精确归属t_user")
	assertColMasked(t, cols, map[string]bool{"phone": true})

	// 11. 派生表非敏感列：MAX(amount)血缘到amount不脱敏不误命中
	rows, cols = runMaskQuery(t, conn, plan, "SELECT t.ma FROM (SELECT MAX(amount) AS ma FROM t_order) t")
	require.Len(t, rows, 1)
	assert.NotEmpty(t, rows[0]["ma"])
	assertColMasked(t, cols, map[string]bool{"ma": false})
}

// TestMaskITPgComplex2 PostgreSQL复杂查询补充：嵌套函数/大小写混合/子查询派生表
func TestMaskITPgComplex2(t *testing.T) {
	conn := maskPgConn(t)
	defer conn.Close()
	setupMaskPgTables(t, conn)
	plan := maskItPlan(t)

	// 1. 嵌套函数表达式：LOWER(UPPER("PHONE")) token匹配脱敏
	rows, cols := runMaskQuery(t, conn, plan, `SELECT LOWER(UPPER("PHONE")) AS lp FROM t_mask_user`)
	require.Len(t, rows, 1)
	assert.Equal(t, "138****1234", rows[0]["lp"], "嵌套函数的PHONE token应命中脱敏")
	assertColMasked(t, cols, map[string]bool{"lp": true})

	// 2. 派生表：按结果列名兜底命中全局规则
	rows, cols = runMaskQuery(t, conn, plan, `SELECT t."PHONE" FROM (SELECT "PHONE" FROM t_mask_user) t`)
	require.Len(t, rows, 1)
	assert.Equal(t, "138****1234", rows[0]["PHONE"], "派生表PHONE应按列名命中全局规则")
	assertColMasked(t, cols, map[string]bool{"PHONE": true})

	// 3. JOIN同列名无AS：位置对应模式下第二列改名后仍精确归属（t2为t_log豁免）
	rows, cols = runMaskQuery(t, conn, plan, `SELECT m."PHONE", t2."PHONE" FROM t_mask_user m JOIN t_log t2 ON t2."ID" = m."ID"`)
	require.Len(t, rows, 1)
	require.Len(t, cols, 2)
	require.NotEqual(t, cols[0].Key, cols[1].Key, "PG重名列应有唯一Key")
	assert.Equal(t, "138****1234", rows[0][cols[0].Key], "第一列PHONE应归属t_mask_user走全局规则")
	assert.Equal(t, "13700003456", rows[0][cols[1].Key], "第二列PHONE应精确归属t_log命中豁免标签")
	assertColMasked(t, cols, map[string]bool{cols[0].Key: true, cols[1].Key: false})
	checkMaskedJson(t, cols)

	// 4. 派生表大写列名：递归血缘精确归属，未命中表标签时全局规则兑底
	rows, cols = runMaskQuery(t, conn, plan, `SELECT t."PHONE" FROM (SELECT "PHONE" FROM t_mask_user) t`)
	require.Len(t, rows, 1)
	assert.Equal(t, "138****1234", rows[0]["PHONE"], "PG派生表PHONE应通过血缘归属并走全局规则脱敏")
	assertColMasked(t, cols, map[string]bool{"PHONE": true})

	// 5. 派生表JOIN真实表混合：派生表列与真实表列各自精确归属
	rows, cols = runMaskQuery(t, conn, plan, `SELECT y.up, m."PHONE" FROM t_mask_user m JOIN (SELECT "ID", "PHONE" AS up FROM t_mask_user) y ON y."ID" = m."ID"`)
	require.Len(t, rows, 1)
	assert.Equal(t, "138****1234", rows[0]["up"], "派生表列up应通过血缘精确归属")
	assert.Equal(t, "138****1234", rows[0]["PHONE"], "真实表限定列PHONE应精确归属")
	assertColMasked(t, cols, map[string]bool{"up": true, "PHONE": true})
	checkMaskedJson(t, cols)

	// 6. 派生表内层星号展开+豁免表：通配血缘精确归属t_log命中整表豁免不脱敏
	// （若血缘未建立则退化为全局规则误脱敏，以此实证精确归属）
	rows, cols = runMaskQuery(t, conn, plan, `SELECT t."PHONE" FROM (SELECT * FROM t_log) t`)
	require.Len(t, rows, 1)
	assert.Equal(t, "13700003456", rows[0]["PHONE"], "星号派生表的t_log列应通过通配血缘命中整表豁免不脱敏")
	assertColMasked(t, cols, map[string]bool{"PHONE": false})

	// 7. 外层SELECT*星号展开：输出列名即内层输出列名，经通配来源表按名兜底精确归属
	rows, cols = runMaskQuery(t, conn, plan, `SELECT * FROM (SELECT * FROM t_mask_user) t`)
	require.Len(t, rows, 1)
	require.Len(t, cols, 4)
	assert.Equal(t, "138****1234", rows[0]["PHONE"], "外层星号展开的PHONE应经通配血缘归属t_mask_user")
	assert.Equal(t, "北京市朝阳区", rows[0]["ADDRESS"], "ADDRESS不应脱敏")
	assertColMasked(t, cols, map[string]bool{"PHONE": true, "USER_NAME": true, "ID": false, "ADDRESS": false})
}
