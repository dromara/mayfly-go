package machinetool

import (
	"context"
	"encoding/json"
	"testing"

	"mayfly-go/internal/ai/application/resource"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/model"

	"github.com/stretchr/testify/require"
)

// 一台机器挂多张授权凭证时，补全卡片必须把每张凭证各自列为可选项。
//
// 这条链路此前只有单凭证样本的真机观测：多凭证是「不猜」规则的另一半——若候选生成时把
// 机器 code 当凭证名（历史缺陷），或把同机器的多张凭证塌成一条，用户在卡片上无论选哪条
// 都会拿到错值，而「不猜」的执行侧兜底此时故意不生效（它只处理唯一凭证）。
// 用注入的资源视图直接钉住候选生成，比依赖模型当轮恰好传对参数的真机路径更稳。

// fakeResourceApp 只提供 List 的资源视图替身；其余方法被调用即暴露误用（nil 接口 panic）
type fakeResourceApp struct {
	resource.App
	list func(ctx context.Context, accountId uint64, query *resource.Query) ([]*resource.Resource, error)
}

func (f *fakeResourceApp) List(ctx context.Context, accountId uint64, query *resource.Query) ([]*resource.Resource, error) {
	return f.list(ctx, accountId, query)
}

// withMachineCerts 装一台 aliyun 机器、挂给定凭证清单（每张展开为一条资源，与 machine_provider 口径一致）
func withMachineCerts(t *testing.T, certs ...string) context.Context {
	t.Helper()
	resources := make([]*resource.Resource, 0, len(certs))
	for _, cert := range certs {
		resources = append(resources, &resource.Resource{
			Type:        resource.TypeMachine,
			Id:          "1",
			Code:        "aliyun-code",
			Name:        "aliyun",
			Description: "59.110.115.25:22",
			Detail:      map[string]string{resource.ExtraKeyIp: "59.110.115.25"},
			Extra:       map[string]any{resource.ExtraKeyAuthCertName: cert},
		})
	}
	restore := resource.GetApp()
	resource.SetDefault(&fakeResourceApp{list: func(context.Context, uint64, *resource.Query) ([]*resource.Resource, error) {
		return resources, nil
	}})
	t.Cleanup(func() { resource.SetDefault(restore) })
	return contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: 1, Username: "admin"})
}

func TestListMachineOptionsOneOptionPerCert(t *testing.T) {
	ctx := withMachineCerts(t, "certA", "certB")

	options := listMachineOptions(ctx)
	require.Len(t, options, 2, "两张凭证必须各成一个候选，不能塌成一条")

	// 每个候选的 value 必须携带它自己的真实凭证名（不是机器 code），label 也必须带凭证名以便区分
	seen := make([]string, 0, len(options))
	for _, opt := range options {
		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(opt.Value), &payload))
		cert, _ := payload["authCertName"].(string)
		require.NotEmpty(t, cert, "候选 payload 必须给出真实凭证名")
		require.NotEqual(t, "aliyun-code", cert, "机器 code 不是授权凭证名，选中之后的执行必然失败")
		require.Contains(t, opt.Label, cert, "同机器多凭证时 label 必须带凭证名，否则用户无从区分")
		seen = append(seen, cert)
	}
	require.ElementsMatch(t, []string{"certA", "certB"}, seen)
}

func TestListMachineOptionsSkipsResourceWithoutCertName(t *testing.T) {
	// 凭证名解析不出来（标签路径未含凭证）的资源不给候选：选中它也只能提交一个空值
	ctx := withMachineCerts(t, "certA", "")

	options := listMachineOptions(ctx)
	require.Len(t, options, 1, "无凭证名的资源必须被跳过，不能进候选")
	require.Contains(t, options[0].Value, "certA")
}

func TestListMachineOptionsWithoutLoginAccountIsFailClosed(t *testing.T) {
	restore := resource.GetApp()
	t.Cleanup(func() { resource.SetDefault(restore) })
	resource.SetDefault(&fakeResourceApp{list: func(context.Context, uint64, *resource.Query) ([]*resource.Resource, error) {
		t.Fatal("无登录态不得查询资源清单")
		return nil, nil
	}})

	// 无登录态时不给任何候选（fail-closed）：否则等于绕过账号级标签过滤把全量机器暴露给选择器
	require.Empty(t, listMachineOptions(context.Background()))
}
