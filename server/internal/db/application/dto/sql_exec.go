package dto

import (
	"io"
	"mayfly-go/internal/db/dbm/dbi"
)

type DbSQLExecReq struct {
	// ExecId 交互式执行标识：非空时本次执行登记进运行中表，可通过 CancelExec 中断。
	// 非交互入口（工单回放、文件导入）留空，即不可被取消——它们没有对应的等待界面
	ExecId string

	DbId   uint64
	Db     string
	SQL    string // 需要执行的sql，支持多条
	Remark string // 执行备注
	DbConn *dbi.DbConn
	// RequireWarnAck 该入口能否弹「直接执行 / 转审批」确认。
	// 由**客户端**声明而不是按语句条数推断：SQL 控制台的批量执行在客户端也是一条一条发的，
	// 条数猜不出「这是一批」，于是每条都会追问一次，批量执行变成逐条弹窗
	RequireWarnAck bool
	// WarnAcknowledged 操作者已确认过「仅提醒」命中，重放本次执行。
	// 由前端在确认框里选「直接执行」时置上，不是客户端可自由选择的策略开关
	WarnAcknowledged bool

	// Atomic 为真时整批语句共用一个事务：全部成功才提交，任一条失败即整批回滚。
	// 只允许 INSERT/UPDATE/DELETE 进原子批次（判据见 application.atomicClasses）：
	// DDL 在 MySQL 等方言隐式提交、SELECT 结果不参与回滚，允许它们只会给出无法兑现的「可回滚」承诺
	Atomic bool
}

type DbSQLExecRes struct {
	SQL      string             `json:"sql"`      // 执行的sql
	ErrorMsg string             `json:"errorMsg"` // 若执行失败，则将失败内容记录到该字段
	Columns  []*dbi.QueryColumn `json:"columns"`  // 响应的列信息
	Res      []map[string]any   `json:"res"`      // 响应结果

	// Truncated 结果集是否因超出行数上限而未取完。
	//
	// 必须单独给一个位而不是拿「行数==上限」去猜：正好那么行的结果与被截断的结果
	// 在界面上长得一样，用户会拿一份缺尾巴的数据去做判断，还会把它导出去
	Truncated bool `json:"truncated,omitempty"`
	// RowLimit 本次生效的行数上限（Truncated 为 true 时才有意义，0 表示不限）
	RowLimit int `json:"rowLimit,omitempty"`

	// Notices 不阻断执行但操作者需要知道的策略结论（「仅提醒」级别）。
	// 标题是前端 i18n key，由前端翻译，后端不下发文案
	Notices []*PolicyNotice `json:"notices,omitempty"`

	// NeedApproval 该语句因触发策略被拦下且可以通过提单审批放行。
	//
	// 必须单独给一个布尔位：「需审批」要给提交工单入口，「已被禁止」不能给，
	// 两者文案同为失败提示，让前端匹配字符串区分会在改文案时静默失效
	NeedApproval bool `json:"needApproval,omitempty"`

	// WarnAck 该语句因命中「仅提醒」而等待操作者确认（未执行）。
	//
	// 与 NeedApproval 一样必须是结构化标记：确认框与提单入口是两个完全不同的后续动作，
	// 靠提示文案区分会在改文案时把用户引向错误方向
	WarnAck bool `json:"warnAck,omitempty"`
}

// PolicyNotice 一条策略提醒
type PolicyNotice struct {
	Title  string         `json:"title"`
	Detail map[string]any `json:"detail,omitempty"`
}

type SQLReaderExec struct {
	DbConn *dbi.DbConn

	Reader   io.Reader
	Filename string

	ClientId string // 客户端id，若存在则会向其发送执行进度消息
	UploadId string // 上传id，用于记录上传进度
}

// QueryExportReq 全量导出一次查询结果集的入参
type QueryExportReq struct {
	DbId   uint64
	Db     string
	SQL    string
	Format string // csv/json，来自导出格式注册表
}

// SqlRollback 依据执行记录旧值生成的回滚 SQL。
//
// 只交回文本而不是「已执行的回滚」：回滚本身是一次新的变更，必须走它自己的预检与审批，
// 平台不能开一条「回滚不过策略」的通道，否则任何变更都能先斩后奏，再用回滚掩盖。
//
// Rows 是「本次生成的语句条数」，不等于原执行的影响行数：旧值采集有上限，
// 拿它当「全部都能回滚」会把截断这件事藏起来（截断时 Note 必须非空）
type SqlRollback struct {
	ExecId uint64   `json:"execId"`
	Type   int8     `json:"type"`
	Sqls   []string `json:"sqls"`
	Rows   int      `json:"rows"`
	// Note 如实说明（如旧值采集时被截断、原语句未能解析、未做并发校验），无则空
	Note string `json:"note,omitempty"`
	// Verified 生成前是否与执行后的行快照做过并发比对（仅 UPDATE 有意义）。
	// false 表示降级生成（快照缺失或回查失败）：语句仍可用，但可能覆盖这些行在此之后的修改
	Verified bool `json:"verified"`
	// Conflicts 未生成回滚的行：它们在执行之后又被修改、删除或无法定位，
	// 对这些行盲生成会把并发修改一并覆盖，故不予生成，交由人工对账
	Conflicts []SqlRollbackConflict `json:"conflicts,omitempty"`
}

// SqlRollbackConflict 一行未生成回滚的原因。
//
// 只带行标识与原因枚举、不带列值：比对在服务端完成，行快照属于业务数据，不因「生成回滚」
// 就把它扩散到前端；行标识（如主键值）本就在回滚 SQL 里出现，不构成额外暴露
type SqlRollbackConflict struct {
	// Key 行标识：UPDATE 用主键值（如 "id=2, tenant=3"），INSERT 用插入行序号（如 "row #2"）
	Key string `json:"key"`
	// Reason 原因枚举，取值见 RollbackConflict* 常量
	Reason string `json:"reason"`
}

// 回滚冲突原因枚举（前端据它映射展示文案）
const (
	// RollbackConflictConcurrentModified 该行在执行之后又被修改，旧值写回会覆盖这次修改
	RollbackConflictConcurrentModified = "concurrent-modified"
	// RollbackConflictRowMissing 该行已不存在（被删除或并发删除）
	RollbackConflictRowMissing = "row-missing"
	// RollbackConflictUnknownPosition 按插入值回查不到新插入的行，无法定位
	RollbackConflictUnknownPosition = "unknown-position"
	// RollbackConflictAmbiguousMatch 按插入值回查到多行，定位不唯一
	RollbackConflictAmbiguousMatch = "ambiguous-match"
)
