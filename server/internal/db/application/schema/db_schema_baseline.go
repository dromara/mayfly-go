package schema

// 表结构漂移基线：建立/刷新基线、比对漂移、定时巡检并通知基线创建者。
//
// 采集一律走结构对比的同一个 Snapshot 入口（含同一套表数上限与截断说明），
// 指纹与差异判定共用同一份取值口径（见 SummarizeTable 的注释）：
// 两处各判一次就会出现「对比说有差异、漂移说没变」这种用户无法自证的矛盾。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/imsg"
	msgdto "mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/pkg/event"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/taskx"

	"github.com/spf13/cast"
)

// BaselineDriftDetailMax 漂移明细的存储长度上限（与实体列宽一致），超出即截断并标注
const BaselineDriftDetailMax = 900

// SchemaBaseline 表结构基线用例
type SchemaBaseline interface {
	// List 返回某个库的基线行
	List(ctx context.Context, dbId uint64, dbName string) ([]*entity.DbSchemaBaseline, error)
	// Save 建立或刷新基线：以当前结构为准，覆盖旧基线并清掉已不存在的表
	Save(ctx context.Context, dbId uint64, dbName string, tables []string) (*BaselineSaveResult, error)
	// Delete 删除某个库的基线
	Delete(ctx context.Context, dbId uint64, dbName string) error
	// Check 比对当前结构与基线，写回漂移状态；notify 为真时把新发生的漂移推给基线创建者
	Check(ctx context.Context, dbId uint64, dbName string, notify bool) (*DriftResult, error)
	// ListByDb 读取某库的基线行（告警评估器统计漂移表数用）
	ListByDb(dbId uint64, dbName string) ([]*entity.DbSchemaBaseline, error)
	// CheckAll 巡检所有已建立基线的库（定时任务入口）
	CheckAll(ctx context.Context)
}

// BaselineSaveResult 建立基线的结果
type BaselineSaveResult struct {
	Tables int    `json:"tables"`
	Note   string `json:"note,omitempty"`
}

// DriftedTable 一张漂移的表
type DriftedTable struct {
	Table  string `json:"table"`
	Detail string `json:"detail"`
}

// DriftResult 一次漂移比对的结果
type DriftResult struct {
	DbId          uint64         `json:"dbId"`
	Db            string         `json:"db"`
	CheckedAt     *time.Time     `json:"checkedAt"`
	DriftedTables []DriftedTable `json:"driftedTables"`
	// Note 基线覆盖范围受限之类的说明，没有它「无漂移」会被读成「整个库都没变」
	Note string `json:"note,omitempty"`
	// ReceiverId 基线创建者，新发生漂移时的通知对象
	ReceiverId uint64 `json:"-"`
}

// DbGetter 基线用例依赖的窄接口：按主键取库实体（建基线时补 InstanceId 用）。
//
// 由消费者定义（Go 惯例），主包的 dbAppImpl 天然满足，避免子包反向 import 主包造成循环。
type DbGetter interface {
	GetById(id uint64, cols ...string) (*entity.Db, error)
}

type SchemaBaselineAppImpl struct {
	dbApp        DbGetter                  `inject:"T"`
	baselineRepo repository.SchemaBaseline `inject:"T"`
	schemaDiff   SchemaDiff                `inject:"T"`
}

func GetSchemaBaselineApp() SchemaBaseline {
	return ioc.Get[SchemaBaseline]()
}

// InitDriftCron 注册漂移巡检。
//
// 周期取 30 分钟：比结构变更的常见发现周期（下次发布、下次对账）快得多，
// 又不至于让一次巡检变成对全部受管库的持续内省压力。
// 用带分布式锁的绑定：多实例部署时只有一个实例执行巡检，避免每实例各扫一遍并重复通知基线创建者
// （锁在正常运行结束后随 defer 释放，TTL 只用于崩溃后自愈，故取得略小于周期）。
func InitDriftCron() {
	if err := taskx.BindCronTaskWithLock("db-schema-drift", "@every 30m", 25*time.Minute, true, func() {
		defer gox.Recover()
		// 每轮新建带 traceId 的任务级 ctx：若注册期取一次长期复用，30 分钟一轮的巡检会全共用同一 traceId，失去串联意义
		GetSchemaBaselineApp().CheckAll(contextx.WithTraceId(context.Background()))
	}); err != nil {
		logx.Errorf("failed to bind schema drift check task: %v", err)
	}
}

// ListByDb 供告警评估器读取某库的基线行
func (s *SchemaBaselineAppImpl) ListByDb(dbId uint64, dbName string) ([]*entity.DbSchemaBaseline, error) {
	return s.baselineRepo.ListByDb(dbId, dbName)
}

func (s *SchemaBaselineAppImpl) List(ctx context.Context, dbId uint64, dbName string) ([]*entity.DbSchemaBaseline, error) {
	return s.baselineRepo.ListByDb(dbId, strings.TrimSpace(dbName))
}

