package mysql

// mysql 时间点恢复（PITR）前提探测。
//
// 这里只回答事实：binlog 开没开、能看到哪些文件、保留多久。
// **不做恢复动作**：真正的 PITR 需要拿到 binlog 文件本体并重放，那是运维侧动作；
// 平台能把「现在能不能做、能回溯到哪」说清，就已经避免了「以为有备份就能恢复」的误判。

import (
	"context"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

var _ dbi.RecoveryInspector = (*MysqlMetadata)(nil)

// RecoveryReadiness 读取 binlog 开关、文件清单与过期策略
func (md *MysqlMetadata) RecoveryReadiness(ctx context.Context) (*dbi.RecoveryReadiness, error) {
	result := &dbi.RecoveryReadiness{Mode: "binlog"}

	_, variables, err := md.di.QueryContext(ctx, "SHOW VARIABLES LIKE 'log_bin'")
	if err != nil {
		return nil, dbi.NewPerfUnavailable("recovery", strings.TrimSpace(err.Error()))
	}
	logBin := ""
	if len(variables) > 0 {
		logBin = strings.ToUpper(cast.ToString(variables[0]["Value"]))
	}
	if logBin != "ON" {
		result.Prepared = false
		result.Mode = "none"
		result.Reason = "binlog 未开启（log_bin=" + firstNonEmpty(logBin, "OFF") + "），平台只能恢复到逻辑备份时刻"
		return result, nil
	}

	// 过期策略：决定「能回溯多远」的上限
	if _, rows, expireErr := md.di.QueryContext(ctx, "SHOW VARIABLES LIKE 'binlog_expire_logs_seconds'"); expireErr == nil && len(rows) > 0 {
		seconds := cast.ToInt64(cast.ToString(rows[0]["Value"]))
		if seconds > 0 {
			result.Retention = cast.ToString(seconds/3600) + " 小时"
		}
	}

	logs, logsErr := md.listBinlogFiles(ctx)
	if logsErr != nil {
		return nil, logsErr
	}
	if len(logs) == 0 {
		result.Prepared = false
		result.Reason = "binlog 已开启但看不到任何 binlog 文件（当前账号可能缺少 REPLICATION CLIENT 权限）"
		return result, nil
	}
	result.OldestPoint = logs[0]

	// 当前位置：8.4 起 SHOW MASTER STATUS 改名为 SHOW BINARY LOG STATUS，两个都试
	position := ""
	for _, sql := range []string{"SHOW BINARY LOG STATUS", "SHOW MASTER STATUS"} {
		if _, rows, posErr := md.di.QueryContext(ctx, sql); posErr == nil && len(rows) > 0 {
			position = cast.ToString(rows[0]["File"]) + ":" + cast.ToString(rows[0]["Position"])
			break
		}
	}
	result.LatestPoint = firstNonEmpty(position, logs[len(logs)-1])
	result.Prepared = true
	return result, nil
}

// listBinlogFiles 返回 binlog 文件名清单（旧 → 新）
func (md *MysqlMetadata) listBinlogFiles(ctx context.Context) ([]string, error) {
	_, rows, err := md.di.QueryContext(ctx, "SHOW BINARY LOGS")
	if err != nil {
		return nil, dbi.NewPerfUnavailable("recovery", strings.TrimSpace(err.Error()))
	}
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		if name := cast.ToString(row["Log_name"]); name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
