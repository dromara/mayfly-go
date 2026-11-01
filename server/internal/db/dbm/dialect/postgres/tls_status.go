package postgres

// postgres 连接加密状态读取（可选能力 TLSStatusInspector 的 postgres 实现）。
//
// 数据源 pg_stat_ssl 对应当前后端进程（pid = pg_backend_pid()），普通账号即可读自己的行，
// 不需要 VIEW SERVER STATE 一类权限。ssl=false 表示这条连接以明文运行。

import (
	"context"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

var _ dbi.TLSStatusInspector = (*PgsqlMetadata)(nil)

// pgTLSStatusSQL 常量 SQL：不含任何外部输入（pid 用 pg_backend_pid() 现算）
const pgTLSStatusSQL = "SELECT ssl, COALESCE(version, '') AS version, COALESCE(cipher, '') AS cipher FROM pg_stat_ssl WHERE pid = pg_backend_pid()"

// TLSStatus 返回本连接是否加密及协商出的协议版本与套件。
func (pm *PgsqlMetadata) TLSStatus(ctx context.Context) (*dbi.TLSStatus, error) {
	_, rows, err := pm.di.QueryContext(ctx, pgTLSStatusSQL)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		// 查不到本进程的行属观测缺口，不是「没加密」：交给上层标不可判定
		return nil, dbi.ErrTLSStatusUnavailable
	}
	row := rows[0]
	return &dbi.TLSStatus{
		Available: true,
		Encrypted: cast.ToBool(row["ssl"]),
		Version:   cast.ToString(row["version"]),
		Cipher:    cast.ToString(row["cipher"]),
	}, nil
}
