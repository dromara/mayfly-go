package sync

import (
	"context"
	"fmt"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
)

// 内联保留集合的资源上限；超限必须中止，不能拆成多个独立 NOT IN 删除。
// 更大规模的对账需要方言暂存表及反连接能力，不能默默截断保留集合。
const maxReconcileKeys = 100000
const maxReconcileSQLBytes = 16 << 20

func needsDeleteReconcile(mode entity.DataSyncMode) bool {
	return mode == entity.DataSyncModeIncrementalHardDel || mode == entity.DataSyncModeIncrementalSoftDel
}

func validateReconcileKeys(dbType dbi.DbType, meta *dbi.TargetTableMeta, columns []dbi.Column) error {
	if dbType == "clickhouse" {
		return fmt.Errorf("delete reconciliation requires synchronous deletion, unsupported for clickhouse")
	}
	if meta == nil || len(meta.UniqueColumns) == 0 {
		return fmt.Errorf("delete reconciliation requires a mapped primary key or a single non-null unique key")
	}
	for _, key := range meta.UniqueColumns {
		found := false
		for _, column := range columns {
			if column.ColumnName != key {
				continue
			}
			found = true
			if column.Nullable && !column.IsPrimaryKey {
				return fmt.Errorf("delete reconciliation key %q must not be nullable", key)
			}
			switch dbi.GetDbDataType(dbType, column.DataType).Category() {
			case dbi.TCBinary, dbi.TCVarbinary, dbi.TCBlob, dbi.TCMediumblob, dbi.TCLongblob:
				return fmt.Errorf("binary reconciliation key %q is not supported", key)
			}
		}
		if !found {
			return fmt.Errorf("delete reconciliation key %q is not mapped to an insertable target column", key)
		}
	}
	return nil
}

func (sec *syncExecContext) collectReconcileKeys(rows []map[string]any) error {
	if !needsDeleteReconcile(sec.task.SyncMode) {
		return nil
	}
	if len(sec.reconcileKeys)+len(rows) > maxReconcileKeys {
		return fmt.Errorf("delete reconciliation exceeds %d source keys; target deletion was not executed", maxReconcileKeys)
	}
	for _, row := range rows {
		keys := make([]any, len(sec.targetTableMeta.UniqueColumns))
		for i, name := range sec.targetTableMeta.UniqueColumns {
			v, ok := row[name]
			if !ok || v == nil {
				return fmt.Errorf("missing or NULL reconciliation key %q; target deletion was not executed", name)
			}
			keys[i] = v
		}
		sec.reconcileKeys = append(sec.reconcileKeys, keys)
	}
	return nil
}

// reconcileDeletes 仅在完整源扫描及全部写入成功后执行，任何失败均传播给任务收尾。
func (app *DataSyncAppImpl) reconcileDeletes(ctx context.Context, sec *syncExecContext, syncLog *entity.DataSyncLog) error {
	if !needsDeleteReconcile(sec.task.SyncMode) {
		return nil
	}
	conn, task := sec.targetConn, sec.task
	gen := conn.GetDialect().GetSQLGenerator()
	statements := gen.GenBatchDelete(task.TargetTableName, sec.targetTableMeta.UniqueColumns, sec.reconcileKeys, sec.targetTableMeta)
	if len(sec.reconcileKeys) == 0 {
		if conn.Info.Type == "clickhouse" {
			return fmt.Errorf("empty-source delete reconciliation is not supported for clickhouse")
		}
		statements = []string{"DELETE FROM " + quoteTargetTableRef(conn, task.TargetTableName)}
	}
	if len(statements) != 1 {
		return fmt.Errorf("delete reconciliation requires one complete retention predicate, got %d statements", len(statements))
	}
	if len(statements[0]) > maxReconcileSQLBytes {
		return fmt.Errorf("delete reconciliation SQL exceeds %d bytes; target deletion was not executed", maxReconcileSQLBytes)
	}
	affected, err := conn.ExecContext(ctx, statements[0])
	if err != nil {
		return fmt.Errorf("delete reconciliation of table %q failed: %w", task.TargetTableName, err)
	}
	sec.metrics.DeleteCount += int(affected)
	app.appendRunLog(syncLog, "删除对账完成: 实际删除 %d 行，保留键 %d 个", affected, len(sec.reconcileKeys))
	return nil
}
