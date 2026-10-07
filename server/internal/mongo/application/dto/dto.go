package dto

import (
	"encoding/json"

	"mayfly-go/internal/mongo/application/mongodoc"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Database 库信息（driver 的 DatabaseSpecification 无 json tag，字段名大写会直接漏到前端契约上，
// 因此在这里显式转成小驼峰，全站响应字段风格保持一致）
type Database struct {
	Name       string `json:"name"`
	SizeOnDisk int64  `json:"sizeOnDisk"`
	Empty      bool   `json:"empty"`
}

// Collection 集合信息。Type 为 "collection" / "view"，只读视图前端需据此收敛操作入口。
type Collection struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	ReadOnly bool   `json:"readOnly"`
}

// DocQuery 文档查询条件。filter/sort/projection 已由 API 层解码为保留字段顺序的 BSON 文档。
type DocQuery struct {
	Database   string
	Collection string
	Filter     bson.D
	Sort       bson.D
	Projection bson.D
	Skip       int64
	Limit      int64 // 原始请求值，生效值由 QueryPage.Limit 回传
	WithCount  bool  // 是否额外统计匹配总数（大集合上开销可观，由调用方显式开启）
}

// DocPage 一次查询的结果。
type DocPage struct {
	Docs []*mongodoc.QueryDoc `json:"docs"`
	// Total 匹配总数；未开启统计时为 -1，前端不得把已加载条数或集合总条数当成匹配数
	Total int64 `json:"total"`
	// Truncated 结果因上限被截断，前端应提示收窄条件而不是「就这么多条」
	Truncated bool `json:"truncated"`
	// Limit 实际生效的条数上限
	Limit int64 `json:"limit"`
	// Stats 集合统计。与查询同一次动作返回，避免前端为头部读数再发一跳
	Stats *CollectionStats `json:"stats,omitempty"`
	// StatsError 统计不可用的原因（如目标是视图、账号无 collStats 权限）。
	// 统计失败不阻断查询也不让整页报错：读数区说明原因即可，数据本身已经取到了
	StatsError string `json:"statsError,omitempty"`
}

// CollectionStats 集合头部读数。
//
// 只保留能驱动决策的几个值：$collStats 的原始结果会把整块 WiredTiger 内部指标（几十 KB）
// 一起透给前端，既拖慢响应也没人看。字段取平与取嵌套（storageStats）的差异在服务端吸收，
// 不同服务版本给前端的契约保持一致。
type CollectionStats struct {
	Ns              string `json:"ns"`
	Count           int64  `json:"count"`
	AvgObjSize      int64  `json:"avgObjSize"`
	StorageSize     int64  `json:"storageSize"`
	FreeStorageSize int64  `json:"freeStorageSize"`
	TotalIndexSize  int64  `json:"totalIndexSize"`
	TotalSize       int64  `json:"totalSize"`
	NIndexes        int64  `json:"nindexes"`
	// IndexSizes 索引名 → 字节数，供索引面板逐行展示大小
	IndexSizes map[string]int64 `json:"indexSizes,omitempty"`
}

// CollectionMeta 集合元信息：一次面板打开只需要一次往返（统计 + 索引列表）。
type CollectionMeta struct {
	Stats      *CollectionStats `json:"stats,omitempty"`
	StatsError string           `json:"statsError,omitempty"`
	Indexes    []*IndexInfo     `json:"indexes"`
	// TotalDocs 集合总文档数（等于 stats.count，单独给出便于前端直接读）
	TotalDocs int64 `json:"totalDocs"`
}

// DocInsert 插入文档。Docs 为已解码的文档列表，支持一次提交多条。
type DocInsert struct {
	Database   string
	Collection string
	Docs       []bson.D
}

// AggQuery 聚合查询。Pipeline 为已校验的 stage 列表（至少一个）。
type AggQuery struct {
	Database     string
	Collection   string
	Pipeline     []bson.D
	AllowDiskUse bool
	// Explain 只解释不执行：服务端不会跑写 stage，因此不需要写权限
	Explain bool
}

// IndexKey 单个索引键：字段名 + 方向。复合索引的顺序是语义的一部分，所以用数组而不是 map。
type IndexKey struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// IndexInfo 索引信息。Spec 为服务返回的完整定义（保真 JSON），用于复制定义与排查选项差异。
type IndexInfo struct {
	Name      string          `json:"name"`
	Keys      []IndexKey      `json:"keys"`
	Spec      json.RawMessage `json:"spec"`
	SizeBytes int64           `json:"sizeBytes"`
	// Blocking/Unique 等常见标志只用于列表展示，真实选项以 Spec 为准
	Unique bool `json:"unique"`
}

// BatchUpdate 按条件批量更新。
//
// ExpectCount 是调用方确认过的命中数：服务端先统计再比对，不一致就中止。
// 没这道护栏时，一个写错的 filter 会静默改掉整个集合，且无法回退。
type BatchUpdate struct {
	Database    string
	Collection  string
	Filter      bson.D
	Update      *mongodoc.UpdateSpec
	Upsert      bool
	ExpectCount int64
}

// BatchDelete 按条件批量删除，同样要求命中数先被确认
type BatchDelete struct {
	Database    string
	Collection  string
	Filter      bson.D
	ExpectCount int64
}

// ExportRequest 导出取数请求。Limit 由服务端按配置上限取，不受调用方无限拉大
type ExportRequest struct {
	Database   string
	Collection string
	Filter     bson.D
	Format     string
}

// ExportFile 导出结果
type ExportFile struct {
	Content     []byte `json:"-"`
	ContentType string `json:"contentType"`
	// Count 实际写出的文档数（等于服务端最大结果集时即为被截断）
	Count int `json:"count"`
}

// DocUpdate 以主键令牌定位、以「编辑后的完整文档」更新单个文档。
//
// 服务端拿当前存储值做差异比对，因此：
//   - 从 JSON 里删掉一个 key 等价于 $unset 该字段（旧实现只有 $set，删字段静默无效）；
//   - _id 由服务端剔除，不依赖调用方自觉；
//   - BaseHash 为读取该文档时服务端下发的内容指纹，不匹配即拒绝写入，避免静默覆盖他人的修改。
type DocUpdate struct {
	Database   string
	Collection string
	IDToken    string
	BaseHash   string
	Doc        bson.D
}

// DocDelete 按主键令牌批量删除。
type DocDelete struct {
	Database   string
	Collection string
	IDTokens   []string
}

// WriteResult 写操作结果。各计数字段来自驱动，语义为「实际影响条数」，
// 调用方不能把「请求删除 n 条」等同于「删除了 n 条」。
type WriteResult struct {
	MatchedCount  int64 `json:"matchedCount"`
	ModifiedCount int64 `json:"modifiedCount"`
	DeletedCount  int64 `json:"deletedCount"`
	InsertedCount int64 `json:"insertedCount"`
	// UpsertedCount 批量更新开启 upsert 时新增的文档数
	UpsertedCount int64 `json:"upsertedCount"`
	// NoChange 更新请求与存储内容完全一致，未产生任何写入
	NoChange bool `json:"noChange"`
	// InsertedIDs 插入成功时各文档的主键令牌，前端可据此立刻定位新文档而无需重新查询
	InsertedIDs []string `json:"insertedIds,omitempty"`
}
