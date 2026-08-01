package client

import (
	"fmt"
	"net/url"
)

// --- MongoDB 相关 API ---

// ListMongo 获取 MongoDB 实例列表
func (c *ApiClient) ListMongo() (*PageResult, error) {
	var result PageResult
	if err := c.Get("/mongos", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetMongoDatabases 获取 MongoDB 数据库列表
func (c *ApiClient) GetMongoDatabases(mongoId uint64) (interface{}, error) {
	path := fmt.Sprintf("/mongos/%d/databases", mongoId)
	var result interface{}
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMongoCollections 获取 MongoDB 集合列表
func (c *ApiClient) GetMongoCollections(mongoId uint64, database string) (interface{}, error) {
	path := fmt.Sprintf("/mongos/%d/collections?database=%s", mongoId, url.QueryEscape(database))
	var result interface{}
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// MongoRunCommand 执行 MongoDB 命令
func (c *ApiClient) MongoRunCommand(mongoId uint64, database string, command map[string]interface{}) (interface{}, error) {
	body := map[string]interface{}{
		"database": database,
		"command":  command,
	}
	var result interface{}
	if err := c.Post(fmt.Sprintf("/mongos/%d/run-command", mongoId), body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// MongoFind 执行 MongoDB 查询
func (c *ApiClient) MongoFind(mongoId uint64, database, collection string, filter map[string]interface{}, limit int64, skip int64) (interface{}, error) {
	body := map[string]interface{}{
		"database":   database,
		"collection": collection,
		"filter":     filter,
		"limit":      limit,
		"skip":       skip,
	}
	var result interface{}
	if err := c.Post(fmt.Sprintf("/mongos/%d/command/find", mongoId), body, &result); err != nil {
		return nil, err
	}
	return result, nil
}
