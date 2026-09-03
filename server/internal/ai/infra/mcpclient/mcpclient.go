// Package mcpclient MCP HTTP 客户端封装（对齐 tokhub agent-mcp，剪裁为仅
// Streamable HTTP 传输）：供 application 层（工具发现/连接测试）与
// agent/ext/mcp（运行期工具调用）共用，统一连接握手与结果转换。
package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// Client MCP 服务器客户端（一次连接，多次调用）
type Client struct {
	mcp *client.Client
}

// Connect 建立连接并完成 MCP 握手（initialize）
//   - url MCP 服务器地址（Streamable HTTP / SSE 自动协商）
//   - headers 自定义请求头（如 Authorization）
//   - timeout 请求超时（<=0 时默认 30s）
func Connect(ctx context.Context, url string, headers map[string]string, timeout time.Duration) (*Client, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	opts := []transport.StreamableHTTPCOption{
		transport.WithHTTPTimeout(timeout),
	}
	if len(headers) > 0 {
		opts = append(opts, transport.WithHTTPHeaders(headers))
	}
	c, err := client.NewStreamableHttpClient(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("create mcp client: %w", err)
	}
	if err := c.Start(ctx); err != nil {
		return nil, fmt.Errorf("start mcp transport: %w", err)
	}
	initReq := mcp.InitializeRequest{}
	initReq.Params.ClientInfo = mcp.Implementation{Name: "mayfly-go", Version: "1.0.0"}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		return nil, fmt.Errorf("initialize mcp session: %w", err)
	}
	return &Client{mcp: c}, nil
}

// ToolInfo MCP 工具元信息
type ToolInfo struct {
	// Name 工具名
	Name string `json:"name"`
	// Description 工具描述
	Description string `json:"description"`
	// InputSchema 参数 JSON Schema
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ListTools 列出服务器全部工具（按 NextCursor 自动翻页，兼容分页型服务器）
func (c *Client) ListTools(ctx context.Context) ([]ToolInfo, error) {
	var tools []ToolInfo
	req := mcp.ListToolsRequest{}
	for {
		result, err := c.mcp.ListTools(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("list mcp tools: %w", err)
		}
		for _, t := range result.Tools {
			schema := t.RawInputSchema
			if len(schema) == 0 {
				schema, err = json.Marshal(t.InputSchema)
				if err != nil {
					continue
				}
			}
			tools = append(tools, ToolInfo{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: schema,
			})
		}
		if result.NextCursor == "" {
			break
		}
		req.Params.Cursor = result.NextCursor
	}
	return tools, nil
}

// CallTool 调用工具并返回文本结果（多段文本以换行拼接；isError 结果转为 error）
func (c *Client) CallTool(ctx context.Context, name string, arguments json.RawMessage) (string, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	if len(arguments) > 0 {
		req.Params.Arguments = map[string]any{}
		if err := json.Unmarshal(arguments, &req.Params.Arguments); err != nil {
			return "", fmt.Errorf("invalid tool arguments: %w", err)
		}
	}
	result, err := c.mcp.CallTool(ctx, req)
	if err != nil {
		return "", fmt.Errorf("call mcp tool %s: %w", name, err)
	}
	if result.IsError {
		return "", fmt.Errorf("mcp tool %s execution failed: %s", name, textOf(result.Content))
	}
	return textOf(result.Content), nil
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.mcp.Close()
}

// textOf 拼接内容段的文本部分（非文本段以占位描述保留）
func textOf(contents []mcp.Content) string {
	parts := make([]string, 0, len(contents))
	for _, content := range contents {
		switch c := content.(type) {
		case mcp.TextContent:
			parts = append(parts, c.Text)
		case mcp.ImageContent:
			parts = append(parts, "[image content]")
		case mcp.AudioContent:
			parts = append(parts, "[audio content]")
		case mcp.ResourceLink:
			parts = append(parts, "[resource link: "+c.URI+"]")
		case mcp.EmbeddedResource:
			parts = append(parts, "[embedded resource]")
		default:
			parts = append(parts, "[unknown content]")
		}
	}
	return strings.Join(parts, "\n")
}
