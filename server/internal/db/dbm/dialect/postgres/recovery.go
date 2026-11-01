package postgres

// postgres 时间点恢复（PITR）前提探测。
//
// 只看 pg_stat_archiver 与 archive_mode：能归档才有「恢复到归档之前的任意时刻」。
// 需要注意 pg_stat_archiver 是**实例级**统计，不含「归档到哪去了」——归档目的地
// 属于部署配置（archive_command），平台据此只能判断「有没有在推进」，不猜测路径。

import (
	"context"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/spf13/cast"
)

var _ dbi.RecoveryInspector = (*PgsqlMetadata)(nil)

// RecoveryReadiness 读取 archive_mode 与归档器统计
func (md *PgsqlMetadata) RecoveryReadiness(ctx context.Context) (*dbi.RecoveryReadiness, error) {
	result := &dbi.RecoveryReadiness{Mode: "wal_archive"}

	_, modeRows, err := md.di.QueryContext(ctx, "SHOW archive_mode")
	if err != nil {
		return nil, dbi.NewPerfUnavailable("recovery", strings.TrimSpace(err.Error()))
	}
	archiveMode := ""
	if len(modeRows) > 0 {
		for _, value := range modeRows[0] {
			archiveMode = strings.ToLower(cast.ToString(value))
			break
		}
	}
	if archiveMode != "on" && archiveMode != "always" {
		result.Prepared = false
		result.Mode = "none"
		result.Reason = "WAL 归档未开启（archive_mode=" + firstNonEmpty(archiveMode, "off") + "），平台只能恢复到逻辑备份时刻"
		return result, nil
	}

	_, statRows, statErr := md.di.QueryContext(ctx,
		"SELECT archived_count, failed_count, COALESCE(last_archived_wal, '') last_wal,"+
			" COALESCE(last_archived_time::text, '') last_time FROM pg_stat_archiver")
	if statErr != nil {
		return nil, dbi.NewPerfUnavailable("recovery", strings.TrimSpace(statErr.Error()))
	}
	if len(statRows) == 0 {
		result.Prepared = false
		result.Reason = "读不到 pg_stat_archiver 统计，无法确认归档是否在推进"
		return result, nil
	}
	row := statRows[0]
	archived := cast.ToInt64(row["archived_count"])
	failed := cast.ToInt64(row["failed_count"])
	result.OldestPoint = cast.ToString(row["last_wal"])
	result.Retention = "已归档 " + cast.ToString(archived) + " 个"

	// 有失败且从未成功：归档链路是坏的，此时「开着」等于没开
	if archived == 0 && failed > 0 {
		result.Prepared = false
		result.Reason = "WAL 归档从未成功（失败 " + cast.ToString(failed) + " 次），需先修 archive_command"
		return result, nil
	}

	if _, lsnRows, lsnErr := md.di.QueryContext(ctx, "SELECT pg_current_wal_lsn()::text lsn"); lsnErr == nil && len(lsnRows) > 0 {
		result.LatestPoint = cast.ToString(lsnRows[0]["lsn"])
	}
	result.Prepared = true
	return result, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
