package client

import (
	"fmt"
	"mayfly-go/cli/i18n"
	"net/url"
)

// --- 数据库相关 API ---

// ListDbs 获取数据库实例列表
func (c *ApiClient) ListDbs() (*PageResult, error) {
	var result PageResult
	if err := c.Get("/dbs", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ExecSql 执行 SQL 语句
func (c *ApiClient) ExecSql(dbId uint64, db, sql string) (interface{}, error) {
	// 服务端要求 SQL 使用 AES-192-CBC 加密（key = token[:24]），然后 base64 编码
	encryptedSql, err := AesEncryptBase64([]byte(sql), []byte(c.token))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("client.encrypt_sql_failed"), err)
	}

	body := map[string]interface{}{
		"db":  db,
		"sql": encryptedSql,
	}

	var result interface{}
	if err := c.Post(fmt.Sprintf("/dbs/%d/exec-sql", dbId), body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// TestDbConnection 测试数据库连接
func (c *ApiClient) TestDbConnection(dbId uint64, db string) error {
	result, err := c.ExecSql(dbId, db, "SELECT 1 AS test")
	if err != nil {
		return fmt.Errorf("%s: %w", i18n.T("client.connection_failed"), err)
	}
	if result == nil {
		return fmt.Errorf("%s", i18n.T("client.connection_anomaly"))
	}
	return nil
}

// GetInstanceDatabases 获取实例下的数据库列表
func (c *ApiClient) GetInstanceDatabases(authCertName string) ([]string, error) {
	path := fmt.Sprintf("/instances/databases/%s", url.PathEscape(authCertName))
	var result []string
	if err := c.Get(path, &result); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("client.get_instance_dbs_failed"), err)
	}
	return result, nil
}

// GetInstanceDatabasesById 通过实例 ID 获取数据库列表
func (c *ApiClient) GetInstanceDatabasesById(dbId uint64) ([]string, error) {
	var dbInfo map[string]interface{}
	if err := c.Get(fmt.Sprintf("/dbs/%d", dbId), &dbInfo); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("client.get_db_instance_failed"), err)
	}

	authCertName, _ := dbInfo["authCertName"].(string)
	if authCertName == "" {
		return nil, fmt.Errorf("%s", i18n.T("client.no_cert_info"))
	}

	return c.GetInstanceDatabases(authCertName)
}

// GetTableInfos 获取表信息列表
func (c *ApiClient) GetTableInfos(dbId uint64, db string) (interface{}, error) {
	path := fmt.Sprintf("/dbs/%d/t-infos?id=%d&db=%s", dbId, dbId, url.QueryEscape(db))

	var result interface{}
	if err := c.Get(path, &result); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("client.get_table_info_failed"), err)
	}
	return result, nil
}

// GetDbInfo 获取数据库信息
func (c *ApiClient) GetDbInfo(dbId uint64) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.Get(fmt.Sprintf("/dbs/%d", dbId), &result); err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("client.get_db_info_failed"), err)
	}
	return result, nil
}

// GetColumnMetadata 获取表的列元数据（含列名、类型、主键、备注等）
func (c *ApiClient) GetColumnMetadata(dbId uint64, db, tableName string) (interface{}, error) {
	path := fmt.Sprintf("/dbs/%d/c-metadata?db=%s&tableName=%s", dbId, url.QueryEscape(db), url.QueryEscape(tableName))
	var result interface{}
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetTableDDL 获取建表语句
func (c *ApiClient) GetTableDDL(dbId uint64, db, tableName string) (interface{}, error) {
	path := fmt.Sprintf("/dbs/%d/t-create-ddl?db=%s&tableName=%s", dbId, url.QueryEscape(db), url.QueryEscape(tableName))
	var result interface{}
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetTableIndex 获取表索引信息
func (c *ApiClient) GetTableIndex(dbId uint64, db, tableName string) (interface{}, error) {
	path := fmt.Sprintf("/dbs/%d/t-index?db=%s&tableName=%s", dbId, url.QueryEscape(db), url.QueryEscape(tableName))
	var result interface{}
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetDbVersion 获取数据库版本信息
func (c *ApiClient) GetDbVersion(dbId uint64, db string) (interface{}, error) {
	path := fmt.Sprintf("/dbs/%d/version?db=%s", dbId, url.QueryEscape(db))
	var result interface{}
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// DumpDb 导出数据库 SQL（返回原始字节流）
// dumpType: 1=结构, 2=数据, 3=结构+数据
func (c *ApiClient) DumpDb(dbId uint64, db, dumpType, tables string) ([]byte, error) {
	path := fmt.Sprintf("/dbs/%d/dump?db=%s&type=%s", dbId, url.QueryEscape(db), dumpType)
	if tables != "" {
		path += "&tables=" + url.QueryEscape(tables)
	}
	return c.getRaw(path)
}
