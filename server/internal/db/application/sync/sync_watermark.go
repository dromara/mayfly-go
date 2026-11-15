package sync

import (
	"cmp"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"

	"github.com/spf13/cast"
)

// 同步水位线推进与主键采集辅助。

// collectKeyStrings 执行查询 SQL 并将每行的多列值拼接为单个字符串切片。
// 返回底层查询错误：调用方据此区分「查询失败」与「结果为空」，避免跨方言等场景下
// SQL 执行失败被静默当成空结果、导致上层 checksum 校验假通过。
func collectKeyStrings(conn *dbi.DbConn, ctx context.Context, sql string) ([]string, error) {
	var keys []string
	if _, err := conn.WalkQueryRows(ctx, sql, func(row map[string]any, columns []*dbi.QueryColumn) error {
		keys = append(keys, encodeValidationRow(row, columns))
		return nil
	}); err != nil {
		return nil, err
	}
	return keys, nil
}

// encodeValidationRow 以长度前缀区分列边界，NULL 单独编码，避免分隔符和文本碰撞。
func encodeValidationRow(row map[string]any, columns []*dbi.QueryColumn) string {
	var encoded strings.Builder
	for _, column := range columns {
		value := row[column.Key]
		if value == nil {
			encoded.WriteString("N;")
			continue
		}
		text := fmt.Sprint(value)
		encoded.WriteString(strconv.Itoa(len(text)))
		encoded.WriteByte(':')
		encoded.WriteString(text)
	}
	return encoded.String()
}

// paginateTopN 用连接所属方言的分页改写能力，将不含分页子句的 baseSQL 改写为取前 n 行。
// 优先方言专用 rewriter（Oracle/DM/MSSQL 语法各异），无则回退默认 LIMIT/OFFSET（MySQL/PG/SQLite/ClickHouse 通用）。
// 源/目标可能异构，跨方言执行路径禁止直接拼 `LIMIT n`，否则不支持该语法的方言会整条 SQL 报错。
func paginateTopN(conn *dbi.DbConn, baseSQL string, n int64) (string, error) {
	// 分页改写是必需能力；历史上这里回退默认 LIMIT 语法，在 Oracle/DM/MSSQL 上会直接报错
	return conn.GetDialect().GetSQLParser().RewritePagination(baseSQL, 0, n)
}

// advanceSyncWatermark 把本批**所有行**的增量字段值并入「本轮最大值」，并据此刷新内存水位。
//
// 为什么逐行取最大而不是只看末行：
//   - 不分片时源查询带 `order by UpdField asc`，末行即最大，两种算法等价；
//   - 开启分片扫描后，分片查询按**分片键**排序（片内续扫的充分条件，见 shardScanOrder），
//     增量字段与排序键不同序：末行完全可能是中间值，只看末行会把水位停在过去
//     （下一轮大量重放，Append 模式下就是重复行）；
//   - 即使某片内末行恰好最大，片与片之间仍是交错。
//
// 仍不落库：写库只发生在整轮成功后（见 doDataSync 末尾），此处只维护内存值。
func advanceSyncWatermark(srcRes []map[string]any, sec *syncExecContext) error {
	if len(srcRes) == 0 {
		return nil
	}
	task := sec.task
	waterField := cmp.Or(task.UpdFieldSrc, sec.updFieldName)
	if waterField == "" {
		return nil
	}
	found := false
	seenNull := false
	var maxVal any
	for _, row := range srcRes {
		val, ok := lookupRowValue(row, waterField)
		if !ok {
			continue
		}
		if val == nil {
			seenNull = true
			continue
		}
		if !found || watermarkAhead(val, maxVal) {
			maxVal = val
			found = true
		}
	}
	if !found {
		// 本批没有任何可用的增量值：保持水位不动，并给出可检索的原因（不静默）
		if seenNull {
			return fmt.Errorf("update field [%s] is NULL in every row of the synced batch, keep watermark [%s] unchanged", waterField, task.UpdFieldVal)
		}
		return fmt.Errorf("update field [%s] not found in the synced batch, keep watermark [%s] unchanged", waterField, task.UpdFieldVal)
	}
	if sec.watermarkVal != nil && !watermarkAhead(maxVal, sec.watermarkVal) {
		// 本轮已有更靠后的值（分片间增量值交错时的正常情形），水位保持不动
		return nil
	}
	sec.watermarkVal = maxVal
	task.UpdFieldVal = cast.ToString(maxVal)
	return nil
}

// watermarkAhead 判断候选水位是否比已记录的水位更靠后。
//
// time.Time 必须按时间点比：Go 的 time 字符串布局带时区后缀，跳时区/夏令时下字典序不再等于
// 时间序，而判反方向的代价是「水位推得过分 → 中间那段行被永久跳过」，属于不可静默的数据丢失。
// 其余形态（整型/decimal/日期串）沿用过滤条件同一个比较器 compareOrdered，避免同一模块里两套大小判据。
// 类型对不上（同列不应出现）一律保守返 false：宁可下轮重放，不可漏数据。
func watermarkAhead(candidate, current any) bool {
	ct, ctOk := current.(time.Time)
	kt, ktOk := candidate.(time.Time)
	if ctOk || ktOk {
		if !ctOk || !ktOk {
			return false
		}
		return kt.After(ct)
	}
	return compareOrdered(cast.ToString(candidate), ">", cast.ToString(current))
}

// lookupRowValue 从查询行中取列值
func lookupRowValue(row map[string]any, column string) (any, bool) {
	if column == "" {
		return nil, false
	}
	if val, ok := row[column]; ok {
		return val, true
	}
	for name, val := range row {
		if strings.EqualFold(name, column) {
			return val, true
		}
	}
	return nil, false
}

func (app *DataSyncAppImpl) persistUpdFieldVal(ctx context.Context, task *entity.DataSyncTask) error {
	if task.UpdField == "" {
		return nil
	}
	ut := &entity.DataSyncTask{Id: task.Id, UpdFieldVal: task.UpdFieldVal}
	if err := app.UpdateById(ctx, ut); err != nil {
		return fmt.Errorf("failed to persist the data sync watermark of task [%d]: %s", task.Id, err.Error())
	}
	return nil
}
