package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/logx"
	"strings"

	"github.com/spf13/cast"
)

// doDataValidation 数据校验模式。
// 对比源/目标行数，并可选抽样 checksum 对比（按字段映射的主键列）。
func (app *DataSyncAppImpl) doDataValidation(ctx context.Context, task *entity.DataSyncTask, srcConn, targetConn *dbi.DbConn, syncLog *entity.DataSyncLog, metrics *SyncMetrics) {
	app.appendRunLog(syncLog, "开始数据校验模式: 任务 [%s]", task.TaskName)
	app.appendRunLog(syncLog, "源库: %s，目标库: %s", task.SrcDbName, task.TargetDbName)

	// 校验查询与同步查询(buildSyncQuery)一致：去除 DataSQL 尾部空白与分号，
	// 否则作为子查询包裹 `SELECT ... FROM (DataSQL) _src` 时尾分号会导致整条 SQL 语法错误
	dataSQL := strings.TrimRight(strings.TrimSpace(task.DataSQL), ";")

	// 对比全部映射字段的原值；无有效映射时不能声称内容校验通过。
	var keyColumns []string
	var srcKeyCols []string
	var fieldMap []map[string]string
	mapErr := json.Unmarshal([]byte(task.FieldMap), &fieldMap)
	if mapErr != nil || len(fieldMap) == 0 {
		syncLog.Status = entity.DataSyncTaskStateFail
		syncLog.ErrText = "validation requires a valid, non-empty field mapping"
		return
	}
	srcQuote := srcConn.GetDialect().Quoter().QuoteIdent
	targetQuote := targetConn.GetDialect().Quoter().QuoteIdent
	for _, fm := range fieldMap {
		if fm["src"] == "" || fm["target"] == "" {
			syncLog.Status = entity.DataSyncTaskStateFail
			syncLog.ErrText = "validation field mapping contains an empty source or target column"
			return
		}
		keyColumns = append(keyColumns, targetQuote(fm["target"]))
		srcKeyCols = append(srcKeyCols, srcQuote(fm["src"]))
	}
	targetTable := quoteTargetTableRef(targetConn, task.TargetTableName)

	// 源数据行数
	srcCount := 0
	if _, srcErr := srcConn.WalkQueryRows(ctx, dataSQL, func(row map[string]any, columns []*dbi.QueryColumn) error {
		srcCount++
		return nil
	}); srcErr != nil {
		syncLog.Status = entity.DataSyncTaskStateFail
		syncLog.ErrText = fmt.Sprintf("validation failed: source count query error: %s", srcErr.Error())
		app.appendRunLog(syncLog, "校验失败: 源库行数统计查询错误: %s", srcErr.Error())
		return
	}

	// 目标数据行数
	targetCountSQL := fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", targetTable)
	targetCount := 0
	if _, tgtErr := targetConn.WalkQueryRows(ctx, targetCountSQL, func(row map[string]any, columns []*dbi.QueryColumn) error {
		if cnt, ok := row["cnt"]; ok {
			targetCount = cast.ToInt(cnt)
		}
		return dbi.NewStopWalkQueryError("count done")
	}); tgtErr != nil {
		syncLog.Status = entity.DataSyncTaskStateFail
		syncLog.ErrText = fmt.Sprintf("validation failed: target count query error: %s", tgtErr.Error())
		app.appendRunLog(syncLog, "校验失败: 目标库行数统计查询错误: %s", tgtErr.Error())
		return
	}

	// 行数对比
	if srcCount != targetCount {
		syncLog.Status = entity.DataSyncTaskStateFail
		syncLog.ErrText = fmt.Sprintf("validation failed: source=%d, target=%d, diff=%d", srcCount, targetCount, srcCount-targetCount)
		syncLog.ResNum = srcCount
		metrics.TotalRows = srcCount
		app.appendRunLog(syncLog, "校验失败: 源=%d行，目标=%d行，差异=%d行", srcCount, targetCount, srcCount-targetCount)
		return
	}
	app.appendRunLog(syncLog, "行数校验通过: 源=%d行，目标=%d行", srcCount, targetCount)

	// 行数一致时，对比排序后的前 100 行映射字段原值。
	//
	// 配了实际转换规则时跳过内容比对：目标写入的列值是转换后的结果（upper、常量、表达式等），
	// 拿源原值与目标转换值直接比对会把“转换成功”误判为“内容不一致”，与同步写入语义相背。
	// 行数比对不依赖列内容，保留即可；内容比对需同层转换能力（目前不具备），宁缺勿误。
	//
	// 与 doDataSync 共享同一个构造入口 NewTransformEngine：“” / 非法 JSON / 空数组都归为无转换
	// （engine==nil）。非法 JSON 先失败返回，不默默“无转换”跟实际执行链路语义分叉。
	transformEngine, terr := NewTransformEngine(task.TransformRules)
	if terr != nil {
		syncLog.Status = entity.DataSyncTaskStateFail
		syncLog.ErrText = fmt.Sprintf("validation failed: invalid TransformRules: %s", terr.Error())
		app.appendRunLog(syncLog, "校验失败: 字段转换规则解析错误: %s", terr.Error())
		return
	}
	if len(keyColumns) > 0 && srcCount > 0 && transformEngine != nil {
		app.appendRunLog(syncLog, "检测到字段转换规则，跳过内容比对（源原值与目标转换值天然不等），仅保留行数校验")
		logx.InfofContext(ctx, "data validation: skipping content check because TransformRules is configured (task=%d)", task.Id)
	} else if len(keyColumns) > 0 && srcCount > 0 {
		failValidation := func(errText, runMsg string) {
			syncLog.Status = entity.DataSyncTaskStateFail
			syncLog.ErrText = errText
			syncLog.ResNum = srcCount
			metrics.TotalRows = srcCount
			app.appendRunLog(syncLog, "%s", runMsg)
		}

		const checkSampleRows = 100
		// 两端分别排序抽样；不同排序规则可能选中不同样本，不能据此证明全量一致。
		// 先构造不含分页子句的基础 SQL，再按各自方言改写：源/目标可能异构，Oracle/DM/MSSQL 不识别裸 LIMIT。
		srcBaseCheckSQL := fmt.Sprintf("SELECT %s FROM (%s) _src ORDER BY %s",
			strings.Join(srcKeyCols, ", "), dataSQL, strings.Join(srcKeyCols, ", "))
		targetBaseCheckSQL := fmt.Sprintf("SELECT %s FROM %s ORDER BY %s",
			strings.Join(keyColumns, ", "), targetTable, strings.Join(keyColumns, ", "))

		srcCheckSQL, pgErr := paginateTopN(srcConn, srcBaseCheckSQL, checkSampleRows)
		if pgErr != nil {
			failValidation(fmt.Sprintf("validation failed: build source checksum sql error: %s", pgErr.Error()),
				fmt.Sprintf("校验失败: 源端checksum SQL构建错误: %s", pgErr.Error()))
			return
		}
		targetCheckSQL, pgErr := paginateTopN(targetConn, targetBaseCheckSQL, checkSampleRows)
		if pgErr != nil {
			failValidation(fmt.Sprintf("validation failed: build target checksum sql error: %s", pgErr.Error()),
				fmt.Sprintf("校验失败: 目标端checksum SQL构建错误: %s", pgErr.Error()))
			return
		}

		// checksum 查询失败必须显式判失败：否则空结果会让两侧「相等」而假通过，掩盖跨方言执行错误
		srcKeys, srcCheckErr := collectKeyStrings(srcConn, ctx, srcCheckSQL)
		if srcCheckErr != nil {
			failValidation(fmt.Sprintf("validation failed: source checksum query error: %s", srcCheckErr.Error()),
				fmt.Sprintf("校验失败: 源端checksum查询错误: %s", srcCheckErr.Error()))
			return
		}
		targetKeys, tgtCheckErr := collectKeyStrings(targetConn, ctx, targetCheckSQL)
		if tgtCheckErr != nil {
			failValidation(fmt.Sprintf("validation failed: target checksum query error: %s", tgtCheckErr.Error()),
				fmt.Sprintf("校验失败: 目标端checksum查询错误: %s", tgtCheckErr.Error()))
			return
		}

		if len(srcKeys) != len(targetKeys) {
			failValidation(fmt.Sprintf("validation passed row count (%d) but checksum mismatch: source keys=%d, target keys=%d",
				srcCount, len(srcKeys), len(targetKeys)),
				fmt.Sprintf("校验失败: 行数一致但checksum数量不匹配(源%d个，目标%d个)", len(srcKeys), len(targetKeys)))
			return
		}
		// 逐行比较带列边界的映射值编码
		mismatch := false
		for i := range srcKeys {
			if srcKeys[i] != targetKeys[i] {
				mismatch = true
				break
			}
		}
		if mismatch {
			failValidation(fmt.Sprintf("validation passed row count (%d) but checksum content mismatch (first %d rows)", srcCount, checkSampleRows),
				fmt.Sprintf("校验失败: 行数一致但checksum内容不匹配(前%d行)", checkSampleRows))
			return
		}
		logx.InfofContext(ctx, "data validation: row count=%d, checksum passed (first %d rows match)", srcCount, checkSampleRows)
		app.appendRunLog(syncLog, "抽样校验通过: 前%d行映射字段原值匹配，不代表全部数据一致", len(srcKeys))
	}

	syncLog.Status = entity.DataSyncTaskStateSuccess
	syncLog.ErrText = fmt.Sprintf("validation passed: source=%d, target=%d", srcCount, targetCount)
	syncLog.ResNum = srcCount
	metrics.TotalRows = srcCount
	app.appendRunLog(syncLog, "数据校验完成: 源=%d行，目标=%d行", srcCount, targetCount)
}
