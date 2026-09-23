package scratchclean

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunAlwaysCleans(t *testing.T) {
	for _, testCode := range []int{0, 1, 2} {
		for _, cleanupFails := range []bool{false, true} {
			called := false
			code := run(func() int { return testCode }, func(ctx context.Context) error {
				called = true
				_, bounded := ctx.Deadline()
				assert.True(t, bounded)
				assert.NoError(t, ctx.Err())
				if cleanupFails {
					return errors.New("模拟清理失败")
				}
				return nil
			})
			assert.True(t, called)
			if cleanupFails {
				assert.NotZero(t, code)
			} else {
				assert.Equal(t, testCode, code)
			}
		}
	}
}

func TestRunPanicCleans(t *testing.T) {
	called := false
	assert.Panics(t, func() {
		run(func() int { panic("模拟测试异常") }, func(context.Context) error {
			called = true
			return nil
		})
	})
	assert.True(t, called)
}

func TestCleanupRejectsUnownedNames(t *testing.T) {
	for _, name := range []string{"mayfly_dbm_it", "postgres", "mayfly_it_" + strings.Repeat("g", 24), "mayfly_it_short"} {
		t.Run(name, func(t *testing.T) {
			scope := Scope{dbs: map[string]database{"test": {name: name}}}
			require.ErrorContains(t, scope.Cleanup(context.Background()), "拒绝清理非托管数据库")
			assert.Len(t, scope.dbs, 1, "失败资源必须保留，不能报告已清理")
		})
	}
	require.NoError(t, (&Scope{}).Cleanup(context.Background()))
}
