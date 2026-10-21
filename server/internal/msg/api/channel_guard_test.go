package api

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 消息模块读侧的接线守卫。
//
// 存在理由（本轮真机前读代码抓到，非子代理上报）：渠道列表的「名称」搜索框把 `?name=xxx` 发得
// 出去，接口也照返 200，但 `GetMsgChannels` 当年直接 `&entity.MsgChannel{}` 空条件去查——参数
// 在绑定层就被丢弃。仓储侧后来补上了 name 谓词也救不了：谓词拿不到值。两层各缺一块时症状完全
// 一致（返回全量），而编译、类型检查、lint、界面全绿，所以把「搜索框维度 ↔ 参数读取」钉成守卫。

// readApiSource 读取本包源文件
func readApiSource(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(".", name))
	require.NoErrorf(t, err, "读取 %s 失败（改名必须同步本守卫）", name)
	return string(raw)
}

// funcBodyOf 截取以 prefix 开头的函数体（到下一个顶格 } 为止，够接线断言用）
func funcBodyOf(t *testing.T, source, prefix string) string {
	t.Helper()
	start := strings.Index(source, prefix)
	require.GreaterOrEqualf(t, start, 0, "函数 %s 不存在（改名/删除必须同步本守卫）", prefix)
	rest := source[start:]
	end := strings.Index(rest, "\n}")
	require.Greater(t, end, 0, "函数体闭合括号未顶格，本守卫的截取方式已失配")
	return rest[:end]
}

// TestChannelListReadsItsSearchDimensions 渠道列表必须把界面搜索框的两个维度都读进查询条件。
//
// 只钉仓储不钉这里等于没钉：本轮的缺陷恰恰发生在「参数根本没进 condition」这一层
func TestChannelListReadsItsSearchDimensions(t *testing.T) {
	body := funcBodyOf(t, readApiSource(t, "msg_channel.go"), "func (m *MsgChannel) GetMsgChannels(")

	for _, dim := range []string{"name", "code"} {
		require.Containsf(t, body, `rc.Query("`+dim+`")`,
			"渠道列表的「%s」搜索维度必须在 api 层读进查询条件，否则参数被静默丢弃、接口返回全量", dim)
	}
	require.Contains(t, body, "condition", "读到的参数必须真的交给查询条件，而不是收下就丢")

	// 收窄动作必须发生在查询之前（先查后赋等于没筛）
	require.Less(t, strings.Index(body, `rc.Query("name")`), strings.Index(body, "GetPageList(condition"),
		"条件必须在真正查询前填好")
}

// TestChannelCredentialStaysBehindSavePermission bot 渠道的加签密钥不得随列表外溢，
// 且取回它的唯一出口必须受保存权限把守。
//
// 存在理由（真机抓下来）：列表接口把整个 extra 原样回吐，而 secret 是钉钉 HMAC 签名密钥
// （ding_bot 拿它算 &sign=），拿到就能以该渠道身份往外部群伪造消息；而列表按菜单基权限即可达。
// 修法是「列表脱敏 + 受 msg:channel:save 把守的详情接口」，两个前提必须同时成立：
// 只加详情不改列表，泄漏面一点没收；只改列表不加详情，编辑一保存就把密钥抹空
func TestChannelCredentialStaysBehindSavePermission(t *testing.T) {
	source := readApiSource(t, "msg_channel.go")

	list := funcBodyOf(t, source, "func (m *MsgChannel) GetMsgChannels(")
	require.Contains(t, list, "msgx.ExtraKeySecret", "列表必须脱敏密钥")
	require.Contains(t, list, `SetExtraValue(msgx.ExtraKeySecret, "")`, "脱敏必须是清空该键，不是只换个字段名")

	require.Contains(t, source, `req.NewGet("/:id", m.GetMsgChannel).RequiredPermissionCode("msg:channel:save")`,
		"详情端点必须存在且受保存权限把守（菜单基权限就能读密钥等于没修）")
	detail := funcBodyOf(t, source, "func (m *MsgChannel) GetMsgChannel(")
	require.NotContains(t, detail, "ExtraKeySecret", "详情不得脱敏：它就是编辑回填的唯一合法来源")
}

// TestSecretExtraKeyHasSingleSource 「secret」只能由 msgx.ExtraKeySecret 定义一次。
//
// 脱敏侧写一个键名、发送侧读另一个键名时不会报错：只会「该脱的没脱」或「加签静默不算」
func TestSecretExtraKeyHasSingleSource(t *testing.T) {
	root := filepath.Join("..", "..") // internal/msg
	// 常量定义处的路径识别用后缀而不是等值：WalkDir 交的是相对 root 的路径（带 ../），
	// 拼个“msgx/msgx.go”去 == 比永不相等，会把定义处自己也当成违规项
	const constFileSuffix = "msgx" + string(filepath.Separator) + "msgx.go"
	var hits []string
	require.NoError(t, filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		require.NoError(t, err)
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		if strings.HasSuffix(path, constFileSuffix) {
			require.Contains(t, string(raw), `ExtraKeySecret = "secret"`, "常量定义处本身必须仍是唯一定义")
			return nil
		}
		if strings.Contains(string(raw), `"secret"`) {
			hits = append(hits, path)
		}
		return nil
	}))
	require.Emptyf(t, hits, "除 msgx.ExtraKeySecret 定义处外不得再出现「secret」字面量（两侧键名分叉不报错）：%v", hits)
}