func (s *SchemaBaselineAppImpl) Save(ctx context.Context, dbId uint64, dbName string, tables []string) (*BaselineSaveResult, error) {
	dbName = strings.TrimSpace(dbName)
	snapshots, note, err := s.schemaDiff.Snapshot(ctx, dbId, dbName, tables)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrSchemaBaselineEmpty)
	}

	db, err := s.dbApp.GetById(dbId)
	if err != nil {
		return nil, errorx.NewBiz("database not found")
	}

	now := time.Now()
	existing := s.existingByTable(ctx, dbId, dbName)
	for _, snapshot := range snapshots {
		summary := SummarizeTable(snapshot)
		table := strings.ToLower(snapshot.Name)
		row, found := existing[table]
		if !found {
			row = &entity.DbSchemaBaseline{DbId: dbId, Db: dbName, Table: table}
			row.CreateTime = &now
			if account := contextx.GetLoginAccount(ctx); account != nil {
				row.CreatorId = cast.ToUint64(account.Id)
			}
		}
		row.InstanceId = db.InstanceId
		row.Fingerprint = summary.Fingerprint
		row.ColumnCount = summary.ColumnCount
		row.PrimaryKeys = summary.PrimaryKeys
		row.IndexCount = summary.IndexCount
		// 刷新基线即「以当前结构为准」，旧的漂移状态必须清掉：留着就会一边显示已刷新、一边还在报漂移
		row.Drifted = false
		row.DriftDetail = ""
		row.CheckedAt = &now
		if err = s.baselineRepo.Save(ctx, row); err != nil {
			return nil, err
		}
		delete(existing, table)
	}

	// 基线里有、这次快照里没有的表：说明它已被删除，留在表里会让下次比对把「表不存在」误当成新变化
	for _, stale := range existing {
		if err = s.baselineRepo.DeleteById(ctx, stale.Id); err != nil {
			return nil, err
		}
	}

	return &BaselineSaveResult{Tables: len(snapshots), Note: note}, nil
}

func (s *SchemaBaselineAppImpl) Delete(ctx context.Context, dbId uint64, dbName string) error {
	return s.baselineRepo.DeleteByDb(ctx, dbId, strings.TrimSpace(dbName))
}

func (s *SchemaBaselineAppImpl) Check(ctx context.Context, dbId uint64, dbName string, notify bool) (*DriftResult, error) {
	dbName = strings.TrimSpace(dbName)
	baselines, err := s.baselineRepo.ListByDb(dbId, dbName)
	if err != nil {
		return nil, err
	}
	if len(baselines) == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrSchemaBaselineEmpty)
	}

	snapshots, note, err := s.schemaDiff.Snapshot(ctx, dbId, dbName, nil)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	result := &DriftResult{DbId: dbId, Db: dbName, CheckedAt: &now, Note: note, ReceiverId: baselines[0].CreatorId}
	byTable := make(map[string]*entity.DbSchemaBaseline, len(baselines))
	for _, baseline := range baselines {
		byTable[baseline.Table] = baseline
	}

	receiverId := baselines[0].CreatorId
	newlyDrifted := 0
	for _, snapshot := range snapshots {
		table := strings.ToLower(snapshot.Name)
		baseline := byTable[table]
		if baseline == nil {
			// 基线里没有这张表：建基线之后新增的表，同样是漂移的一种，不能因为「没得比」就跳过
			baseline = &entity.DbSchemaBaseline{DbId: dbId, Db: dbName, Table: table, InstanceId: baselines[0].InstanceId}
			baseline.CreateTime = &now
			baseline.CreatorId = receiverId
		}
		summary := SummarizeTable(snapshot)
		drifted, detail := s.evaluateDrift(ctx, baseline, summary)
		if drifted && !baseline.Drifted {
			newlyDrifted++
		}
		baseline.Drifted = drifted
		baseline.DriftDetail = detail
		baseline.CheckedAt = &now
		if err = s.baselineRepo.Save(ctx, baseline); err != nil {
			return nil, err
		}
		delete(byTable, table)
		if drifted {
			result.DriftedTables = append(result.DriftedTables, DriftedTable{Table: snapshot.Name, Detail: detail})
		}
	}

	// 基线里有、当前库已经没有的表
	for _, missing := range byTable {
		// 先取上一轮状态再改写：写完 Drifted=true 再判「是否已报过」永远为真，
		// 新发生的漂移就一条也不会推出去
		hadDrift := missing.Drifted
		missing.Drifted = true
		missing.DriftDetail = i18n.TC(ctx, imsg.SchemaDriftTableRemoved)
		if !hadDrift {
			newlyDrifted++
		}
		missing.CheckedAt = &now
		if err = s.baselineRepo.Save(ctx, missing); err != nil {
			return nil, err
		}
		result.DriftedTables = append(result.DriftedTables, DriftedTable{Table: missing.Table, Detail: missing.DriftDetail})
	}

	if notify && newlyDrifted > 0 {
		s.notifyDrift(ctx, dbId, dbName, result)
	}
	return result, nil
}

