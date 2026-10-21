package dbtool

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsConnectionLevelErrorCoversDriverWrappedFailures 连接层判定必须覆盖真机遇到的形态。
func TestIsConnectionLevelErrorCoversDriverWrappedFailures(t *testing.T) {
	for _, msg := range []string{
		"db connection failed: invalid connection", // MySQL 首次建连把 1044 吞成 ErrBadConn 后的形态
		"db connection failed: dial tcp 127.0.0.1:3306: connect: connection refused",
		"db connection failed: dial tcp: lookup host on 127.0.0.1:53: no such host",
		"db connection failed: context deadline exceeded: i/o timeout",
	} {
		require.True(t, isConnectionLevelError(errors.New(msg)), "%q 应识别为连接层失败", msg)
	}

	// 库名/库权限类错误归另一条判据（那边会给候选库名清单），两者必须互斥
	require.False(t, isConnectionLevelError(errors.New("Error 1049 (42000): Unknown database '本地库'")))
	require.False(t, isConnectionLevelError(errors.New("Error 1044 (42000): Access denied for user 'x'@'%' to database 'mayfly-go'")))
	require.False(t, isConnectionLevelError(nil))
}

// TestConnLevelFailureHintGivesActionableNextStep 连接层失败必须给出一条可执行动作。
//
// 钉住真机测到的浪费：上一轮同一 args 首次失败什么提示都没有，模型只能换实例再撞一次
func TestConnLevelFailureHintGivesActionableNextStep(t *testing.T) {
	connErr := errors.New("db connection failed: invalid connection")

	hint := connLevelFailureHint("mayfly-go", connErr)
	require.NotNil(t, hint, "带库名的连接层失败必须补提示")
	require.Contains(t, hint.Error(), "db connection failed: invalid connection", "原始错误不能被替换掉")
	require.Contains(t, hint.Error(), "建立连接阶段", "要说清这不是库名问题")
	require.Contains(t, hint.Error(), "留空", "必须给出可执行的下一步（留空重调即可拿到库清单）")

	// 非连接层失败、或本来就没传库名（那条路径已有守卫）时不得重复提示
	require.Nil(t, connLevelFailureHint("", connErr), "空 dbName 由上面的守卫负责，不叠第二句提示")
	require.Nil(t, connLevelFailureHint("mayfly-go", errors.New("Error 1049 (42000): Unknown database 'x'")),
		"库名类错误由候选清单分支负责")
	require.Nil(t, connLevelFailureHint("mayfly-go", nil))
}

// TestTwoErrorBranchesAreDisjoint 两条提示分支必须互斥，避免同一次失败叠出两句矛盾提示。
func TestTwoErrorBranchesAreDisjoint(t *testing.T) {
	msgs := []string{
		"db connection failed: Error 1049 (42000): Unknown database '本地库'",
		"db connection failed: Error 1044 (42000): Access denied for user 'x'@'%' to database 'y'",
		`db connection failed: pq: database "test" does not exist`,
		"db connection failed: 数据库文件不存在",
		"db connection failed: invalid connection",
		"db connection failed: dial tcp 127.0.0.1:3306: connect: connection refused",
	}
	for _, msg := range msgs {
		err := errors.New(msg)
		both := isUnknownDatabaseError(err) && isConnectionLevelError(err)
		require.False(t, both, "%q 不应同时命中两条判据（否则会叠出矛盾提示）", msg)
		// 每条已知的失败形态至少要有一条判据接住，不能出现「两种都不认」的无提示真空
		if strings.Contains(msg, "invalid connection") || strings.Contains(msg, "refused") {
			require.True(t, isConnectionLevelError(err))
		} else {
			require.True(t, isUnknownDatabaseError(err))
		}
	}
}
