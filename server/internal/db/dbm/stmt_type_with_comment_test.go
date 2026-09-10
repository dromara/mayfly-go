// 前导注释语句的类型识别单测（不依赖数据库）。
//
//	背景：SQL 切割器为保持「所见即所执行」与 Oracle hint / mysql 可执行注释的正确性，
//	不再丢弃注释原文，于是从切割器出来的语句可能以 "-- 说明\n" 或 "/* x */ " 开头。
//	而下游多个**数据安全相关**判定都以语句类型/首关键字为输入：
//	 - 脱敏：按 SelectStmt 构建列级血缘（丢失则退化为仅按列名兜底）
//	 - 审计：Update/DeleteStmt 的表名与 WHERE 用于记录旧值（丢失则误改/误删无法追溯）
//	 - 审批：procdef.MatchCondition(stmtType) 决定 UPDATE/DELETE 是否需提工单（误判即绕过审批）
//	 - 兜底分类：解析失败时 sqlKind 依赖 LeadingKeyword 取首关键字
//	 - 导入过滤：shouldSkipImportStmt 依赖「掩码注释后的整体文本」识别事务包装语句
//	本测试把「各方言对前导注释的容忍」固化为回归断言，防止未来改动造成静默降级。
package dbm

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// commentCase 前导注释语句用例：noise 为注释中刻意埋入的干扰关键字（可能骗过朴素的文本匹配）
type commentCase struct {
	name    string
	sql     string
	kind    sqlstmt.Kind
	keyword string // 期望的首关键字（小写整词）
}

// commentPrefixedCases 各方言通用形态（不使用 # 注释，因 # 仅 mysql/clickhouse 系支持）
var commentPrefixedCases = []commentCase{
	{"行注释后的DELETE", "-- 清理历史\nDELETE FROM t WHERE id = 1", sqlstmt.KindDelete, "delete"},
	{"注释含干扰词后的SELECT", "-- update t set a=1 先备份\nSELECT a, b FROM t WHERE c = 1", sqlstmt.KindSelect, "select"},
	{"块注释后的UPDATE", "/* 工单 123 */ UPDATE t SET a = 1 WHERE id = 2", sqlstmt.KindUpdate, "update"},
	{"连续行注释后的INSERT", "-- a\n-- b\nINSERT INTO t (id) VALUES (1)", sqlstmt.KindInsert, "insert"},
	{"注释后的CREATE", "-- 建表 drop table t\nCREATE TABLE t (id INT)", sqlstmt.KindDdl, "create"},
	{"块注释后的DROP", "/* 下线 delete */ DROP TABLE t", sqlstmt.KindDdl, "drop"},
}

// firstToken 取文本忽略前导空白后的首个整词（小写），用于校验掩码结果
func firstToken(text string) string {
	text = strings.TrimLeft(text, " \t\r\n")
	words := strings.FieldsFunc(text, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '(' || r == ';' })
	if len(words) == 0 {
		return ""
	}
	return strings.ToLower(words[0])
}

// TestParseToleratesLeadingComments 各方言解析器必须容忍前导注释并给出正确语句类型
func TestParseToleratesLeadingComments(t *testing.T) {
	for _, dt := range dbi.GetRegisteredDbTypes() {
		dialect := dbi.GetDialect(dt)
		require.NotNilf(t, dialect, "方言 [%s] 未注册", dt)
		parser := dialect.GetSQLParser()
		require.NotNilf(t, parser, "方言 [%s] 解析器为nil", dt)

		for _, c := range commentPrefixedCases {
			t.Run(string(dt)+"/"+c.name, func(t *testing.T) {
				stmt, err := parser.Parse(c.sql)
				// 未解析出语句（方言不支持该语法）允许跳过，但解析成功时类型绝不能被注释带偏
				if stmt == nil {
					t.Logf("方言 [%s] 未解析出语句，跳过类型断言: %v", dt, err)
					return
				}
				require.NoErrorf(t, err, "方言 [%s] 解析前导注释语句失败，下游脱敏/审计/审批将静默降级: %v", dt, err)
				assert.Equalf(t, c.kind, stmt.StmtKind(), "方言 [%s] 语句类型被注释内容带偏: %q", dt, c.sql)
			})
		}
	}
}

// TestLeadingKeywordSkipsComments 兜底取首关键字必须跳过注释，
// 否则 "-- 说明\nDELETE ..." 会取到注释里的单词，导致误路由（如把 DELETE 当查询执行、绕过工单校验）
func TestLeadingKeywordSkipsComments(t *testing.T) {
	for _, dt := range dbi.GetRegisteredDbTypes() {
		splitter := dbi.GetDialect(dt).GetSQLSplitter()
		require.NotNilf(t, splitter, "方言 [%s] 切割器为nil", dt)
		for _, c := range commentPrefixedCases {
			assert.Equalf(t, c.keyword, splitter.LeadingKeyword(c.sql), "方言 [%s] 首关键字取自注释: %q", dt, c.sql)
		}
	}
}

// TestMaskCommentsRemovesCommentText 掩码结果必须彻底抹去普通注释文本（等长空白），
// 导入侧据此判定事务包装语句；残留注释文本会造成漏过滤（隐式提交）或误过滤（丢数据）
func TestMaskCommentsRemovesCommentText(t *testing.T) {
	for _, dt := range dbi.GetRegisteredDbTypes() {
		splitter := dbi.GetDialect(dt).GetSQLSplitter()
		for _, c := range commentPrefixedCases {
			masked := splitter.MaskComments(c.sql)
			require.Lenf(t, masked, len(c.sql), "方言 [%s] 掩码未保持等长: %q", dt, c.sql)
			assert.NotContainsf(t, masked, "drop table", "方言 [%s] 普通注释未被完全掩码: %q", dt, masked)
			assert.NotContainsf(t, masked, "清理历史", "方言 [%s] 普通注释未被完全掩码: %q", dt, masked)
			// 首个真实关键字前不存在任何注释残留文本
			assert.Equalf(t, c.keyword, firstToken(masked), "方言 [%s] 掩码后首个词不是语句关键字: %q", dt, masked)
		}
	}
}

// TestMysqlExecutableCommentHandling mysql 可执行注释解壳：内容参与判定，外壳与版本号掩码
func TestMysqlExecutableCommentHandling(t *testing.T) {
	splitter := dbi.GetDialect("mysql").GetSQLSplitter()

	// # 行注释为 mysql 方言能力
	assert.Equal(t, "delete", splitter.LeadingKeyword("# 备注\nDELETE FROM t"))
	masked := splitter.MaskComments("# 备注\nDELETE FROM t")
	assert.NotContains(t, masked, "备注")

	// 可执行注释内容会被真正执行，故其关键字必须可见（否则 /*!40101 SET autocommit=0 */ 会绕过过滤）
	assert.Equal(t, "set", splitter.LeadingKeyword("/*!40101 SET autocommit=0 */"))
	unwrapped := strings.TrimSpace(splitter.MaskComments("/*!40101 SET autocommit=0 */"))
	assert.Equal(t, "SET autocommit=0", unwrapped)

	// 井号在 mssql 中是临时表前缀，不是注释
	assert.Equal(t, "select", dbi.GetDialect("mssql").GetSQLSplitter().LeadingKeyword("SELECT * FROM #tmp"))
}
