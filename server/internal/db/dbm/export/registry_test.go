package export

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDescriptorsRegistered 内置三种导出格式必须完成注册且描述符信息完备。
// Descriptors() 是上层（API/前端）动态格式清单的唯一数据源：新增格式在此追加断言即可，
// 漏注册时 Get 返回 nil 会被 DumpDbScript 显式报错，而非静默回退 SQL
func TestDescriptorsRegistered(t *testing.T) {
	ds := Descriptors()
	require.Len(t, ds, 3, "内置格式应为 sql/csv/json")

	// 按 Format 字典序输出（csv < json < sql，排序稳定性守卫）
	formats := []string{ds[0].Format, ds[1].Format, ds[2].Format}
	assert.Equal(t, []string{"csv", "json", "sql"}, formats)

	for _, d := range ds {
		assert.NotEmpty(t, d.Name, "格式[%s]缺少展示名", d.Format)
		assert.NotEmpty(t, d.ContentType, "格式[%s]缺少MIME类型", d.Format)
		assert.True(t, strings.HasPrefix(d.FileExtension, "."), "格式[%s]扩展名应带点前缀", d.Format)
		require.NotNil(t, Get(d.Format), "描述符与Get不一致：格式[%s]", d.Format)
	}
}

// TestGetStatefulIsolation 有状态消费者（工厂注册）每次必须返回新实例，
// 无状态消费者返回共享实例——两种注册方式语义不可互换
func TestGetStatefulIsolation(t *testing.T) {
	// csv/json 经 RegisterFactory 注册（持有写入缓冲/行状态）：必须隔离实例
	assert.NotSame(t, Get("csv"), Get("csv"), "csv 工厂注册必须每次新建实例")
	assert.NotSame(t, Get("json"), Get("json"), "json 工厂注册必须每次新建实例")
	// sql 经 Register 注册（无状态单例）：共享实例
	assert.Same(t, Get("sql"), Get("sql"), "sql 单例注册应共享实例")
}
