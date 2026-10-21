package machinetool

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/ai/application/resource"
)

// machineResource 构造一条机器资源：真实凭证名放在 Extra（machine_provider 由标签路径解析得到）
func machineResource(id, code, name, ipPort, ip, certName string) *resource.Resource {
	return &resource.Resource{
		Type: resource.TypeMachine, Id: id, Code: code, Name: name, Description: ipPort,
		Detail: map[string]string{resource.ExtraKeyIp: ip, resource.ExtraKeyAuthCertName: certName},
		Extra:  map[string]any{resource.ExtraKeyAuthCertName: certName},
	}
}

// aliyunLike 真实形态：机器 code=UMaq3B2KHkKg 与凭证 name=tVUcRFKwOC 是两个不同的值
// （旧代码把前者当后者用，导致 MachineCommandExec 必然 ac not found）
func aliyunLike(cert string) *resource.Resource {
	return machineResource("1", "UMaq3B2KHkKg", "aliyun", "59.110.115.25:22", "59.110.115.25", cert)
}

// TestPickUniqueCertCorrectsMachineNameToSoleCert 唯一凭证时把机器标识纠正为真实凭证名。
//
// 钉住端到端实测到的固定空转：模型首轮不查 ListResources，直接拿机器名 aliyun 当
// authCertName 传，修复前 11 次调用 10 次失败，修复后 1 次调用 0 失败
func TestPickUniqueCertCorrectsMachineNameToSoleCert(t *testing.T) {
	resources := []*resource.Resource{aliyunLike("tVUcRFKwOC")}

	cert, ok := pickUniqueCert(resources, "aliyun")
	require.True(t, ok, "机器名 + 唯一凭证应能确定性纠正")
	require.Equal(t, "tVUcRFKwOC", cert)
}

// TestPickUniqueCertMatchesAnyMachineIdentifier 模型代指机器的四种形态都要认得：
// 机器名、机器 code、ip:port、裸 ip。少认一种就仍会空转一次。
func TestPickUniqueCertMatchesAnyMachineIdentifier(t *testing.T) {
	resources := []*resource.Resource{aliyunLike("tVUcRFKwOC")}

	for _, input := range []string{"aliyun", "UMaq3B2KHkKg", "59.110.115.25:22", "59.110.115.25"} {
		cert, ok := pickUniqueCert(resources, input)
		require.True(t, ok, "%q 应能反查到唯一凭证", input)
		require.Equal(t, "tVUcRFKwOC", cert, "%q 纠正结果应一致", input)
	}
}

// TestPickUniqueCertRefusesAmbiguousMachine 多凭证机器绝不猜。
//
// 替模型选凭证等于凭空授权：审批与审计里记的凭证可能不是实际连接的凭证，
// 这里必须保持原错，让参数补全卡片交给操作者选
func TestPickUniqueCertRefusesAmbiguousMachine(t *testing.T) {
	resources := []*resource.Resource{aliyunLike("certA"), aliyunLike("certB")}

	_, ok := pickUniqueCert(resources, "aliyun")
	require.False(t, ok, "两张凭证时不能猜其中一张")
}

// TestPickUniqueCertNoSelfReferenceOnValidCert 传入值已是凭证名时不自指纠正。
//
// 否则「凭证名恰好等于某台机器名」这种数据会走一条无意义的替换路径
func TestPickUniqueCertNoSelfReferenceOnValidCert(t *testing.T) {
	resources := []*resource.Resource{machineResource("1", "code1", "aliyun", "1.1.1.1:22", "1.1.1.1", "tVUcRFKwOC")}

	_, ok := pickUniqueCert(resources, "tVUcRFKwOC")
	require.False(t, ok, "已经是凭证名就不该再纠正，该值由调用方直接用于建连")
}

