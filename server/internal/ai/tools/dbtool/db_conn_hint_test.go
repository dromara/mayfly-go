package dbtool

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsUnknownDatabaseErrorCoversRealForms 判据必须覆盖真机实测到的四种失败表述。
//
// 其中 1044 那条是真机复验抓出的漏判：把实例显示名当 schema 传入（dbName="mayfly-go"）时
// MySQL 报的是 Access denied 而不是 Unknown database，只匹配 1049 会让这种错完全拿不到任何提示，
// 结果模型连撞两次同样的错、最后放弃作答
func TestIsUnknownDatabaseErrorCoversRealForms(t *testing.T) {
	for _, msg := range []string{
		"db connection failed: Error 1049 (42000): Unknown database '本地库'",
		"db connection failed: Error 1044 (42000): Access denied for user 'dev_mayfly_go'@'%' to database 'mayfly-go'",
		`db connection failed: pq: database "test" does not exist`,
		"db connection failed: 数据库文件不存在",
	} {
		require.True(t, isUnknownDatabaseError(errors.New(msg)), "%q 应识别为库名/库权限类错误", msg)
	}

	require.False(t, isUnknownDatabaseError(errors.New("db connection failed: invalid connection")))
	require.False(t, isUnknownDatabaseError(errors.New("dial tcp 127.0.0.1:5432: connect: connection refused")))
	require.False(t, isUnknownDatabaseError(nil))
}

// TestDbUnknownNameFailureSemanticsByCandidateCount 按候选数量断言提示**语义**
// （取代原先扫源码数文案出现次数的静态守卫）。
//
// 关键约束来自真机：候选为空时曾输出「真实库名为 []，只能取其中之一」——对着空清单提要求，
// 模型无从自纠，而正确的「无法枚举 + 可能无权限」兜底永远命中不到
func TestDbUnknownNameFailureSemanticsByCandidateCount(t *testing.T) {
	base := errors.New("db connection failed: Error 1049 (42000): Unknown database '本地库'")

	withCandidates := dbUnknownNameFailure(base, []string{"mayfly_go", "sys"}).Error()
	require.Contains(t, withCandidates, "mayfly_go, sys", "候选非空时必须把可选库名逐个列出来")
	require.NotContains(t, withCandidates, "无法枚举", "有候选时不能说拿不到清单")
	require.Contains(t, withCandidates, "Unknown database '本地库'", "原始错误必须保留，提示是追加不是替换")

	noCandidates := dbUnknownNameFailure(base, nil).Error()
	require.NotContains(t, noCandidates, "只能取其中之一", "空清单不得要求模型从中选一个")
	require.NotContains(t, noCandidates, "真实库名为 []", "不得把空数组当成候选清单输出")
	require.Contains(t, noCandidates, "当前无法枚举该实例的库清单")
	require.Contains(t, noCandidates, "没有权限", "还要交代另一种可能：凭证对该库无权限")
	require.Contains(t, noCandidates, "Unknown database '本地库'", "原始错误必须保留")
}

// TestDbUnknownNameFailureTreatsPlaceholderAsNoCandidate 只含占位空值的候选等于没有候选。
//
// 与 nonEmptyDbNames 成对：兜底项若混进候选清单，len>0 会让这里走错分支（真机踩过）
func TestDbUnknownNameFailureTreatsPlaceholderAsNoCandidate(t *testing.T) {
	base := errors.New("db connection failed: 数据库文件不存在")
	outcome := dbUnknownNameFailure(base, nonEmptyDbNames([]dbOptionItem{{dbName: "", label: "tokhub"}})).Error()
	require.NotContains(t, outcome, "只能取其中之一", "过滤后为空就必须走兜底分支")
}
