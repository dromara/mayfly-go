package gormx

import (
	"context"
	"errors"
	"fmt"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"reflect"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrIdsRequired 按主键删除/更新时必须给出至少一个主键
var ErrIdsRequired = errors.New("ids cannot be empty")

// ErrUpdateValues 更新值形态非法：只接受实体或 map
var ErrUpdateValues = errors.New("invalid update values, must be a model struct pointer or map")

// ErrConflictColumnsRequired 冲突列未指明时无法判定「已存在」的口径，可能误按主键跳过插入
var ErrConflictColumnsRequired = errors.New("insert if absent requires conflict columns")

// ErrCondRequired 破坏性操作（删除、按条件更新、恢复）必须带调用方声明的过滤条件。
//
// 驱动层也会拦无条件的 UPDATE/DELETE，但那是实现细节：AllowGlobalUpdate 一开就失效，而且报
// 出来的文案说不清是调用方漏传了条件。能不能整表操作应该是基座的显式策略
var ErrCondRequired = errors.New("destructive operation requires at least one caller-declared condition")

// requireConds 破坏性操作的入口卡点
func requireConds(cond *model.QueryCond) error {
	if !toCond(cond).HasConds() {
		return ErrCondRequired
	}
	return nil
}

// DefaultBatchSize 批量插入的单批条数。
//
// 整批一次性提交会生成一条巨型 INSERT，受 max_allowed_packet 限制在数据导入场景直接失败；
// 500 行在元数据库的列数规模下远低于该限制，同时不至于产生过多往返
const DefaultBatchSize = 500

// 根据id获取实体对象。model需为指针类型（需要将查询出来的值赋值给model）
//
// 若记录不存在，返回包装了 model.ErrNotFound 的错误（对 gorm.ErrRecordNotFound 仍成立）
//   - dbModel  数据库映射实体模型
func GetById(dbModel model.ModelI, id uint64, cols ...string) error {
	return translateNotFound(NewQuery(dbModel, model.NewCond().Columns(cols...).EqStrict(model.IdColumn, id)).GenGdb().First(dbModel).Error)
}

// 根据model获取单个实体对象
//
// 若 error不为nil，则为不存在该记录
//   - cond  查询条件
func GetByCond(dbModel model.ModelI, cond *model.QueryCond) error {
	dest, err := condDest(cond)
	if err != nil {
		return err
	}
	return translateNotFound(NewQuery(dbModel, cond).GenGdb().First(dest).Error)
}

// CountByCond 根据条件cond获取指定model表统计数量。
//
// 必须回报错误：表不存在、条件生成失败等 SQL 错误若被吞掉，返回值 0 与「确实没有数据」
// 就无法区分，而统计值常被用于「为 0 则新建」这类幂等判断，把 SQL 错误当成 0 会直接写坏数据
func CountByCond(dbModel model.ModelI, cond *model.QueryCond) (int64, error) {
	var count int64
	gdb := NewQuery(dbModel, toCond(cond)).GenGdb()
	if gdb.Error != nil {
		return 0, gdb.Error
	}
	if err := gdb.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// Exists 判断是否存在满足条件的数据。
//
// ctx 里绑定了事务时使用事务连接，因此可以在事务内看到自己尚未提交的写入——按「先查重再插入」
// 的入口做存在性判断必须用它，而不是用 COUNT(*)：后者既读不到事务内的未提交数据，也比 LIMIT 1 贵
func Exists(ctx context.Context, dbModel model.ModelI, cond *model.QueryCond) (bool, error) {
	cond = toCond(cond)
	// 条件在构造期就带了非法片段（如 OrderBy 注入）时必须报错，而不是静默忽略：
	// 其它读入口都经 GenGdb 做这件事，Exists 少这一步会让同一个坏条件在两条路上表现不同
	if err := cond.GetErr(); err != nil {
		return false, err
	}
	gdb := DB(ctx).Model(dbModel).Select(model.IdColumn).Limit(1)
	if err := gdb.Error; err != nil {
		return false, err
	}
	setGdbWhere(gdb, cond)
	applyDeletedFilter(gdb, dbModel, cond)

	var ids []uint64
	if err := gdb.Find(&ids).Error; err != nil {
		return false, err
	}
	return len(ids) > 0, nil
}

// PageQuery 根据查询条件分页查询数据。若未指定查询列，则查询列以 toModels 字段为准；
// 软删过滤由语句生成时按模型支持性与调用方声明自动判定，无需调用方关心
func (q *Query) PageQuery[T any](pageParam model.PageParam, toModels []T) (*model.PageResult[T], error) {
	// 计数与列表用两条独立语句：计数不吃列投影——GORM 对单列投影的 Count 会生成
	// COUNT(该列)，列里出现 NULL 时行数被少计、页尾数据被静默截断；分页总数的语义恒为 count(*)
	countGdb := q.GenGdb()
	if countGdb.Error != nil {
		return nil, countGdb.Error
	}
	countGdb.Statement.Selects = nil
	delete(countGdb.Statement.Clauses, "SELECT")
	var count int64
	if err := countGdb.Count(&count).Error; err != nil {
		return nil, err
	}

	// 生效的分页参数写进结果：请求被上限夹小时，调用方必须能看出来自己只拿到了一部分。
	// List 一律给出空切片而不是 nil：序列化后前端拿到的是 []，否则每个消费点都要各自防 null
	pageNum, pageSize := pageParam.Normalize()
	if count == 0 {
		return &model.PageResult[T]{Total: 0, List: []T{}, PageNum: pageNum, PageSize: pageSize}, nil
	}

	// 偏移必须在 int64 内算：页码由请求直接传入，(pageNum-1)*pageSize 在 int 上先乘会在
	// 极大页码时溢出为负数，绕过越界判断并给 SQL 一个负 OFFSET
	offset := int64(pageNum-1) * int64(pageSize)
	if offset >= count {
		// 页码越界：总数照常回报，列表为空
		return &model.PageResult[T]{Total: count, List: []T{}, PageNum: pageNum, PageSize: pageSize}, nil
	}

	listGdb := q.GenGdb()
	if listGdb.Error != nil {
		return nil, listGdb.Error
	}
	if err := listGdb.Limit(pageSize).Offset(int(offset)).Find(&toModels).Error; err != nil {
		return nil, err
	}
	return &model.PageResult[T]{Total: count, List: toModels, PageNum: pageNum, PageSize: pageSize}, nil
}

// PageByCond 根据指定查询条件分页查询数据
func PageByCond[T any](dbModel model.ModelI, cond *model.QueryCond, pageParam model.PageParam, toModels []T) (*model.PageResult[T], error) {
	return NewQuery(dbModel, cond).PageQuery(pageParam, toModels)
}

// SelectByCond 根据条件查询结果集
func SelectByCond(dbModel model.ModelI, cond *model.QueryCond) error {
	dest, err := condDest(cond)
	if err != nil {
		return err
	}
	return NewQuery(dbModel, cond).GenGdb().Find(dest).Error
}

// SelectBySQL 根据sql查询数据。
//
// 原生 SQL 无从推断所属事务，因此调用方需自行通过 ctxOf 决定用事务连接还是连接池，见 ExecSQL
func SelectBySQL(ctx context.Context, sql string, toEntity any, params ...any) error {
	return DB(ctx).Raw(sql, params...).Scan(toEntity).Error
}

// ExecSQL 执行原生 SQL 并回报影响行数。
//
// 走 ctx 绑定连接：事务内的原生 SQL 必须与同事务的其他写入一起提交或一起回滚，否则会出现
// 「业务写入回滚了、原生 SQL 已生效」的半提交状态
func ExecSQL(ctx context.Context, sql string, params ...any) (int64, error) {
	result := DB(ctx).Exec(sql, params...)
	return result.RowsAffected, result.Error
}

// Insert 插入model
//   - model  数据库映射实体模型
func Insert(ctx context.Context, model model.ModelI) error {
	return DB(ctx).Create(model).Error
}

// InsertIfAbsent 插入实体，命中给定唯一列冲突时静默跳过（不报错、不覆盖既有行）。
//
// 落库队列、关联关系这类「有则跳过」的写入必须靠数据库唯一约束而不是先查再插：后者在并发下
// 两个请求都会查到「不存在」从而各插一行。冲突列必须由调用方指明，避免误按主键冲突跳过
func InsertIfAbsent(ctx context.Context, entity model.ModelI, conflictColumns ...string) error {
	if len(conflictColumns) == 0 {
		return ErrConflictColumnsRequired
	}
	columns := make([]clause.Column, 0, len(conflictColumns))
	for _, column := range conflictColumns {
		columns = append(columns, clause.Column{Name: column})
	}
	return DB(ctx).Clauses(clause.OnConflict{Columns: columns, DoNothing: true}).Create(entity).Error
}

// BatchInsert 批量插入，按 DefaultBatchSize 分批提交。
//
// 空集合视为调用方的无操作，直接返回 nil：此时生成一条没有 VALUES 的 INSERT 只会得到语法错误
func BatchInsert[T model.ModelI](ctx context.Context, models []T) error {
	total := len(models)
	if total == 0 {
		return nil
	}
	gdb := DB(ctx)
	for start := 0; start < total; start += DefaultBatchSize {
		end := min(start+DefaultBatchSize, total)
		if err := gdb.Create(models[start:end]).Error; err != nil {
			return err
		}
	}
	return nil
}

// UpdateById 根据id更新model，更新字段为model中不为空的值，即int类型不为0，ptr类型不为nil这类字段值
//   - model  数据库映射实体模型
func UpdateById(ctx context.Context, model model.ModelI, columns ...string) error {
	return DB(ctx).Model(model).Select(columns).Updates(model).Error
}

// UpdateByCond 更新满足条件的数据并回报真正被改动的行数，见 docOfUpdateByCond
func UpdateByCond(ctx context.Context, dbModel model.ModelI, values any, cond *model.QueryCond) (int64, error) {
	if err := requireConds(cond); err != nil {
		return 0, err
	}
	// 更新列白名单只来自 values，不再取自 cond：cond 的 selectColumns 语义是「读哪些列」，
	// 复用成「写哪些列」会让读侧的任何列处理（如自动并入主键）变成写侧的主键改写
	gormDb := DB(ctx).Model(dbModel)
	setGdbWhere(gormDb, toCond(cond))

	// map 形态没有实体钩子可挂，修改审计统一在这里补齐；实体形态由调用方的 fillBaseInfo 负责。
	// 先归一成副本再补：addUpdateAuditColumns 会就地增删键，直接改调用方的 map 会让复用于重试或
	// 多次提交的同一份参数被污染。两条路都必须按实体真实拥有的列过滤：CreateModelNLD 没有
	// update_time、DeletedModel 没有修改人，无条件写入会生成不存在的列名直接报错
	if valueMap, isMap := toUpdateMap(values); isMap {
		values = valueMap
		if err := addUpdateAuditColumns(ctx, gormDb, dbModel, valueMap); err != nil {
			return 0, err
		}
	} else if !isValidUpdateValues(values, dbModel) {
		return 0, fmt.Errorf("%w, got %T", ErrUpdateValues, values)
	}

	result := gormDb.Updates(values)
	return result.RowsAffected, result.Error
}

// toUpdateMap 将 map 形态的更新值归一成 map[string]any 副本；不是 map 形态时回报 false
func toUpdateMap(values any) (map[string]any, bool) {
	var source map[string]any
	switch m := values.(type) {
	case map[string]any:
		source = m
	case collx.M:
		source = map[string]any(m)
	default:
		return nil, false
	}
	copied := make(map[string]any, len(source))
	for k, v := range source {
		copied[k] = v
	}
	return copied, true
}

// 根据id删除model
//   - model  数据库映射实体模型
func DeleteById(ctx context.Context, model_ model.ModelI, id ...uint64) (int64, error) {
	if len(id) == 0 {
		return 0, ErrIdsRequired
	}
	return DeleteByCond(ctx, model_, model.NewCond().In(model.IdColumn, id))
}

// DeleteByCond 根据cond条件删除指定model表数据，并回报真正被影响的行数
//
//   - dbModel  数据库映射实体模型
//   - cond  条件
func DeleteByCond(ctx context.Context, dbModel model.ModelI, cond *model.QueryCond) (int64, error) {
	if err := requireConds(cond); err != nil {
		return 0, err
	}
	gdb := setGdbWhere(DB(ctx).Model(dbModel), toCond(cond))
	if !dbModel.LogicDelete() {
		result := gdb.Delete(dbModel)
		return result.RowsAffected, result.Error
	}

	// 逻辑删除同样必须留下可归因的修改记录：只写 is_deleted/delete_time 时，界面上「谁删的」
	// 会显示成最后一次修改人（或创建人），审计链在删除这一步断掉
	values := map[string]any{
		model.DeletedColumn:    model.ModelDeleted,
		model.DeleteTimeColumn: time.Now(),
	}
	if err := addAuditColumns(ctx, gdb, dbModel, values); err != nil {
		return 0, err
	}
	result := gdb.Updates(values)
	return result.RowsAffected, result.Error
}

// addUpdateAuditColumns 为 map 形态的更新补齐修改审计。
//
// 与 addAuditColumns 的区别是这里**不覆盖调用方已显式给出的列**：更新补丁里写明的
// update_time/修改人有业务含义（如回填历史操作的执行时间），基座无权改写
func addUpdateAuditColumns(ctx context.Context, gdb *gorm.DB, dbModel model.ModelI, values map[string]any) error {
	if _, ok := values[model.UpdateTimeColumn]; !ok {
		values[model.UpdateTimeColumn] = time.Now()
	}
	if account := contextx.GetLoginAccount(ctx); account != nil {
		if _, ok := values[model.ModifierColumn]; !ok {
			values[model.ModifierColumn] = account.Username
		}
		if _, ok := values[model.ModifierIdColumn]; !ok {
			values[model.ModifierIdColumn] = account.Id
		}
	}
	return dropMissingAuditColumns(gdb, dbModel, values)
}

// addAuditColumns 补上更新时间与修改人，且只写该表真实存在的审计列。
//
// 审计列有无是实体选型的结果（DeletedModel 就没有 update_time/modifier），无条件写入会生成
// 不存在的列名直接报错。与 addUpdateAuditColumns 不同，这里会覆盖既有值：删除/恢复的归因必须
// 是本次操作人
func addAuditColumns(ctx context.Context, gdb *gorm.DB, dbModel model.ModelI, values map[string]any) error {
	values[model.UpdateTimeColumn] = time.Now()
	// 无登录账号时回落系统账号，与实体审计填充口径一致：留着上一次的修改人，会让这次
	// 删除/恢复看起来是「最后编辑这行的人干的」，审计归因因此失真
	account := contextx.GetLoginAccount(ctx)
	if account == nil {
		account = model.SysAccount
	}
	values[model.ModifierColumn] = account.Username
	values[model.ModifierIdColumn] = account.Id
	return dropMissingAuditColumns(gdb, dbModel, values)
}

// dropMissingAuditColumns 删除实体表不存在的审计列。
//
// 直接查 schema 的字段表而不是构建整表列集合：审计列只需逐个确认存在性。
// schema 解析走 GORM 的类型级缓存，同类型重复解析无额外开销；解析失败时如实报错
// 而不是猜：拿不到 schema 事实就静默丢审计列，与静默写不存在的列同样是数据完整性事故
func dropMissingAuditColumns(gdb *gorm.DB, dbModel model.ModelI, values map[string]any) error {
	statement := &gorm.Statement{DB: gdb}
	if err := statement.Parse(dbModel); err != nil {
		return fmt.Errorf("parse %T schema: %w", dbModel, err)
	}
	for _, column := range []string{model.UpdateTimeColumn, model.ModifierColumn, model.ModifierIdColumn} {
		if _, ok := statement.Schema.FieldsByDBName[column]; !ok {
			delete(values, column)
		}
	}
	return nil
}

// ErrUnknownColumn 更新里出现了该表不存在的列。
//
// 列名写错不能被静默接受：GORM 会把 map 里查不到对应字段的键当作物理列名直接拼进 SET 子句，
// 于是「写错的列名」变成「运行期 SQL 报错」或更糟——写进了另一个名字相近的合法列
var ErrUnknownColumn = errors.New("unknown column for this model")

// ValidateColumns 校验给定的列名是否都是该实体表真实存在的列
func ValidateColumns(dbModel model.ModelI, columns []string) error {
	if len(columns) == 0 {
		return nil
	}
	statement := &gorm.Statement{DB: DB(context.Background())}
	if err := statement.Parse(dbModel); err != nil {
		// schema 解析失败即实体定义错误，报错而非放行：放行会让列校验空转，
		// 「列不存在」的拦截因此静默失效
		return fmt.Errorf("parse %T schema: %w", dbModel, err)
	}
	for _, column := range columns {
		// 表限定列（t.status）只校验列名部分
		name := column
		if _, suffix, hasAlias := strings.Cut(column, "."); hasAlias {
			name = suffix
		}
		if _, ok := statement.Schema.FieldsByDBName[name]; !ok {
			return fmt.Errorf("%w: %s on table %s", ErrUnknownColumn, name, statement.Schema.Table)
		}
	}
	return nil
}

// isValidUpdateValues 校验实体形态的更新值。
//
// 形态不对时绝不能原样交给 GORM：values 与 Model 不是同一对象时，GORM 会把主键当普通列放进
// SET 子句，等于允许调用方搬动主键
func isValidUpdateValues(values any, dbModel model.ModelI) bool {
	// 实体形态要求与目标模型同为一种实体，避免拿 A 实体去更新 B 表
	valuesType, modelType := indirectType(values), indirectType(dbModel)
	return valuesType != nil && modelType != nil && valuesType == modelType
}

// indirectType 取掉指针后的结构体类型，非结构体返回 nil
func indirectType(v any) reflect.Type {
	if v == nil {
		return nil
	}
	rt := reflect.TypeOf(v)
	for rt != nil && rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	if rt == nil || rt.Kind() != reflect.Struct {
		return nil
	}
	return rt
}

// condDest 取得结果集绑定目标；没有绑定则返回错误而不是 panic，
// 用法问题不应该让进程付出代价
func condDest(cond *model.QueryCond) (any, error) {
	if cond == nil {
		return nil, ErrCondDestMissing
	}
	if !cond.HasDest() {
		return nil, ErrCondDestMissing
	}
	return cond.GetDest(), nil
}

// toCond 归一化条件指针，typed-nil 与 nil 都换成空条件，避免解引用空指针
func toCond(cond *model.QueryCond) *model.QueryCond {
	if cond == nil {
		return model.NewCond()
	}
	return cond
}

// translateNotFound 把驱动层的「未找到记录」翻译成基座哨兵，同时保留原错误以便既有判读继续成立
func translateNotFound(err error) error {
	if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return fmt.Errorf("%w: %w", model.ErrNotFound, err)
}

// ErrNotSoftDeletable 该实体不是逻辑删除模型，没有「已删除」状态可撤销
var ErrNotSoftDeletable = errors.New("model does not use logical deletion")

// Restore 撤销满足条件的数据的逻辑删除状态，并回报真正被改动的行数。
//
// 删除有 DeleteByCond，撤销删除必须有对称入口：否则「回收站恢复」「重新启用同名记录」这类需求
// 只能靠手写 UPDATE is_deleted=0 绕开基座，且必然漏掉审计列
func Restore(ctx context.Context, dbModel model.ModelI, cond *model.QueryCond) (int64, error) {
	if !dbModel.LogicDelete() {
		return 0, ErrNotSoftDeletable
	}
	if err := requireConds(cond); err != nil {
		return 0, err
	}
	// 未声明包含已删除数据时，基座会自动附加 is_deleted = 0，这条「恢复」更新就永远匹配不到
	// 任何行并静默返回 0 行，因此这里要求调用方显式声明范围
	if needDeletedFilter(dbModel, toCond(cond)) {
		return 0, ErrRestoreNeedsIncludeDeleted
	}
	gdb := setGdbWhere(DB(ctx).Model(dbModel), toCond(cond))
	values := map[string]any{
		model.DeletedColumn:    model.ModelUndeleted,
		model.DeleteTimeColumn: nil,
	}
	if err := addAuditColumns(ctx, gdb, dbModel, values); err != nil {
		return 0, err
	}
	result := gdb.Updates(values)
	return result.RowsAffected, result.Error
}

// ErrRestoreNeedsIncludeDeleted 撤销逻辑删除必须显式声明包含已删除数据：
// 否则基座自动附加的未删除过滤会让这条更新永远匹配不到任何行，并静默返回 0 行
var ErrRestoreNeedsIncludeDeleted = errors.New("restore requires a condition that includes deleted rows")
