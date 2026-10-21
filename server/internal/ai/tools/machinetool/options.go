package machinetool

import (
	"context"
	"encoding/json"
	"strings"

	"mayfly-go/internal/ai/application/resource"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/pkg/logx"
)

// queryMachineOptions 查询当前用户有权限操作的机器列表并格式化为可选项
// （数据源为统一资源查询服务，经「机器+授权凭证」标签做账号级权限过滤）
func queryMachineOptions(ctx context.Context) []tools.CompletionOption {
	return tools.SafeOptionsFn(ctx, "queryMachineOptions", func() []tools.CompletionOption {
		return listMachineOptions(ctx)
	})
}

// resolveUniqueAuthCert 模型把机器名 / ip:port / 机器 code 当成 authCertName 传入时，
// 按「该机器唯一可操作凭证」确定性纠正。
//
// 为什么必须做：实测模型首轮不会去查 ListResources，而是直接拿机器名（如 "aliyun"）当凭证名传，
// 固定产生一次 `ac not found` 空转才会自纠。一台机器只有一个可操作凭证时根本不存在歧义，
// 没有理由让调用失败。多凭证时不猜（返回 false），保持原错并交给参数补全由人选
func resolveUniqueAuthCert(ctx context.Context, input string) (string, bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", false
	}
	// 取数统一走 resource.ListVisible：无登录态时它返回空清单，等价于「拿不到可见面就不纠正」
	// （fail-closed：绝不绕过标签过滤去猜全量机器）
	resources, err := resource.ListVisible(ctx, resource.TypeMachine)
	if err != nil {
		logx.WarnfContext(ctx, "[resolveUniqueAuthCert] list machines failed: %v", err)
		return "", false
	}
	return pickUniqueCert(resources, trimmed)
}

// pickUniqueCert 在账号可见的机器资源里按「机器名 / ip:port / 裸 ip / 机器 code」定位机器，
// 仅当它只有一个可操作凭证时返回该凭证。
//
// 与取数分开：资源清单由调用方按登录态取得，这里只做判定，便于把「多凭证不猜」这条规则
// 当作纯逻辑断言（它才是真正需要钉住的安全判据）
func pickUniqueCert(resources []*resource.Resource, input string) (string, bool) {
	// 先排除「input 本身已经是某个可见凭证名」：合法凭证名绝不能被当成机器标识去改写。
	//
	// 少了这一步会发生跨机器误改：机器 B 的名字（或 ip/code）恰好等于机器 A 的凭证名时，
	// 模型传 A 的合法凭证名会匹配到 B，B 若只有唯一凭证就被改写回去——登记期这条路径没有
	// 「建连失败才兜底」的前置，改错的值直接写进提案，审批人批准的就是被改错的目标机器
	for _, r := range resources {
		if r == nil {
			continue
		}
		if cert, _ := r.Extra[resource.ExtraKeyAuthCertName].(string); cert != "" && cert == input {
			return "", false
		}
	}

	// 同一机器可能挂多张凭证（展开后为多条资源），收集去重后仅在唯一时才纠正
	certs := make(map[string]struct{}, 2)
	for _, r := range resources {
		if r == nil {
			continue
		}
		if r.Name != input && r.Code != input && r.Description != input &&
			r.Detail[resource.ExtraKeyIp] != input {
			continue
		}
		cert, _ := r.Extra[resource.ExtraKeyAuthCertName].(string)
		if cert == "" || cert == input {
			// 传入值本身已是凭证名时不自指纠正：否则会走一条无意义的替换路径
			continue
		}
		certs[cert] = struct{}{}
	}
	if len(certs) != 1 {
		return "", false
	}
	for cert := range certs {
		return cert, true
	}
	return "", false
}

func listMachineOptions(ctx context.Context) []tools.CompletionOption {
	// 取数统一走 resource.ListVisible（与 resolveUniqueAuthCert 同一入口，不再各写一份三步样板）
	resources, err := resource.ListVisible(ctx, resource.TypeMachine)
	if err != nil {
		logx.WarnfContext(ctx, "[queryMachineOptions] failed to list machines: %v", err)
		return nil
	}

	var options []tools.CompletionOption
	for _, r := range resources {
		// 凭证名取资源 Extra（machine_provider 由标签路径解析出的真实凭证 name）。
		//
		// 这里以前写 r.Code：机器 code 与凭证 name 是两个不同的值，用户在补全卡片里选好
		// 之后照样 ac not found——选项本身就是错的。单凭证时这个脏值被执行侧兜底解析掩盖了，
		// 但多凭证机器兜底不猜，选中即失败；且脏值会留在卡片 payload 里被其它消费方拿到
		authCertName, _ := r.Extra[resource.ExtraKeyAuthCertName].(string)
		if authCertName == "" {
			continue
		}
		payload := map[string]any{
			"authCertName": authCertName,
		}
		valueBytes, err := json.Marshal(payload)
		if err != nil {
			continue
		}
		label := r.Name
		if r.Description != "" {
			label += " (" + r.Description + ")"
		}
		// 一台机器挂多张凭证时展开成多条资源，仅靠机器名与地址无法区分；带上凭证名
		label += " · " + authCertName
		options = append(options, tools.CompletionOption{
			Label: label,
			Value: string(valueBytes),
		})
	}

	return options
}
