package dto

import (
	"io"
	"mayfly-go/internal/db/dbm/dbi"
)

type DbSQLExecReq struct {
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
}

type DbSQLExecRes struct {
	SQL      string             `json:"sql"`      // 执行的sql
	ErrorMsg string             `json:"errorMsg"` // 若执行失败，则将失败内容记录到该字段
	Columns  []*dbi.QueryColumn `json:"columns"`  // 响应的列信息
	Res      []map[string]any   `json:"res"`      // 响应结果

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
