package application

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"mayfly-go/internal/ai/agent"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
	"mayfly-go/internal/ai/infra/mcpclient"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

// McpPlugin MCP 服务器插件管理服务（对齐 tokhub MCP 插件，剪裁为仅 HTTP 传输）
type McpPlugin interface {
	base.App[*entity.McpServer]

	// ListServers MCP 服务器列表（id 排序）
	ListServers(ctx context.Context) ([]*entity.McpServer, error)

	// DiscoverTools 实时连接服务器发现工具（连接测试 + 前端回显）
	DiscoverTools(ctx context.Context, id uint64) ([]mcpclient.ToolInfo, error)
}

type mcpPluginAppImpl struct {
	base.AppImpl[*entity.McpServer, repository.McpServer]
}

var _ McpPlugin = (*mcpPluginAppImpl)(nil)

// Insert 覆写：新增后重置默认 Agent，新 MCP 服务器的工具运行期即时生效
func (a *mcpPluginAppImpl) Insert(ctx context.Context, e *entity.McpServer) error {
	if err := a.AppImpl.Insert(ctx, e); err != nil {
		return err
	}
	a.invalidateAgent()
	return nil
}

// UpdateByCond 覆写：更新（含启停/地址/请求头变更）后重置默认 Agent，运行期即时生效
func (a *mcpPluginAppImpl) UpdateByCond(ctx context.Context, values any, cond any) error {
	if err := a.AppImpl.UpdateByCond(ctx, values, cond); err != nil {
		return err
	}
	a.invalidateAgent()
	return nil
}

// DeleteById 覆写：删除后重置默认 Agent，对应工具运行期即时下线
func (a *mcpPluginAppImpl) DeleteById(ctx context.Context, id ...uint64) error {
	if err := a.AppImpl.DeleteById(ctx, id...); err != nil {
		return err
	}
	a.invalidateAgent()
	return nil
}

// invalidateAgent MCP 配置变更后重置默认 Agent 单例（fail-open，不阻断管理操作）：
// 下次对话懒重建 Agent 并重新聚合工具清单，配合 mcpext 连接缓存仅重连配置变更的服务器
func (a *mcpPluginAppImpl) invalidateAgent() {
	agent.ResetDefaultAgent()
	logx.Debug("[mcp_plugin] mcp server changed, default agent reset for next turn")
}

func (a *mcpPluginAppImpl) ListServers(ctx context.Context) ([]*entity.McpServer, error) {
	return a.ListByCond(model.NewCond().OrderByAsc("id"))
}

func (a *mcpPluginAppImpl) DiscoverTools(ctx context.Context, id uint64) ([]mcpclient.ToolInfo, error) {
	server, err := a.GetById(id)
	if err != nil {
		return nil, err
	}
	cli, err := mcpclient.Connect(ctx, server.Url, parseMcpHeaders(server.Headers), time.Duration(server.TimeoutSec)*time.Second)
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	return cli.ListTools(ctx)
}

// parseMcpHeaders 解析请求头 JSON（空或非法时返回 nil，不阻断连接）
func parseMcpHeaders(headers string) map[string]string {
	if strings.TrimSpace(headers) == "" {
		return nil
	}
	parsed := make(map[string]string)
	if err := json.Unmarshal([]byte(headers), &parsed); err != nil {
		return nil
	}
	if len(parsed) == 0 {
		return nil
	}
	return parsed
}
