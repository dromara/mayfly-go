// Package mcpext MCP 工具扩展（对齐 tokhub PluginToolsExtension）
//
// 实现 ToolContributor 通道：遍历启用的 MCP 服务器（仅 HTTP 传输），
// 逐个连接发现工具并包装为 eino InvokableTool 贡献给 Agent。
// 连接经进程级缓存管理器复用：配置指纹未变时不重连，单个服务器
// 连接失败 fail-open（Warnf 日志，不影响装配）。
// 管理端增删改/启停变更经 application 层重置默认 Agent 实现运行期即时生效。
package mcpext

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/infra/mcpclient"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
)

// ToolNamePrefix MCP 工具名前缀（贡献给 LLM 的工具名约定：mcp_{code}_{name}，
// tool search 按该前缀识别可延迟加载的 MCP 工具）
const ToolNamePrefix = "mcp_"

// ServerConfig MCP 连接配置（由 t_ai_plugin_instance.config 内联配置解析产出）
type ServerConfig struct {
	// Id 实例 ID（连接缓存键）
	Id uint64
	// Code 实例唯一标识（工具名前缀）
	Code string
	// Url MCP 服务器地址（Streamable HTTP / SSE）
	Url string
	// Headers 请求头 JSON
	Headers string
	// TimeoutSec 请求超时秒数
	TimeoutSec int
}

// serverLoader MCP 服务器加载器（application.Init 注入，避免 ext → application 依赖）
var serverLoader func(ctx context.Context) ([]*ServerConfig, error)

// SetServerLoader 注入 MCP 服务器加载器
//
// 时序契约（与 RegisterHostInstaller 一致）：须在宿主装配完成前调用
// （首次 AssembleDefault / GetDefaultAgent 之前）—— Tools 贡献在装配期
// 读取 loader，装配后才注入的 loader 需重置默认 Agent（ResetDefaultAgent）
// 方可生效
func SetServerLoader(loader func(ctx context.Context) ([]*ServerConfig, error)) {
	serverLoader = loader
}

// McpToolsExtension MCP 工具扩展
type McpToolsExtension struct{}

// NewExtension 创建 MCP 工具扩展
func NewExtension() *McpToolsExtension {
	return &McpToolsExtension{}
}

var _ contributor.ToolContributor = (*McpToolsExtension)(nil)

func (e *McpToolsExtension) Id() string { return "mcp_tools" }

func (e *McpToolsExtension) Tools(ctx context.Context, tc *contributor.ToolContributionContext) ([]tool.BaseTool, error) {
	if serverLoader == nil {
		return nil, nil
	}
	servers, err := serverLoader(ctx)
	if err != nil {
		// fail-open：MCP 工具加载失败不阻断 Agent 装配
		logx.WarnfContext(ctx, "[mcp_tools] load mcp servers failed: %v", err)
		return nil, nil
	}
	return connMgr.toolsOf(ctx, servers), nil
}

// serverConn 单个 MCP 服务器的长连接缓存（连接保持至不再被引用或配置变更）
type serverConn struct {
	cli         *mcpclient.Client
	serverCode  string
	tools       []tool.BaseTool
	fingerprint string // 配置指纹（code|url|headers|timeout），变更即重连
}

// connManager MCP 连接缓存管理器（进程级单例，并发安全）：
// 按服务器 Id 缓存长连接，配置指纹未变时复用（Agent 重建不重连），
// 配置变更重连、停用/删除关闭移除，实现管理端变更的运行期生效
type connManager struct {
	mu    sync.Mutex
	conns map[uint64]*serverConn
}

var connMgr = &connManager{conns: make(map[uint64]*serverConn)}

