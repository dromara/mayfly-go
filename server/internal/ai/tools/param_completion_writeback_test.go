package tools

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"mayfly-go/internal/ai/session"
	"mayfly-go/pkg/utils/collx"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"
)

// 参数补全的最终参数必须回写历史 tool_call 记录。
//
// 只把补全值合并进内存 args 时，工具能用真凭证执行成功，历史里那条 tool_call 的 arguments 却
// 仍是模型最初提交的空值 —— 审计与前端回显同实际执行不一致（真机核实：output 里是用户选的凭证，
// payload 的 authCertName 为空）。回写原先只挂在 ResumeParamCompletion 的 eino 分支上，而真机实际
// 走的是 TryApplyResumedParams 的缓存分支，它 apply 成功后直接 return，整段回写被跳过。
// 下面两条守卫分别锁住「回写确实发生」与「没有恢复分支绕过绑定」。

// probeCompletionArgs 参数补全测试用的探针工具参数（不用真实工具名与其 struct，避免污染注册表）
type probeCompletionArgs struct {
	AuthCertName string `json:"authCertName"`
	Command      string `json:"command"`
	Remark       string `json:"remark"`
}

// writeBackFakeStore 记录回写的会话存储替身；嵌入接口使未预期的方法调用直接暴露为 panic
type writeBackFakeStore struct {
	session.Store
	messages []*session.Message
	updated  []*session.Message
}

func (f *writeBackFakeStore) GetMessage(_ context.Context, query *session.MessageQuery) ([]*session.Message, error) {
	if query.MessageType != session.MsgTypeToolCall {
		return nil, nil
	}
	var msgs []*session.Message
	for _, msg := range f.messages {
		if query.ToolCallId == "" || query.ToolCallId == msg.ToolCalls[0].ID {
			msgs = append(msgs, msg)
		}
	}
	return msgs, nil
}

func (f *writeBackFakeStore) UpdateMessage(_ context.Context, msg *session.Message) error {
	f.updated = append(f.updated, msg)
	return nil
}

// newWriteBackStore 预置一条「模型最初提交的空凭证」tool_call 消息，并把替身装进全局存储
func newWriteBackStore(t *testing.T) *writeBackFakeStore {
	t.Helper()
	store := &writeBackFakeStore{messages: []*session.Message{{
		Id: 77,
		ToolCalls: []schema.ToolCall{{
			ID:       "call_probe",
			Type:     "function",
			Function: schema.FunctionCall{Name: "ProbeCompletionTool", Arguments: `{"authCertName":"","command":"date","remark":"r"}`},
		}},
	}}}
	restore := session.DefaultSessionStore
	session.DefaultSessionStore = store
	t.Cleanup(func() { session.DefaultSessionStore = restore })
	return store
}

func TestApplyResumedPayloadWritesBackFinalArgs(t *testing.T) {
	store := newWriteBackStore(t)
	args := &probeCompletionArgs{Command: "date", Remark: "r"}

	require.NoError(t, applyResumedPayload(context.Background(), "call_probe", collx.M{"authCertName": "tVUcRFKwOC"}, args))

	// 内存参数生效，且 payload 未提及的字段保留原值
	require.Equal(t, "tVUcRFKwOC", args.AuthCertName)
	require.Equal(t, "date", args.Command)

	// 回写必须落到存储层，且写的是补全后的最终参数（不是模型最初的空值）
	require.Len(t, store.updated, 1, "补全值必须回写历史 tool_call 记录")
	finalArgs := store.updated[0].ToolCalls[0].Function.Arguments
	require.Contains(t, finalArgs, "tVUcRFKwOC")
	require.NotContains(t, finalArgs, `"authCertName":""`)
}

func TestApplyResumedPayloadDoesNotWriteBackWhenMergeFails(t *testing.T) {
	store := newWriteBackStore(t)
	args := &probeCompletionArgs{Command: "date"}

	// 空 payload 提取不到任何参数，合并在先失败，就不该用半截状态去改写历史
	require.Error(t, applyResumedPayload(context.Background(), "call_probe", collx.M{}, args))
	require.Empty(t, store.updated)
}

func TestApplyResumedPayloadWithoutToolCallIdSkipsWriteBack(t *testing.T) {
	store := newWriteBackStore(t)
	args := &probeCompletionArgs{Command: "date"}

	// 无 toolCallId 时无法定位记录：只合并内存参数，不做无目标的库查询（也不该 panic）
	require.NoError(t, applyResumedPayload(context.Background(), "", collx.M{"authCertName": "c1"}, args))
	require.Equal(t, "c1", args.AuthCertName)
	require.Empty(t, store.updated)
}

// funcNamePattern 从函数声明行取出函数名（兼容方法声明的接收器前缀）
var funcNamePattern = regexp.MustCompile(`^func\s+(?:\([^)]*\)\s*)?([A-Za-z0-9_]+)`)

// TestResumedParamMergeIsAlwaysBoundToWriteBack 合并补全值的调用点必须全部经由 applyResumedPayload。
//
// 上面几个用例只能证明「绑定后的函数自身正确」；本缺陷的成因是某条恢复分支直接调用了
// applyPayloadToArgs 而绕开回写，将来新增恢复来源（新的 resume 通道）同样可能重犯，而那时
// 行为测试不会失败。因此扫源码断言这个合并函数除绑定处与自身定义外没有别的调用点。
func TestResumedParamMergeIsAlwaysBoundToWriteBack(t *testing.T) {
	const mergeCall = "applyPayloadToArgs("

	var offenders []string
	err := filepath.Walk(".", func(path string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return walkErr
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		currentFunc := ""
		for _, line := range strings.Split(string(src), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "func ") {
				if m := funcNamePattern.FindStringSubmatch(trimmed); m != nil {
					currentFunc = m[1]
				}
				continue // 声明行自身含括号，不参与调用点判定
			}
			if strings.Contains(trimmed, mergeCall) && currentFunc != "applyResumedPayload" {
				offenders = append(offenders, path+" :: "+currentFunc)
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.Empty(t, offenders, "补全值合并必须与历史回写绑定，绕开 applyResumedPayload 的调用点会让补全参数只存在于内存")
}
