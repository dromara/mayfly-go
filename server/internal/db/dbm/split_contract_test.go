package dbm

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

// SQL 切割跨语言契约（服务端侧）：与前端 utils/sqlParser.ts 读取同一份 testdata/split_cases.json。
//
// 为什么需要它：同一套切割语义在 Go 与 TS 各有一份状态机实现，前端切割结果直接决定
// 「SQL 编辑器把哪些文本作为独立语句发给服务端」。两侧只在各自测试里断言自己的输出时，
// 任何一侧单独改规则都不会让另一侧变红（历史上出现过三次漂移），故用同一份 fixture 钉死。
//
// 本测试刻意驱动生产方言装配得到的切割器（dialect.GetSQLSplitter），不在测试内复刻方言规则：
// 方言包漏接/接错 config 会在此暴露。
//
// 运行（无需任何数据库）：cd server && go test -run TestSplitContract ./internal/db/dbm/

// splitContractFile 前后端共享的切割用例契约文件（相对本包目录）
const splitContractFile = "sqlparser/testdata/split_cases.json"

// splitContractCase 契约用例：expected 按真实数据库词法书写，而非任一实现的当前输出
type splitContractCase struct {
	Name     string   `json:"name"`
	DbType   string   `json:"dbType"`
	Sql      string   `json:"sql"`
	Expected []string `json:"expected"`
}

func TestSplitContractWithFrontend(t *testing.T) {
	raw, err := os.ReadFile(splitContractFile)
	require.NoError(t, err, "切割契约文件缺失：%s", splitContractFile)

	var doc struct {
		Cases []splitContractCase `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	require.NotEmpty(t, doc.Cases, "切割契约不应为空")

	covered := make(map[dbi.DbType]int, len(doc.Cases))
	for _, tc := range doc.Cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			dt := dbi.DbType(tc.DbType)
			dialect := dbi.GetDialect(dt)
			require.NotNil(t, dialect, "契约中的方言 [%s] 未注册", tc.DbType)

			splitter := dialect.GetSQLSplitter()
			require.NotNil(t, splitter, "方言 [%s] 未装配切割器", tc.DbType)

			stmts := make([]string, 0, len(tc.Expected))
			err := splitter.SplitSQL(strings.NewReader(tc.Sql), func(stmt string) error {
				stmts = append(stmts, stmt)
				return nil
			})
			// 契约用例全部为良构脚本：报错说明切割器误判了某个区域（会直接导致漏执行/错执行）
			require.NoError(t, err, "[%s] 良构脚本不应报切割错误\nsql: %s", tc.DbType, tc.Sql)
			assert.Equal(t, tc.Expected, stmts, "[%s] 切割结果与契约不符\nsql: %s", tc.DbType, tc.Sql)

			covered[dt]++
		})
	}

	// 覆盖度：每个已注册方言（含别名）都至少有一条契约用例，
	// 新增方言只加方言包而不补切割语义时会在此变红（expectedDbTypes 见 dialect_registry_test.go）
	for _, dt := range expectedDbTypes {
		assert.Positive(t, covered[dt], "方言 [%s] 未被切割契约覆盖，请为其补充用例", dt)
	}
}
