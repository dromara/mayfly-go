package dbm

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// SQL 读写/语句类型跨语言契约（服务端侧）：与前端 core/__tests__/classifyContract.test.ts 读取同一份
// sqlparser/testdata/classify_cases.json。
//
// 为什么需要它：语句「读还是写」的判定在 Go（sqlparser.Classify，权威、决定执行路由与触发策略）与
// TS（isQueryStatement，决定结果 tab 与本地确认）各有一份实现，历史上已出现 SELECT INTO 这类
// 「后端判读、前端判写」的漂移。两侧只在各自测试里断言自己的输出时不会互相变红，故用同一份 fixture 钉死。
//
// 本测试驱动生产方言装配得到的解析器（dialect.GetSQLParser/GetSQLSplitter），不在测试内复刻规则：
// 方言漏接/接错 config 会在此暴露。
//
// 运行（无需任何数据库）：cd server && go test -run TestClassifyContract ./internal/db/dbm/

// classifyContractFile 前后端共享的分类用例契约文件（相对本包目录）
const classifyContractFile = "sqlparser/testdata/classify_cases.json"

// classifyContractCase 契约用例：stmtType 为后端 Classify 期望，isQuery 为前端判定期望（前端侧断言）。
type classifyContractCase struct {
	Name     string `json:"name"`
	DbType   string `json:"dbType"`
	SQL      string `json:"sql"`
	StmtType string `json:"stmtType"`
	IsQuery  bool   `json:"isQuery"`
}

func TestClassifyContractWithFrontend(t *testing.T) {
	raw, err := os.ReadFile(classifyContractFile)
	require.NoError(t, err, "分类契约文件缺失：%s", classifyContractFile)

	var doc struct {
		Cases []classifyContractCase `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	require.NotEmpty(t, doc.Cases, "分类契约不应为空")

	covered := make(map[dbi.DbType]int, len(doc.Cases))
	for _, tc := range doc.Cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			dt := dbi.DbType(tc.DbType)
			dialect := dbi.GetDialect(dt)
			require.NotNilf(t, dialect, "契约中的方言 [%s] 未注册", tc.DbType)

			parser, splitter := dialect.GetSQLParser(), dialect.GetSQLSplitter()
			require.NotNilf(t, parser, "方言 [%s] 未装配解析器", tc.DbType)
			require.NotNilf(t, splitter, "方言 [%s] 未装配切割器", tc.DbType)

			stmt, parseErr := parser.Parse(tc.SQL)
			got := sqlparser.Classify(parser, splitter, tc.SQL, stmt, parseErr)
			assert.Equalf(t, sqlstmt.StmtType(tc.StmtType), got, "[%s] Classify 与契约不符\nsql: %s", tc.DbType, tc.SQL)

			covered[dt]++
		})
	}

	// 覆盖度：每个已注册方言（含别名）都至少有一条契约用例，新增方言不补分类用例时在此变红。
	for _, dt := range expectedDbTypes {
		assert.Positivef(t, covered[dt], "方言 [%s] 未被分类契约覆盖，请为其补充用例", dt)
	}
}
