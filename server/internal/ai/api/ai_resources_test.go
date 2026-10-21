package api

import (
	"testing"

	"mayfly-go/internal/ai/api/vo"
	"mayfly-go/internal/ai/application/resource"

	"github.com/stretchr/testify/require"
)

// TestToChatResourceVOs 授权过滤后的资源投影必须去重且字段来自 Detail。
//
// 背景：`ChatResources` 曾直连 machineApp/dbApp 的 ListByCond，把全部启用机器（含 ip/port）
// 与全部数据库返回给任一登录用户；改走 resource 门面后，投影层需要自己处理它的两个形状事实：
//  1. 机器按「机器 × 授权凭证」展开，同一台机器挂多个凭证会出现多条 → 菜单必须按 (type,id) 去重，
//     否则前端 `@` 列表里同一台机器重复出现；
//  2. ip/port 在 Detail 里（字符串），不在 Resource 顶层 → 端口要解析，缺失时给 0 而不是崩溃。
func TestToChatResourceVOs(t *testing.T) {
	// 同一台机器两个凭证（provider 的真实展开形态）+ 一个数据库
	list := []*resource.Resource{
		{
			Type: resource.TypeMachine, Id: "7", Code: "m-7", Name: "web-01",
			Description: "10.0.0.7:22",
			Detail:      map[string]string{resource.ExtraKeyIp: "10.0.0.7", resource.ExtraKeyPort: "22"},
		},
		{
			Type: resource.TypeMachine, Id: "7", Code: "m-7", Name: "web-01",
			Description: "10.0.0.7:22",
			Detail:      map[string]string{resource.ExtraKeyIp: "10.0.0.7", resource.ExtraKeyPort: "22"},
		},
		{
			Type: resource.TypeDb, Id: "3", Code: "order_db", Name: "订单库",
			Description: "order_db",
		},
	}

	got := toChatResourceVOs(list)
	require.Len(t, got, 2, "同一台机器按凭证展开成多条时必须去重成一条")

	machine := got[0]
	require.Equal(t, &vo.ChatResourceVO{
		Id: "7", ResourceType: resource.TypeMachine, Name: "web-01", Code: "m-7",
		Ip: "10.0.0.7", Port: 22, Description: "10.0.0.7:22",
	}, machine)

	// 数据库没有 Detail：ip/port 保持零值，不得因缺键 panic
	require.Equal(t, resource.TypeDb, got[1].ResourceType)
	require.Equal(t, "order_db", got[1].Code)
	require.Zero(t, got[1].Port)
	require.Empty(t, got[1].Ip)

	// 空输入返回空切片而不是 nil：ResData 直接序列化，nil 会变成 null 需要前端额外兜底
	require.NotNil(t, toChatResourceVOs(nil))
	require.Empty(t, toChatResourceVOs(nil))
}