// toolsOf 同步缓存与当前启用配置并返回全部可用工具（fail-open：单个连接失败跳过）
func (m *connManager) toolsOf(ctx context.Context, servers []*ServerConfig) []tool.BaseTool {
	m.mu.Lock()
	defer m.mu.Unlock()

	tools := make([]tool.BaseTool, 0)
	validIds := make(map[uint64]struct{}, len(servers))
	for _, server := range servers {
		validIds[server.Id] = struct{}{}
		fp := fmt.Sprintf("%s|%s|%s|%d", server.Code, server.Url, server.Headers, server.TimeoutSec)
		// 指纹未变：复用缓存连接，不重连
		if conn, ok := m.conns[server.Id]; ok && conn.fingerprint == fp {
			tools = append(tools, conn.tools...)
			continue
		}
		// 新增或配置变更：关闭旧连接重建
		if old, ok := m.conns[server.Id]; ok {
			old.cli.Close()
			delete(m.conns, server.Id)
		}
		conn, err := connectServerTools(ctx, server, fp)
		if err != nil {
			logx.WarnfContext(ctx, "[mcp_tools] connect mcp server failed, skip (code=%s, url=%s): %v",
				server.Code, server.Url, err)
			continue
		}
		m.conns[server.Id] = conn
		logx.InfofContext(ctx, "[mcp_tools] mcp server '%s' contributed %d tools", server.Code, len(conn.tools))
		tools = append(tools, conn.tools...)
	}
	// 停用/删除：关闭并移除失效连接
	for id, conn := range m.conns {
		if _, ok := validIds[id]; !ok {
			conn.cli.Close()
			delete(m.conns, id)
			logx.InfofContext(ctx, "[mcp_tools] mcp server '%s' connection closed (disabled or removed)", conn.serverCode)
		}
	}
	return tools
}

// connectServerTools 连接单个 MCP 服务器并包装其工具
func connectServerTools(ctx context.Context, server *ServerConfig, fingerprint string) (*serverConn, error) {
	timeout := time.Duration(server.TimeoutSec) * time.Second
	cli, err := mcpclient.Connect(ctx, server.Url, parseHeaders(server.Headers), timeout)
	if err != nil {
		return nil, err
	}
	toolInfos, err := cli.ListTools(ctx)
	if err != nil {
		cli.Close()
		return nil, err
	}

	result := make([]tool.BaseTool, 0, len(toolInfos))
	for i := range toolInfos {
		wrapped, err := newMcpInvokableTool(cli, server.Code, toolInfos[i])
		if err != nil {
			logx.WarnfContext(ctx, "[mcp_tools] wrap mcp tool failed, skip (server=%s, tool=%s): %v",
				server.Code, toolInfos[i].Name, err)
			continue
		}
		result = append(result, wrapped)
	}
	return &serverConn{cli: cli, serverCode: server.Code, tools: result, fingerprint: fingerprint}, nil
}

var _ tool.InvokableTool = (*mcpInvokableTool)(nil)

// mcpInvokableTool MCP 工具的 eino 适配器（工具名加服务器前缀防跨服务器冲突）
type mcpInvokableTool struct {
	cli        *mcpclient.Client
	serverCode string
	info       mcpclient.ToolInfo
	// name 贡献给 LLM 的工具名（mcp_{code}_{name}）
	name string
}

func newMcpInvokableTool(cli *mcpclient.Client, serverCode string, info mcpclient.ToolInfo) (*mcpInvokableTool, error) {
	// InputSchema 预解析校验（装配期发现非法 schema 即跳过）
	var js jsonschema.Schema
	if len(info.InputSchema) > 0 {
		if err := json.Unmarshal(info.InputSchema, &js); err != nil {
			return nil, err
		}
	}
	return &mcpInvokableTool{
		cli:        cli,
		serverCode: serverCode,
		info:       info,
		name:       ToolNamePrefix + serverCode + "_" + info.Name,
	}, nil
}

func (t *mcpInvokableTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	info := &schema.ToolInfo{
		Name: t.name,
		Desc: t.info.Description,
	}
	if len(t.info.InputSchema) > 0 {
		var js jsonschema.Schema
		if err := json.Unmarshal(t.info.InputSchema, &js); err != nil {
			return nil, err
		}
		info.ParamsOneOf = schema.NewParamsOneOfByJSONSchema(&js)
	}
	return info, nil
}

func (t *mcpInvokableTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return t.cli.CallTool(ctx, t.info.Name, json.RawMessage(argumentsInJSON))
}

// parseHeaders 请求头 JSON 解析（非法时返回 nil，不阻断连接）
func parseHeaders(headers string) map[string]string {
	if headers == "" {
		return nil
	}
	parsed := make(map[string]string)
	if err := json.Unmarshal([]byte(headers), &parsed); err != nil {
		return nil
	}
	return parsed
}

// Install 注册扩展到 Builder
func Install(b *contributor.Builder) {
	b.RegisterTool(NewExtension())
}
