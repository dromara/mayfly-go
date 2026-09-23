package dbi

// TargetTableMeta 目标表元信息，用于生成 upsert/merge 类插入语句。
// 由调用方（持有真实目标库连接）预先查询好传入，保证 SQLGenerator 为纯函数：
// 生成 SQL 的过程中不做任何数据库查询，避免在伪连接（仅生成SQL）场景下 panic。
//
// 「如何从表内省结果推导出冲突键/自增列」（即 upsert 冲突键选择策略）属于数据同步用例的业务决策，
// 已上移至 application/sync.BuildTargetTableMeta；本包仅保留承载结果的类型与 GenInsert 契约。
type TargetTableMeta struct {
	UniqueColumns   []string // 主键与唯一索引涉及到的列名（不含引号）
	IdentityColumns []string // 自增列名（生成merge语句时需排除）
}
