package mysql

// mysql 连接加密状态读取（可选能力 TLSStatusInspector 的 mysql 实现）。
//
// 用 SHOW STATUS 而不是 performance_schema：前者只反映**当前会话**，且不需要额外授权。
// Ssl_cipher 为空就是「这条连接没加密」——服务端允许明文时语句照样成功，只是值为空，
// 因此不能拿「查询成功」当成「已加密」。

import (
	"context"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

var _ dbi.TLSStatusInspector = (*MysqlMetadata)(nil)

// TLSStatus 返回本连接协商出的协议版本与加密套件。
func (md *MysqlMetadata) TLSStatus(ctx context.Context) (*dbi.TLSStatus, error) {
	_, rows, err := md.di.QueryContext(ctx, "SHOW STATUS WHERE Variable_name IN ('Ssl_cipher', 'Ssl_version')")
	if err != nil {
		return nil, err
	}

	values := make(map[string]string, len(rows))
	for _, row := range rows {
		// 列名大小写随服务端与驱动的返回形态，按小写归一后再取值，避免键名比对落空
		var name, value string
		for key, val := range row {
			switch strings.ToLower(key) {
			case "variable_name":
				name = strings.ToLower(cast.ToString(val))
			case "value":
				value = cast.ToString(val)
			}
		}
		if name != "" {
			values[name] = value
		}
	}

	cipher := values["ssl_cipher"]
	return &dbi.TLSStatus{
		Available: true,
		Encrypted: cipher != "",
		Version:   values["ssl_version"],
		Cipher:    cipher,
	}, nil
}
