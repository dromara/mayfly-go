package schema

// 结构对比的应用层：把「两侧库的内省结果」取成快照，交给 DiffSnapshots 比对。
//
// 拆成两层是为了让跨方言归一化规则能被单测覆盖（纯函数），而内省读取由真实数据库集成测试覆盖。
// 本文件只做取数与限额，不掺判定逻辑。

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
)

// DbConnReader 结构对比依赖的窄接口：按库取连接。
//
// 由消费者定义（Go 惯例），主包的 dbAppImpl 天然满足，避免子包反向 import 主包造成循环；
// 走这个入口而不是自己建连接，是为了复用连接层的资源权限校验——结构里含列名与注释，
// 本身就是要授权的库侧信息
type DbConnReader interface {
	GetDbConn(ctx context.Context, dbId uint64, dbName string) (*dbi.DbConn, error)
}

// DiffReq 一次结构对比的入参
type DiffReq struct {
	SrcDbId uint64
	SrcDb   string
	DstDbId uint64
	DstDb   string
	// Tables 限定比对的表名；为空表示比对两侧表名并集（在限额内）
	Tables []string
}

// SchemaDiff 结构对比用例入口
type SchemaDiff interface {
	// Diff 比对两侧库的表结构，只返回有差异或比不了的表
	Diff(ctx context.Context, req *DiffReq) (*Result, error)

	// Snapshot 取一个库的表结构快照（tables 为空则取该库全部表，截断时返回 note）。
	//
	// 漂移检测与基线指纹必须走这一个采集入口：另起一条内省路径，两边看到的
	// 「表有哪些、列是什么」就会不一致，漂移判定与对比结果就会互相矛盾
	Snapshot(ctx context.Context, dbId uint64, dbName string, tables []string) ([]TableSnapshot, string, error)
}

const (
	// maxAutoDiffTables 未指定表清单时自动比对的上限：
	// 每张表要读列、主键与索引，整库不限额就是上千次元数据查询，一个页面请求扛不住
	maxAutoDiffTables = 50
	// maxDiffTables 显式指定表清单时的上限，超出即要求收窄范围
	maxDiffTables = 200
	// maxBaselineTables 基线/快照一次覆盖的表数上限：基线是显式建立、后台定时重比的，
	// 比页面交互可以宽，但不能不限额（上千张表就是三千次内省查询，一个 cron 周期跑不完）
	maxBaselineTables = 500
)

// SchemaDiffAppImpl 结构对比用例的实现；dbApp 由 IOC 注入（主包 dbAppImpl 天然满足窄接口）
type SchemaDiffAppImpl struct {
	dbApp DbConnReader `inject:"T"`
}

var _ SchemaDiff = (*SchemaDiffAppImpl)(nil)

func GetSchemaDiffApp() SchemaDiff {
	return ioc.Get[SchemaDiff]()
}

func (s *SchemaDiffAppImpl) Diff(ctx context.Context, req *DiffReq) (*Result, error) {
	if req.SrcDbId == req.DstDbId && strings.TrimSpace(req.SrcDb) == strings.TrimSpace(req.DstDb) {
		// 两侧同一个库：比出自洽的「无差异」不是结论，只会让人以为对比真的发生过
		return nil, errorx.NewBizI(ctx, imsg.ErrSchemaDiffSameDb)
	}

	srcConn, err := s.dbApp.GetDbConn(ctx, req.SrcDbId, req.SrcDb)
	if err != nil {
		return nil, err
	}
	dstConn, err := s.dbApp.GetDbConn(ctx, req.DstDbId, req.DstDb)
	if err != nil {
		return nil, err
	}

	scope, note, err := resolveScope(ctx, srcConn, dstConn, req.Tables)
	if err != nil {
		return nil, err
	}
	if len(scope) == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrSchemaDiffNoTables)
	}

	srcSnapshots, err := loadSnapshots(ctx, srcConn, scope)
	if err != nil {
		return nil, err
	}
	dstSnapshots, err := loadSnapshots(ctx, dstConn, scope)
	if err != nil {
		return nil, err
	}

	result := &Result{
		Source:   connLabel(srcConn),
		Target:   connLabel(dstConn),
		Compared: len(scope),
		Note:     note,
	}
	result.Tables = DiffSnapshots(srcSnapshots, dstSnapshots)
	for _, table := range result.Tables {
		if table.Kind != TableSame {
			result.DiffTables++
		}
	}
	return result, nil
}

func (s *SchemaDiffAppImpl) Snapshot(ctx context.Context, dbId uint64, dbName string, tables []string) ([]TableSnapshot, string, error) {
	conn, err := s.dbApp.GetDbConn(ctx, dbId, dbName)
	if err != nil {
		return nil, "", err
	}

	note := ""
	scoped := make([]string, 0, len(tables))
	for _, table := range tables {
		if trimmed := strings.TrimSpace(table); trimmed != "" {
			scoped = append(scoped, trimmed)
		}
	}
	if len(scoped) == 0 {
		all, listErr := conn.Metadata().GetTables()
		if listErr != nil {
			return nil, "", listErr
		}
		scoped = tableNames(all)
		sort.Strings(scoped)
		if len(scoped) > maxBaselineTables {
			// 截断必须说清：基线只覆盖前 N 张表，其后的表变化不会被报为漂移
			note = i18n.TC(ctx, imsg.SchemaBaselineTruncatedNote, "max", maxBaselineTables, "total", len(scoped))
			scoped = scoped[:maxBaselineTables]
		}
	} else if len(scoped) > maxBaselineTables {
		return nil, "", errorx.NewBizI(ctx, imsg.ErrSchemaDiffTooManyTables, "max", maxBaselineTables)
	}

	snapshots, err := loadSnapshots(ctx, conn, scoped)
	if err != nil {
		return nil, "", err
	}
	return snapshots, note, nil
}

