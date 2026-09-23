package dbi

import (
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"sync"
	"time"
)

// schemaCacheTTL 服务端元数据缓存有效期：兜底「他端/其他会话改了表结构，本连接未收到失效信号」的陈旧。
const schemaCacheTTL = 5 * time.Minute

// schemaEntry 单条缓存：值 + 写入时间。
type schemaEntry struct {
	val any
	at  time.Time
}

// schemaCache 进程内、按逻辑库连接（DbInfo 单例）的 Schema 元数据缓存。
//
// 归属说明：缓存策略集中于 MetadataReader 门面消费（本仓约定「facade absorbs probing & caching」），
// 方言 MetadataProvider 实现不感知缓存；缓存跨 HTTP 请求存活，TTL 到期或结构变更后主动失效即重新内省。
//
// 作用域正确性：连接池对每个 connId(dbId:database) 仅持有一个 DbConn/DbInfo（CachePool MaxConns=1），
// 故本缓存即「每逻辑库」单例；连接关闭/重建会连带丢弃 DbInfo，缓存自然作废，无泄漏、无跨库串味。
type schemaCache struct {
	mu    sync.Mutex
	items map[string]schemaEntry
}

func newSchemaCache() *schemaCache {
	return &schemaCache{items: make(map[string]schemaEntry)}
}

// get 命中且未过期时返回缓存值；过期条目顺手删除，避免常驻内存。
func (c *schemaCache) get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.items[key]
	if !ok {
		return nil, false
	}
	if time.Since(entry.at) > schemaCacheTTL {
		delete(c.items, key)
		return nil, false
	}
	return entry.val, true
}

// set 写入缓存。仅应在调用方确认内省成功（err==nil）后调用——错误结果绝不入缓存。
func (c *schemaCache) set(key string, val any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = schemaEntry{val: val, at: time.Now()}
}

// invalidate 清空该连接的全部元数据缓存（DDL/结构变更后调用）。
func (c *schemaCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]schemaEntry)
}

// invalidateIfDDL 语句执行成功后调用：该语句为 DDL（建/改/删表、索引、视图、授权等）时清空元数据缓存。
//
// 结构变更与缓存失效必须收口在执行层：DDL 可来自 SQL 控制台、方言建表/复制表、数据迁移导入、
// 同步 Schema 演化等任意调用方，逐处在业务层补失效必然漏项（复制表曾在 API 层失效，方言层直接调用者
// 仍会读到旧表清单，最长等一个 TTL）。非 DDL 语句不改变表清单与列结构，不触发失效。
//
// 语句类型判定复用 sqlparser 的关键字兜底（与执行路由同源，已跳过前导空白与注释）。
func (di *DbInfo) invalidateIfDDL(execSQL string) {
	// 无缓存（未建立连接的 DbInfo）与无方言后端（纯测试构造）都无从判断，直接跳过
	if di.schemaCache == nil || di.Backend == nil {
		return
	}
	dialect := di.Backend.GetDialect(di)
	if dialect == nil {
		return
	}
	if sqlparser.StatementTypeByKeyword(dialect.GetSQLSplitter(), execSQL) != sqlstmt.StmtTypeDDL {
		return
	}
	di.InvalidateSchemaCache()
}
