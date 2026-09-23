package dbi

import (
	"strings"
	"sync"

	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/stringx"
)

// ------------------------- 元数据 sql 模板解析 -------------------------
//
// 各方言元数据SQL模板由方言包自持（//go:embed 与方言实现同居一处，新增方言时包内自包含），
// dbi 仅提供通用的解析与缓存能力（SQLTemplates）

// SQLTemplates 方言元数据SQL模板：解析「--KEY 备注说明」分段格式的sql文件内容，
// 按备注key取用并缓存。格式：段落以分隔线切分，每段首行为 --KEY 备注信息
// （如 --MYSQL_TABLE_INFO 表详细信息），正文为实际sql
//
// 用法（方言包内）：
//
//	//go:embed meta.sql
//	var metaSQLFile string
//	var metaSQL = dbi.NewSQLTemplates(metaSQLFile)
type SQLTemplates struct {
	content string
	mu      sync.RWMutex // 保护 cache 的并发读写
	cache   map[string]string
}

func NewSQLTemplates(content string) *SQLTemplates {
	return &SQLTemplates{content: content, cache: make(map[string]string, 20)}
}

// Get 获取key对应的sql内容，首次访问时解析全量段落并缓存
func (t *SQLTemplates) Get(key string) string {
	t.mu.RLock()
	sql := t.cache[key]
	t.mu.RUnlock()
	if sql != "" {
		return sql
	}

	allSQL := t.content
	sqls := strings.Split(allSQL, "---------------------------------------")
	var resSQL string
	for _, sql := range sqls {
		sql = stringx.TrimSpaceAndBr(sql)
		if sql == "" {
			continue
		}
		// 获取sql第一行的sql备注信息如：--MYSQL_TABLE_MA 表信息元数据
		info := strings.SplitN(sql, "\n", 2)
		if len(info) < 2 {
			// 内容只有一行（无实际sql），跳过，避免越界
			continue
		}
		// 获取sql key；如：MYSQL_TABLE_MA，格式不合法则跳过
		keyParts := strings.Split(strings.Split(info[0], " ")[0], "--")
		if len(keyParts) < 2 {
			continue
		}
		sqlKey := keyParts[1]
		// 原始sql，即去除第一行的key与备注信息
		rowSQL := info[1]
		if key == sqlKey {
			resSQL = rowSQL
		}
		t.mu.Lock()
		t.cache[sqlKey] = rowSQL
		t.mu.Unlock()
	}
	if resSQL == "" {
		logx.Errorf("sql metadata key not found: %s", key)
	}
	return resSQL
}
