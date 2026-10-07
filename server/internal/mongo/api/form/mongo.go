package form

import "encoding/json"

// Mongo 实例登记表单。
//
// Uri 允许留空表示「保持原值不变」：列表与详情接口不再回传明文连接串（其中含账号密码），
// 编辑时无法回填，因此以留空表达不修改，仅新建时要求必填（见 application.SaveMongo）。
type Mongo struct {
	Id                 uint64   `json:"id"`
	Uri                string   `json:"uri"`
	Name               string   `binding:"required" json:"name"`
	SshTunnelMachineId int      `json:"sshTunnelMachineId"` // ssh隧道机器id
	TagCodePaths       []string `binding:"required" json:"tagCodePaths"`
}

// QueryDocsForm 文档查询。
//
// filter/sort/projection 用 json.RawMessage 而不是 map[string]any：
//   - 复合排序键的顺序是语义的一部分，map 交给 driver 编码时按 Go 随机迭代序输出，
//     翻页会出现重复与漏行；
//   - RawMessage 同时让 Extended JSON 的类型包装（$oid/$date 等）原样进入解码，
//     按 ObjectId 形状的猜测不再需要。
type QueryDocsForm struct {
	Database   string          `binding:"required" json:"database"`
	Collection string          `binding:"required" json:"collection"`
	Filter     json.RawMessage `json:"filter"`
	Sort       json.RawMessage `json:"sort"`
	Projection json.RawMessage `json:"projection"`
	Skip       int64           `json:"skip"`
	Limit      int64           `json:"limit"`
	WithCount  bool            `json:"withCount"`
}

// InsertDocsForm 插入文档。Docs 为文档 JSON 数组文本（也允许单个文档对象）。
type InsertDocsForm struct {
	Database   string          `binding:"required" json:"database"`
	Collection string          `binding:"required" json:"collection"`
	Docs       json.RawMessage `json:"docs" binding:"required"`
}

// UpdateDocForm 更新单个文档。
//
// Doc 是「编辑后的完整文档」，$set/$unset 由服务端与当前存储值比对得出；
// BaseHash 取自查询结果，服务端据此拒绝覆盖他人刚写入的版本。
type UpdateDocForm struct {
	Database   string          `binding:"required" json:"database"`
	Collection string          `binding:"required" json:"collection"`
	IDToken    string          `binding:"required" json:"idToken"`
	BaseHash   string          `json:"baseHash"`
	Doc        json.RawMessage `json:"doc" binding:"required"`
}

// DeleteDocsForm 按主键令牌批量删除。
//
// 用 POST 而不是 DELETE：主键值可能含逗号与特殊字符，只能按 JSON 数组放在请求体里传输。
type DeleteDocsForm struct {
	Database   string   `binding:"required" json:"database"`
	Collection string   `binding:"required" json:"collection"`
	IDTokens   []string `binding:"required,dive,required" json:"idTokens"`
}

// RunCommandForm 执行管理命令。Command 为命令 JSON 文本（保留字段顺序与类型包装）。
type RunCommandForm struct {
	Database string          `binding:"required" json:"database"`
	Command  json.RawMessage `json:"command" binding:"required"`
}

// CreateCollectionForm 创建集合。
type CreateCollectionForm struct {
	Collection string `binding:"required" json:"collection"`
}

// CreateIndexesForm 创建索引。Specs 为索引定义数组原文（含 key/name/unique/expireAfterSeconds 等选项），
// 服务端只校验形状、原样交给 Mongo，避免手写选项映射时默默丢字段。
type CreateIndexesForm struct {
	Specs json.RawMessage `json:"indexes" binding:"required"`
}

// AggregateForm 聚合查询。Pipeline 为 stage 数组原文，顺序即执行顺序。
type AggregateForm struct {
	Database     string          `binding:"required" json:"database"`
	Collection   string          `binding:"required" json:"collection"`
	Pipeline     json.RawMessage `json:"pipeline" binding:"required"`
	AllowDiskUse bool            `json:"allowDiskUse"`
	Explain      bool            `json:"explain"`
}

// BatchUpdateForm 按条件批量更新。
//
// ExpectCount 是调用方在预览里确认过的命中数：服务端先统计再比对，不一致即中止；
// 否则「预览了 5 条、实际改了几万条」这种事故没有任何拦截点。
type BatchUpdateForm struct {
	Database    string          `binding:"required" json:"database"`
	Collection  string          `binding:"required" json:"collection"`
	Filter      json.RawMessage `json:"filter"`
	Update      json.RawMessage `json:"update" binding:"required"`
	Upsert      bool            `json:"upsert"`
	ExpectCount int64           `json:"expectCount" binding:"required"`
}

// BatchDeleteForm 按条件批量删除，同样要求确认命中数
type BatchDeleteForm struct {
	Database    string          `binding:"required" json:"database"`
	Collection  string          `binding:"required" json:"collection"`
	Filter      json.RawMessage `json:"filter"`
	ExpectCount int64           `json:"expectCount" binding:"required"`
}

// ExportForm 导出取数请求。不支持传 limit：条数上限由服务端配置决定，
// 让它能被请求参数拉大等于开一道无限取数的口。
type ExportForm struct {
	Database   string          `binding:"required" json:"database"`
	Collection string          `binding:"required" json:"collection"`
	Filter     json.RawMessage `json:"filter"`
	Format     string          `json:"format" binding:"required"`
}
