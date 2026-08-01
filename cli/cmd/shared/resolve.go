package shared

import (
	"fmt"
	"mayfly-go/cli/client"
	"mayfly-go/cli/i18n"
	"strconv"
	"strings"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

// --- 资源注册表 ---

// resourceResolver 描述一种可通过名称解析 ID 的资源
type resourceResolver struct {
	listFn func(c *client.ApiClient) ([]map[string]interface{}, error)
}

// resourceRegistry 资源名称 → 解析器映射（消除 ResolveIdByName / RegisterIdCompletion 的重复 switch）
var resourceRegistry = map[string]resourceResolver{
	"machine": {listFn: func(c *client.ApiClient) ([]map[string]interface{}, error) {
		pr, err := c.ListMachines()
		if err != nil {
			return nil, err
		}
		return pr.List, nil
	}},
	"db": {listFn: func(c *client.ApiClient) ([]map[string]interface{}, error) {
		pr, err := c.ListDbs()
		if err != nil {
			return nil, err
		}
		return pr.List, nil
	}},
	"redis": {listFn: func(c *client.ApiClient) ([]map[string]interface{}, error) {
		pr, err := c.ListRedis()
		if err != nil {
			return nil, err
		}
		return pr.List, nil
	}},
	"mongo": {listFn: func(c *client.ApiClient) ([]map[string]interface{}, error) {
		pr, err := c.ListMongo()
		if err != nil {
			return nil, err
		}
		return pr.List, nil
	}},
}

// ResolveIdByName 通过名称解析资源 ID（支持 machine/db/redis/mongo/docker/tag）
func ResolveIdByName(cmd *cobra.Command, resourceType, name string) (uint64, error) {
	if name == "" {
		return 0, nil
	}

	// 如果已经是数字，直接返回
	if id := cast.ToUint64(name); id != 0 {
		return id, nil
	}

	resolver, ok := resourceRegistry[resourceType]
	if !ok {
		return 0, fmt.Errorf("%s", i18n.T(i18n.MsgResolveUnsupported, "type", resourceType))
	}

	apiClient := NewClient(cmd)
	items, err := resolver.listFn(apiClient)
	if err != nil {
		return 0, err
	}

	// 精确匹配
	var matched []map[string]interface{}
	for _, item := range items {
		if cast.ToString(item["name"]) == name {
			matched = append(matched, item)
		}
	}

	// 模糊匹配（唯一时才使用）
	if len(matched) == 0 {
		for _, item := range items {
			if strings.Contains(cast.ToString(item["name"]), name) {
				matched = append(matched, item)
			}
		}
	}

	if len(matched) == 0 {
		return 0, fmt.Errorf("%s", i18n.T(i18n.MsgResolveNotFound, "type", resourceType, "name", name))
	}
	if len(matched) > 1 {
		return 0, fmt.Errorf("%s", i18n.T(i18n.MsgResolveMultiple, "type", resourceType, "name", name))
	}

	id := cast.ToUint64(matched[0]["id"])
	if id == 0 {
		return 0, fmt.Errorf("%s", i18n.T(i18n.MsgResolveCannotParse, "type", resourceType))
	}
	return id, nil
}

// RegisterIdCompletion 为 --id 标志注册智能补全（名称 → ID）
func RegisterIdCompletion(cmd *cobra.Command, resourceType string) {
	cmd.RegisterFlagCompletionFunc("id", func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		resolver, ok := resourceRegistry[resourceType]
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		apiClient := NewClient(c)
		items, err := resolver.listFn(apiClient)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var completions []string
		for _, item := range items {
			name := cast.ToString(item["name"])
			id := cast.ToString(item["id"])
			if toComplete == "" || strings.Contains(strings.ToLower(name), strings.ToLower(toComplete)) {
				completions = append(completions, fmt.Sprintf("%s\t%s (%s)", id, name, resourceType))
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	})
}

// FormatId 将 interface{} 格式化为 ID 字符串
func FormatId(v interface{}) string {
	return cast.ToString(cast.ToUint64(v))
}

// GetIdFromFlags 从 --id 或 --name 标志获取资源 ID
// 优先使用 --id，如果未设置则通过 --name 解析
func GetIdFromFlags(cmd *cobra.Command, resourceType string) (uint64, error) {
	// 优先使用 --id
	if cmd.Flags().Changed("id") {
		id, _ := cmd.Flags().GetUint64("id")
		if id != 0 {
			return id, nil
		}
	}

	// 尝试通过 --name 解析
	if cmd.Flags().Changed("name") {
		name, _ := cmd.Flags().GetString("name")
		if name != "" {
			return ResolveIdByName(cmd, resourceType, name)
		}
	}

	return 0, nil
}

// AddNameFlag 为命令添加 --name 标志
func AddNameFlag(cmd *cobra.Command, usage string) {
	cmd.Flags().String("name", "", usage)
}

// ParseIds 解析逗号分隔的 ID 字符串
func ParseIds(idsStr string) ([]uint64, error) {
	if idsStr == "" {
		return nil, nil
	}
	parts := strings.Split(idsStr, ",")
	ids := make([]uint64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ID '%s': %w", p, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// GetIdsFromFlags 从 --ids 标志获取多个 ID
func GetIdsFromFlags(cmd *cobra.Command) ([]uint64, error) {
	idsStr, _ := cmd.Flags().GetString("ids")
	return ParseIds(idsStr)
}

// Pagination 分页参数
type Pagination struct {
	Page     int
	PageSize int
}

// GetPagination 从命令标志获取分页参数
func GetPagination(cmd *cobra.Command) Pagination {
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return Pagination{Page: page, PageSize: pageSize}
}

// AddPaginationFlags 为命令添加统一的分页标志
func AddPaginationFlags(cmd *cobra.Command) {
	cmd.Flags().Int("page", 1, i18n.T(i18n.MsgFlagPage))
	cmd.Flags().Int("page-size", 20, i18n.T(i18n.MsgFlagPageSize))
}