// evaluateDrift 判断单表是否漂移并给出明细。
// 指纹一致时清掉明细，避免旧文案在结构已回到基线后继续显示
func (s *SchemaBaselineAppImpl) evaluateDrift(ctx context.Context, baseline *entity.DbSchemaBaseline, current TableSummary) (bool, string) {
	if baseline.Fingerprint == current.Fingerprint {
		return false, ""
	}
	if baseline.Fingerprint == "" {
		return true, i18n.TC(ctx, imsg.SchemaDriftTableAdded)
	}
	stored := TableSummary{
		Fingerprint:          baseline.Fingerprint,
		ColumnCount:          baseline.ColumnCount,
		PrimaryKeys:          baseline.PrimaryKeys,
		PrimaryKeysAvailable: true,
		IndexCount:           baseline.IndexCount,
		IndexesAvailable:     true,
	}
	return true, truncateDetail(renderDriftChanges(ctx, DriftChanges(stored, current)))
}

func (s *SchemaBaselineAppImpl) CheckAll(ctx context.Context) {
	keys, err := s.baselineRepo.ListDbKeys()
	if err != nil {
		logx.ErrorfContext(ctx, "schema drift cron: failed to list baselines: %s", err.Error())
		return
	}
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		idKey := fmt.Sprintf("%d|%s", key.DbId, key.Db)
		if seen[idKey] {
			continue
		}
		seen[idKey] = true
		if _, driftErr := s.Check(ctx, key.DbId, key.Db, true); driftErr != nil {
			// 单个库连不上/权限变化不该拖垮整轮巡检，但必须留下日志：静默跳过等于这一轮没跑
			logx.WarnfContext(ctx, "schema drift cron: failed to check db [%d/%s]: %s", key.DbId, key.Db, driftErr.Error())
		}
	}
}

// notifyDrift 推送漂移通知。只推「本轮新发生的漂移」：每轮都推同一批就会把通知变成噪声，
// 而噪声的下场是被人关掉整个通知类型
func (s *SchemaBaselineAppImpl) notifyDrift(ctx context.Context, dbId uint64, dbName string, result *DriftResult) {
	if result.ReceiverId == 0 || len(result.DriftedTables) == 0 {
		return
	}
	lines := make([]string, 0, len(result.DriftedTables))
	for _, drifted := range result.DriftedTables {
		lines = append(lines, fmt.Sprintf("%s: %s", drifted.Table, drifted.Detail))
	}
	global.EventBus.Publish(ctx, event.EventTopicMsgTmplSend,
		msgdto.DbSchemaDriftEvent(dbId, dbName, strings.Join(lines, "; "), len(result.DriftedTables), result.ReceiverId))
}

// existingByTable 取该库现有基线行，按表名索引；读取失败时返回空表（当作全新建立）
func (s *SchemaBaselineAppImpl) existingByTable(ctx context.Context, dbId uint64, dbName string) map[string]*entity.DbSchemaBaseline {
	rows, err := s.baselineRepo.ListByDb(dbId, dbName)
	if err != nil {
		logx.WarnfContext(ctx, "schema baseline: failed to read existing rows of [%d/%s]: %s", dbId, dbName, err.Error())
		return map[string]*entity.DbSchemaBaseline{}
	}
	indexed := make(map[string]*entity.DbSchemaBaseline, len(rows))
	for _, row := range rows {
		indexed[row.Table] = row
	}
	return indexed
}

// renderDriftChanges 把变化种类渲染成人读明细
func renderDriftChanges(ctx context.Context, changes []DriftChange) string {
	parts := make([]string, 0, len(changes))
	for _, change := range changes {
		switch change.Kind {
		case DriftKindPrimaryKeys:
			parts = append(parts, i18n.TC(ctx, imsg.SchemaDriftPrimaryKeys,
				"from", orNoPrimaryKey(ctx, change.From), "to", orNoPrimaryKey(ctx, change.To)))
		case DriftKindColumnCount:
			parts = append(parts, i18n.TC(ctx, imsg.SchemaDriftColumnCount, "from", change.From, "to", change.To))
		case DriftKindIndexCount:
			parts = append(parts, i18n.TC(ctx, imsg.SchemaDriftIndexCount, "from", change.From, "to", change.To))
		default:
			parts = append(parts, i18n.TC(ctx, imsg.SchemaDriftColumnDefs))
		}
	}
	return strings.Join(parts, "; ")
}

func orNoPrimaryKey(ctx context.Context, keys string) string {
	if strings.TrimSpace(keys) == "" {
		return i18n.TC(ctx, imsg.SchemaDriftNoPk)
	}
	return keys
}

// truncateDetail 明细按列宽截断并标注：整句丢掉比截断更糟（用户会以为没有变化）
func truncateDetail(detail string) string {
	runes := []rune(detail)
	if len(runes) <= BaselineDriftDetailMax {
		return detail
	}
	return string(runes[:BaselineDriftDetailMax-1]) + "…"
}
