package form

// DbAiTermForm 文生 SQL 业务词典条目表单。
//
// Tables 允许为空：只写口径说明、不指向具体表的条目同样有价值（「金额单位是分」
// 这类全局约束不属于任何一张表），因此这里不加 required。
type DbAiTermForm struct {
	Id      uint64   `json:"id"`
	DbId    uint64   `binding:"required" json:"dbId"`
	Schema  string   `json:"schema"`
	Term    string   `binding:"required" json:"term"`
	Tables  []string `json:"tables"`
	Details string   `json:"details"`
}
