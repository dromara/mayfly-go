package client

import (
	"fmt"
	"net/url"
)

// --- Redis 相关 API ---

// ListRedis 获取 Redis 实例列表
func (c *ApiClient) ListRedis() (*PageResult, error) {
	var result PageResult
	if err := c.Get("/redis", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ExecRedisCmd 执行 Redis 命令
// cmd 为完整的命令字符串，如 "PING" 或 "SET key value"，会自动解析为数组格式
func (c *ApiClient) ExecRedisCmd(redisId uint64, db int, cmd string) (interface{}, error) {
	cmdArgs := parseRedisCmdString(cmd)

	body := map[string]interface{}{
		"id":  redisId,
		"db":  db,
		"cmd": cmdArgs,
	}

	var result interface{}
	if err := c.Post(fmt.Sprintf("/redis/%d/%d/run-cmd", redisId, db), body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ScanRedisKeys 扫描 Redis Key 列表
func (c *ApiClient) ScanRedisKeys(redisId uint64, db int, match string, count int64, cursor map[string]uint64) (interface{}, error) {
	if cursor == nil {
		cursor = make(map[string]uint64)
	}
	body := map[string]interface{}{
		"cursor": cursor,
		"match":  match,
		"count":  count,
	}
	var result interface{}
	if err := c.Post(fmt.Sprintf("/redis/%d/%d/scan", redisId, db), body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// parseRedisCmdString 将命令字符串解析为参数数组（支持引号）
func parseRedisCmdString(cmd string) []string {
	var args []string
	var current string
	inSingleQuote := false
	inDoubleQuote := false

	for _, ch := range cmd {
		switch {
		case ch == '\'' && !inDoubleQuote:
			inSingleQuote = !inSingleQuote
		case ch == '"' && !inSingleQuote:
			inDoubleQuote = !inDoubleQuote
		case ch == ' ' && !inSingleQuote && !inDoubleQuote:
			if current != "" {
				args = append(args, current)
				current = ""
			}
		default:
			current += string(ch)
		}
	}
	if current != "" {
		args = append(args, current)
	}
	return args
}

// GetKeyInfo 获取 Redis Key 详情
func (c *ApiClient) GetKeyInfo(redisId uint64, db int, key string) (interface{}, error) {
	path := fmt.Sprintf("/redis/%d/%d/key-info?key=%s", redisId, db, url.QueryEscape(key))
	var result interface{}
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetKeyTTL 获取 Redis Key 的 TTL
func (c *ApiClient) GetKeyTTL(redisId uint64, db int, key string) (interface{}, error) {
	path := fmt.Sprintf("/redis/%d/%d/key-ttl?key=%s", redisId, db, url.QueryEscape(key))
	var result interface{}
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetKeyMemoryUsage 获取 Redis Key 的内存占用
func (c *ApiClient) GetKeyMemoryUsage(redisId uint64, db int, key string) (interface{}, error) {
	path := fmt.Sprintf("/redis/%d/%d/key-memuse?key=%s", redisId, db, url.QueryEscape(key))
	var result interface{}
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// --- Dashboard API ---

// GetAssetStats 获取资产统计信息
func (c *ApiClient) GetAssetStats() (map[string]int, error) {
	stats := make(map[string]int)

	if dbs, err := c.ListDbs(); err == nil && dbs != nil {
		stats["dbs"] = int(dbs.Total)
	}

	if machines, err := c.ListMachines(); err == nil && machines != nil {
		stats["machines"] = int(machines.Total)
		onlineCount := 0
		for _, m := range machines.List {
			if status, ok := m["status"].(float64); ok && status == 1 {
				onlineCount++
			}
		}
		stats["machines_online"] = onlineCount
	}

	if redisList, err := c.ListRedis(); err == nil && redisList != nil {
		stats["redis"] = int(redisList.Total)
	}

	return stats, nil
}

// CheckAuth 检查认证是否有效
func (c *ApiClient) CheckAuth() bool {
	_, err := c.ListDbs()
	return err == nil
}
