package mask

// 导出链路的行级脱敏钩子。
//
// 存在的原因是：交互式查询的脱敏入口建立在「解析后的 SELECT 语句 + 结果列」上（要还原列归属哪张表、
// 是否被别名重命名），而 dump/导出侧是按表逐张 `SELECT * FROM table` 游标遍历，没有语句血缘可解。
// 因此这里提供一个以「表名 + 列清单」为键的构建器，判定口径（是否启用、账号豁免、fail-close）
// 全部交给 MaskEngine 单点给出，本文件不重复实现任何一条判定。

import (
	"context"
	"sync"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/export"
	masksvc "mayfly-go/internal/db/domain/mask"
)

// DumpRowMasker 按表懒构建并缓存行脱敏器。
//
// 缓存值允许为 nil，表示「该表已判定过且无需脱敏」：若不缓存，每行都会重新解析一遍规则，
// 大表导出会把这些开销摊到每一行上
type DumpRowMasker struct {
	engine MaskEngine
	dbConn *dbi.DbConn
	dbName string

	mu      sync.Mutex
	maskers map[string]*masksvc.RowMasker
}

func NewDumpRowMasker(engine MaskEngine, dbConn *dbi.DbConn, dbName string) *DumpRowMasker {
	return &DumpRowMasker{engine: engine, dbConn: dbConn, dbName: dbName, maskers: make(map[string]*masksvc.RowMasker)}
}

// fn 返回可直接挂到导出器上的钩子；引擎为空时返回 nil，等价于不脱敏。
func (d *DumpRowMasker) Fn() export.RowMaskFn {
	if d.engine == nil {
		return nil
	}
	return func(ctx context.Context, tableName string, columns []string, row map[string]any) error {
		m, err := d.maskerFor(ctx, tableName, columns)
		if err != nil || m == nil {
			return err
		}
		m.MaskRow(row)
		return nil
	}
}

// maskerFor 取该表的行脱敏器，必要时构建。
//
// 返回 error 表示脱敏计划不可用（fail-close），调用方必须中止整次导出：
// 半途透出未脱敏数据比不出文件更糟，因此这里不缓存失败结果、也不降级为「不脱敏继续导」
func (d *DumpRowMasker) maskerFor(ctx context.Context, tableName string, columns []string) (*masksvc.RowMasker, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if m, ok := d.maskers[tableName]; ok {
		return m, nil
	}
	m, err := d.engine.BuildTableColumnMasker(ctx, d.dbConn, d.dbName, tableName, columns)
	if err != nil {
		return nil, err
	}
	d.maskers[tableName] = m
	return m, nil
}