// TestPickUniqueCertUnknownInput 查不到机器、空清单、脏资源都返回 false 且不 panic。
func TestPickUniqueCertUnknownInput(t *testing.T) {
	resources := []*resource.Resource{aliyunLike("cert1")}

	_, ok := pickUniqueCert(resources, "不存在的机器")
	require.False(t, ok, "匹配不到机器不得凭空给出凭证")

	_, ok = pickUniqueCert(nil, "aliyun")
	require.False(t, ok, "空清单不能纠正出任何凭证")

	// Extra 里凭证名为空（资产未挂凭证）时不能被当成候选返回
	_, ok = pickUniqueCert([]*resource.Resource{aliyunLike("")}, "aliyun")
	require.False(t, ok, "凭证名为空不构成纠正")

	// nil 元素不能 panic（资源清单由外部服务构造，容错在这里兜住）
	_, ok = pickUniqueCert([]*resource.Resource{nil, aliyunLike("cert1")}, "aliyun")
	require.True(t, ok, "脏数据只跳过该条，不影响其余判定")
}

// TestCompletionOptionsCarryRealCertName 参数补全选项的 payload 必须是真实凭证名，不能是机器 code。
//
// 用静态守卫而不是行为用例：取数靠 resource.GetApp()（无可替换入口），装配成本高；
// 而这一处正是“改了旁边函数、却漏改选项构造点”的真实案例——当时只核对了
// pickUniqueCert 已落盘就以为整块改完，端到端复验才发现选项里仍是脏值（单凭证下被
// 执行侧兜底掩盖，多凭证下直接让用户选中失败）。这类漏改只能靠守卫钉住
func TestCompletionOptionsCarryRealCertName(t *testing.T) {
	src, err := os.ReadFile("options.go")
	require.NoError(t, err, "读取 options.go 失败")
	text := string(src)

	require.NotContains(t, text, `"authCertName": r.Code`,
		"补全选项不得把机器 code 当凭证名（两者不等，用户选中即 ac not found）")
	require.Contains(t, text, `authCertName, _ := r.Extra[resource.ExtraKeyAuthCertName].(string)`,
		"补全选项必须从资源 Extra 取标签路径解析出的真实凭证名")
}

// TestResolveUniqueAuthCertFailClosedWithoutLogin 无登录态时必须 fail-closed。
//
// 资源可见面由标签做账号级过滤，绕过它就等于放开全量机器
func TestResolveUniqueAuthCertFailClosedWithoutLogin(t *testing.T) {
	_, ok := resolveUniqueAuthCert(context.Background(), "aliyun")
	require.False(t, ok, "无登录态不得纠正出任何凭证")

	// 空输入不触发纠正
	_, ok = resolveUniqueAuthCert(context.Background(), "   ")
	require.False(t, ok, "空输入直接返回，不查资源服务")
}

// TestPickUniqueCertDoesNotRewriteValidCertName 输入已是合法凭证名时，绝不能再当机器标识去改写。
//
// 真机审查抓出的场景：机器 B 的名字恰好等于机器 A 的凭证名（人名起的字段极易重名）。
// 若按机器名匹配就会命中 B、并把它唯一的凭证塞回去——登记期这条路径没有「建连失败才兜底」
// 的前置，被改错的值会直接写进提案，审批人批准的就是与实际不同的目标机器
func TestPickUniqueCertDoesNotRewriteValidCertName(t *testing.T) {
	resources := []*resource.Resource{
		machineResource("1", "codeA", "机器A", "10.0.0.1:22", "10.0.0.1", "prod-cert"),
		machineResource("2", "codeB", "prod-cert", "10.0.0.2:22", "10.0.0.2", "b-cert"),
	}

	_, ok := pickUniqueCert(resources, "prod-cert")
	require.False(t, ok, "输入本身是可见凭证名时不得改写（否则等于跨机器换凭证）")

	// 同一个输入按「机器名」去匹配另一台机器仍是需要的行为：这里用 B 的另一个标识验证未过度收紧
	cert, ok := pickUniqueCert(resources, "codeB")
	require.True(t, ok, "按机器 code 定位仍应生效")
	require.Equal(t, "b-cert", cert)
}