// resolveScope 决定这次比哪些表：调用方给了就用给的（截断到上限），没给则取两侧表名并集。
//
// 取并集而不是交集，正是为了让「目标缺表 / 目标多表」这类漂移看得见
func resolveScope(ctx context.Context, srcConn, dstConn *dbi.DbConn, requested []string) ([]string, string, error) {
	names := make([]string, 0, len(requested))
	seen := make(map[string]bool)
	for _, table := range requested {
		trimmed := strings.TrimSpace(table)
		if trimmed == "" || seen[strings.ToLower(trimmed)] {
			continue
		}
		seen[strings.ToLower(trimmed)] = true
		names = append(names, trimmed)
	}
	if len(names) > 0 {
		if len(names) > maxDiffTables {
			return nil, "", errorx.NewBizI(ctx, imsg.ErrSchemaDiffTooManyTables, "max", maxDiffTables)
		}
		return names, "", nil
	}

	srcTables, err := srcConn.Metadata().GetTables()
	if err != nil {
		return nil, "", err
	}
	dstTables, err := dstConn.Metadata().GetTables()
	if err != nil {
		return nil, "", err
	}
	for _, table := range append(tableNames(srcTables), tableNames(dstTables)...) {
		key := strings.ToLower(table)
		if seen[key] {
			continue
		}
		seen[key] = true
		names = append(names, table)
	}
	sort.Strings(names)

	note := ""
	if len(names) > maxAutoDiffTables {
		// 截断必须说清：只报了前 50 张表的差异不等于整个库只有这些差异
		note = i18n.TC(ctx, imsg.SchemaDiffTruncatedNote, "max", maxAutoDiffTables, "total", len(names))
		names = names[:maxAutoDiffTables]
	}
	return names, note, nil
}

func tableNames(tables []dbi.Table) []string {
	names := make([]string, 0, len(tables))
	for _, table := range tables {
		names = append(names, table.TableName)
	}
	return names
}

// loadSnapshots 取指定表的列/主键/索引。
//
// 先按该侧自己的表清单过滤：这一侧没有的表不能生成快照，否则「目标缺表」这一最要紧的漂移
// 会被当成一张列读不到的表（快照永远存在，两侧差不到哪里去），界面上就看不见缺表了。
// 列一次批量读（GetColumns 支持多表名），主键与索引按表读：方言侧没有批量入口。
// 单表内省失败不阻断整次对比——那一面标成「比不了」即可，用户要看的是其余表的结果
func loadSnapshots(ctx context.Context, conn *dbi.DbConn, tables []string) ([]TableSnapshot, error) {
	existing, err := conn.Metadata().GetTables(tables...)
	if err != nil {
		return nil, err
	}
	present := make(map[string]bool, len(existing))
	for _, table := range existing {
		present[strings.ToLower(table.TableName)] = true
	}
	scoped := make([]string, 0, len(tables))
	for _, table := range tables {
		if present[strings.ToLower(table)] {
			scoped = append(scoped, table)
		}
	}
	if len(scoped) == 0 {
		return nil, nil
	}

	columns, err := conn.Metadata().GetColumns(scoped...)
	if err != nil {
		return nil, err
	}
	columnsByTable := make(map[string][]dbi.Column, len(scoped))
	for _, column := range columns {
		key := strings.ToLower(column.TableName)
		columnsByTable[key] = append(columnsByTable[key], column)
	}

	dbType := string(conn.Info.Type)
	snapshots := make([]TableSnapshot, 0, len(scoped))
	for _, table := range scoped {
		snapshot := TableSnapshot{
			Name:    table,
			Columns: columnsByTable[strings.ToLower(table)],
			DbType:  dbType,
		}

		primaryKeys, pkErr := conn.Metadata().GetPrimaryKeys(table)
		if pkErr != nil {
			logx.WarnfContext(ctx, "schema diff: failed to read primary keys of [%s]: %s", table, pkErr.Error())
		} else {
			snapshot.PrimaryKeys = primaryKeys
			snapshot.PrimaryKeysAvailable = true
		}

		indexes, idxErr := conn.Metadata().GetTableIndex(table)
		if idxErr != nil {
			logx.WarnfContext(ctx, "schema diff: failed to read indexes of [%s]: %s", table, idxErr.Error())
		} else {
			snapshot.Indexes = indexes
			snapshot.IndexesAvailable = true
		}

		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
}

func connLabel(conn *dbi.DbConn) string {
	if conn == nil || conn.Info == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d/%s", conn.Info.Host, conn.Info.Port, conn.Info.Database)
}
