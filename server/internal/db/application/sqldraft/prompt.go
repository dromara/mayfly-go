package sqldraft

// 提示词与模型调用。
//
// 模板放在本领域自己的 prompts/ 目录（不塞进 ai 模块的提示词库）：SQL 生成的
// 系统约束与业务语义紧密相关，归属 db 域；这里只复用 pkg 级的模板解析能力。
//
// 所有提示词（含选表选择题的系统提示）都必须是 prompts/ 下的文件，不留 Go 常量：
// 调提示词是本领域最频繁的改动，散在代码里会让「改一句话」变成「改代码 + 重新编译」，
// 也让 prompts/ 目录不再是提示词的唯一真源。

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"time"

	"mayfly-go/internal/ai/pkg/textgen"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/stringx"
)

//go:embed prompts/*.md
var promptFS embed.FS

// 模型调用阶段标识：进 textgen.Request.Purpose，也进日志。
//
// 一次草稿生成最多有三种调用（选表选择题 / 生成 / 自纠），不区分阶段时日志里
// 只能看到「sql_draft 花了 12s」，无法判断慢在召回还是生成——而这两者的优化手段完全不同。
const (
	purposeDraft  = "sql_draft"
	purposeChoose = "sql_draft_choose"
	purposeRepair = "sql_draft_repair"
	// purposeCandidate 候选补采（同一任务再要一份答案，用于未通过校验时的重排选优）
	purposeCandidate = "sql_draft_candidate"
)

// candidateTemperature 候选补采的采样温度。
//
// 首轮固定 0（同一输入要同一答案，可复现）；补采的目的恰恰是**换一份答案**，
// 温度 0 会重采出同一个结果、白烧一次调用。0.7 是「够发散又不离谱」的经验值
const candidateTemperature = float32(0.7)

// chooseSystemPrompt 选表选择题的系统提示（无变量，读一次缓存住）
var chooseSystemPromptValue = loadPrompt("choose_system.md")

func chooseSystemPrompt() string { return chooseSystemPromptValue }

// loadPrompt 读取一份无变量模板（读失败返回空串：提示词缺失会由调用侧的模板渲染报错暴露）
func loadPrompt(name string) string {
	raw, err := promptFS.ReadFile("prompts/" + name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

// renderTemplate 渲染 prompts/ 下的模板
func renderTemplate(name string, values any) (string, error) {
	raw, err := promptFS.ReadFile("prompts/" + name)
	if err != nil {
		return "", err
	}
	return stringx.TemplateParse(string(raw), values)
}

// modelCall 一次模型调用的完整参数。
//
// 做成结构体而不是四个位置参数：调用点已经有三处、参数还在增加（阶段标识、输出上限覆盖），
// 位置参数在调用点上无法自解释，且新增一个参数要改所有调用点。
type modelCall struct {
	// purpose 调用阶段（见 purpose* 常量）
	purpose string
	system  string
	user    string
	// timeout 本次调用的超时（0 取 callTimeout）
	timeout time.Duration
	// maxTokens 输出上限覆盖（0 取意图配置或全局默认）
	maxTokens int
	// temperature 采样温度覆盖（nil 取 0）：只有候选补采需要非零，见 candidateTemperature
	temperature *float32
}

// generator 生成器：一个请求内的模型调用计数与用量累计。
//
// 把「调用次数 / 用量」收在一个对象里，是为了让自纠轮次与成本在结果里可核对：
// 一次请求到底调了几次模型必须能从返回值算出来，而不是只能查日志。
type generator struct {
	port  textgen.Port
	usage *textgen.Usage
	calls int
	// maxTokens 意图级输出上限覆盖（0 用默认）：行内补全这类短产物要压小，避免模型长篇大论
	maxTokens int
}

func newGenerator(port textgen.Port) *generator {
	return &generator{port: port, usage: &textgen.Usage{}}
}

// complete 一次生成调用（非流式；温度固定 0，草稿不需要随机性）
func (g *generator) complete(ctx context.Context, call modelCall) (string, error) {
	temperature := float32(0)
	if call.temperature != nil {
		temperature = *call.temperature
	}
	timeout := call.timeout
	if timeout <= 0 {
		timeout = callTimeout
	}
	purpose := call.purpose
	if purpose == "" {
		purpose = purposeDraft
	}
	outputLimit := maxOutputTokens
	switch {
	case call.maxTokens > 0:
		outputLimit = call.maxTokens
	case g.maxTokens > 0:
		outputLimit = g.maxTokens
	}
	started := time.Now()
	text, usage, err := g.port.Complete(ctx, &textgen.Request{
		Purpose:     purpose,
		System:      call.system,
		User:        call.user,
		MaxTokens:   outputLimit,
		Temperature: &temperature,
		Timeout:     timeout,
		// SQL 草稿/选表/行内补全都是短结构化输出：走快速通道（关闭思考模式），实测能砍掉
		// 相当一部分首字节延迟，且不改变产物形态
		PreferFast: true,
	})
	// 耗时与用量入日志：SQL 草稿的体感瓶颈几乎全在模型，运维需要能看到"这一次慢在哪"
	logx.InfofContext(ctx, "[sqldraft] model call purpose=%s elapsed=%s prompt_chars=%d tokens=%v err=%v",
		purpose, time.Since(started).Truncate(time.Millisecond), len([]rune(call.user)), usage, err != nil)
	g.calls++
	if usage != nil {
		g.usage.PromptTokens += usage.PromptTokens
		g.usage.CompletionTokens += usage.CompletionTokens
		g.usage.TotalTokens += usage.TotalTokens
	}
	if err != nil {
		return "", fmt.Errorf("complete: %w", err)
	}
	return text, nil
}
