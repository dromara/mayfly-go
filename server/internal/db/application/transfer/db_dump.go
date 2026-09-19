package transfer

// 本文件为数据库dump导出入口：将源库表结构/数据导出为目标格式。
// 核心编排逻辑由 export.Exporter（dbm/export/exporter.go）统一处理，本文件仅负责参数适配。
// 供主包DbAppImpl的DumpDb功能与本包迁移/校验链路（集成测试直驱真实dump）共用，
// 与导入侧 ImportDumpStream 构成完整迁移引擎

import (
	"context"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/export"
)

// DumpDbScript dump核心逻辑：将dbConn指向的源库表结构/数据导出为目标格式。
//
// 源连接由参数注入而非内部按DbId获取，便于集成测试直接驱动真实dump链路（不经实例/权限体系），
// 也便于迁移等上层已在外部持有连接的场景复用
func DumpDbScript(ctx context.Context, dbConn *dbi.DbConn, reqParam *dto.DumpDb) error {
	// 确定导出格式（默认 SQL 保持向后兼容）
	format := "sql"
	if reqParam.ExportFormat != "" {
		format = reqParam.ExportFormat
	}

	// 获取导出消费者
	consumer := export.Get(format)
	if consumer == nil {
		return &export.FormatError{Format: format}
	}

	// 导出配置
	settings := reqParam.Settings
	if settings == nil {
		settings = export.DefaultSettings(format)
	}

	// 创建共享编排器
	exporter := export.NewExporter(dbConn, consumer, settings, reqParam.Writer).
		WithTables(reqParam.Tables).
		WithTableFilter(reqParam.TableFilter).
		WithDumpDDL(reqParam.DumpDDL).
		WithDumpData(reqParam.DumpData).
		WithTargetType(reqParam.TargetDbType).
		WithLog(reqParam.Log).
		WithProgress(reqParam.Progress)

	return exporter.Export(ctx)
}
