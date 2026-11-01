package mssql

// SQL Server 连接加密状态读取（可选能力 TLSStatusInspector 的 mssql 实现）。
//
// 数据源 sys.dm_exec_connections.encrypt_option 只读当前会话（@@SPID）那一行，
// 取值 'TRUE'/'FALSE'。该 DMV 需要 VIEW SERVER STATE 权限：账号没有权限时查询报错，
// 由上层如实标注「不可判定」，不回落到「未加密」或「已加密」。

import (
	"context"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

var _ dbi.TLSStatusInspector = (*MssqlMetadata)(nil)

// mssqlTLSStatusSQL 常量 SQL：只读自身会话，不含外部输入
const mssqlTLSStatusSQL = "SELECT encrypt_option FROM sys.dm_exec_connections WHERE session_id = @@SPID"

// TLSStatus 返回本连接是否加密。
//
// mssql 没有「当前连接的协议版本/套件」视图（TLS 细节不落到 DMV），故只给布尔事实，
// Version/Cipher 留空而不是编造。
func (mm *MssqlMetadata) TLSStatus(ctx context.Context) (*dbi.TLSStatus, error) {
	_, rows, err := mm.di.QueryContext(ctx, mssqlTLSStatusSQL)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, dbi.ErrTLSStatusUnavailable
	}
	return &dbi.TLSStatus{
		Available: true,
		Encrypted: strings.EqualFold(cast.ToString(rows[0]["encrypt_option"]), "TRUE"),
	}, nil
}
