package dto

import (
	"io"
	"mayfly-go/internal/db/dbm/dbi"
)

type DbSqlExecReq struct {
	DbId      uint64
	Db        string
	Sql       string // 需要执行的sql，支持多条
	Remark    string // 执行备注
	DbConn    *dbi.DbConn
	CheckFlow bool // 是否检查存储审批流程
}

type DbSqlExecRes struct {
	Sql      string             `json:"sql"`      // 执行的sql
	ErrorMsg string             `json:"errorMsg"` // 若执行失败，则将失败内容记录到该字段
	Columns  []*dbi.QueryColumn `json:"columns"`  // 响应的列信息
	Res      []map[string]any   `json:"res"`      // 响应结果
}

type SqlReaderExec struct {
	DbConn *dbi.DbConn

	Reader   io.Reader
	Filename string

	ClientId string // 客户端id，若存在则会向其发送执行进度消息
}
