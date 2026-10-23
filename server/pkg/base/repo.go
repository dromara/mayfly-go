package base

import (
	"context"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"reflect"
	"strings"
	"sync"
)

// 基础repo接口
type Repo[T model.ModelI] interface {

	// Insert 新增一个实体
	Insert(ctx context.Context, e T) error

	// BatchInsert 批量新增实体
	BatchInsert(ctx context.Context, models []T) error

	// InsertIfAbsent 新增实体，命中给定唯一列冲突时跳过且不报错（幂等投递、关系建立用）
	InsertIfAbsent(ctx context.Context, e T, conflictColumns ...string) error

	// UpdateById 根据实体id更新实体信息
	UpdateById(ctx context.Context, e T, columns ...string) error

	// UpdateByCond 更新满足条件的数据，并回报真正被改动的行数（行数的判读边界见 RepoImpl.UpdateByCond）
	//  -  values 需要模型结构体或map
	//  -  cond 条件
	UpdateByCond(ctx context.Context, values any, cond any) (int64, error)

	// UpdateColsByCond 按补丁更新满足条件的数据，并回报真正被改动的行数。
	//
	// 与 UpdateByCond 传实体的区别是「写哪些列」由调用方显式声明：零值会被真实写入（清空备注、
	// 状态归零），并支持数据库侧原子自增与表达式赋值
	UpdateColsByCond(ctx context.Context, patch Patch, cond any) (int64, error)

	// Save 保存实体，实体IsCreate返回true则新增，否则更新
	Save(ctx context.Context, e T) error

	// DeleteById 根据实体主键删除实体
	DeleteById(ctx context.Context, id ...uint64) error

	// DeleteByCond 根据实体条件删除实体
	DeleteByCond(ctx context.Context, cond any) error

	// RestoreByCond 撤销满足条件的实体的逻辑删除状态（回收站恢复、重新启用同名记录）。
	//
	// cond 必须声明 IncludeDeleted()（或显式就 is_deleted 表过态），否则基座自动附加的
	// 未删除过滤会让这条更新匹配不到任何行
	RestoreByCond(ctx context.Context, cond any) (int64, error)

	// ExecBySQL 执行原生sql，使用 ctx 绑定的连接（事务内即随事务一起提交/回滚）
	ExecBySQL(ctx context.Context, sql string, params ...any) error

	// SelectBySQL 根据sql语句查询数据，使用 ctx 绑定的连接
	SelectBySQL(ctx context.Context, sql string, res any, params ...any) error

	// GetById 根据实体id查询
	GetById(id uint64, cols ...string) (T, error)

	// GetByIds 根据实体ids查询
	GetByIds(ids []uint64, cols ...string) ([]T, error)

	// GetByCond 根据实体条件查询实体信息（单个结果集）
	//  -  cond 支持普通结构体或*model.QueryCond。如果cond为model.QueryCond，则需要使用Dest方法绑定值的指针
	GetByCond(cond any) error

	// SelectByCondToAny 根据条件查询数据映射至res
	SelectByCondToAny(cond any, res any) error

	// SelectByCond 根据条件查询模型实例数组
	SelectByCond(cond any, cols ...string) ([]T, error)

	// PageByCond 根据查询条件分页查询
	PageByCond(cond any, pageParam model.PageParam, cols ...string) (*model.PageResult[T], error)

	// CountByCond 根据指定条件统计model表的数量。SQL 出错时返回 error，不与「统计为 0」混淆
	CountByCond(cond any) (int64, error)

	// Exists 根据指定条件判断是否存在数据。
	//
	// 带 ctx：事务内做「先查重再写入」的存在性判断必须用它，它走 ctx 绑定的事务连接，
	// 能看见本事务尚未提交的写入
	Exists(ctx context.Context, cond any) (bool, error)

	// SelectByCondWithOffset 根据条件查询数据并支持 offset + limit 分页
	SelectByCondWithOffset(cond any, limit int, offset int) ([]T, error)
}

var _ (Repo[*model.Model]) = (*RepoImpl[*model.Model])(nil)

// 基础repo接口实现
type RepoImpl[T model.ModelI] struct {
	// initOnce 保护 model/modelType 的懒初始化：仓储实例由 IOC 注册为单例并跨请求共享，
	// 无同步的惰性赋值是真实的数据竞争（-race 可复现）
	initOnce sync.Once

	model     any // 模型实例
	modelType reflect.Type
}

func (br *RepoImpl[T]) Insert(ctx context.Context, e T) error {
	return gormx.Insert(ctx, br.fillBaseInfo(ctx, e))
}

func (br *RepoImpl[T]) InsertIfAbsent(ctx context.Context, e T, conflictColumns ...string) error {
	return gormx.InsertIfAbsent(ctx, br.fillBaseInfo(ctx, e), conflictColumns...)
}

func (br *RepoImpl[T]) BatchInsert(ctx context.Context, es []T) error {
	for _, e := range es {
		br.fillBaseInfo(ctx, e)
	}
	return gormx.BatchInsert[T](ctx, es)
}

func (br *RepoImpl[T]) UpdateById(ctx context.Context, e T, columns ...string) error {
	return gormx.UpdateById(ctx, br.fillBaseInfo(ctx, e), columns...)
}

// UpdateByCond 更新满足条件的数据，并回报真正被改动的行数。
//
// 仓储只回事实不做判断：0 行对多数条件更新是正常态（无满足条件的数据、幂等重写），
// 而驱动口径下把相同值写回去也是 0 行。需要按行数判「有没有抢到这次流转」的调用方，
// 应写在名字就说明意图的业务方法里，并确保 SET 的值必然不同于旧值（如带 status=旧状态 的条件）
func (br *RepoImpl[T]) UpdateByCond(ctx context.Context, values any, cond any) (int64, error) {
	return gormx.UpdateByCond(ctx, br.GetModel(), br.normalizeUpdateValues(ctx, values), toQueryCond(cond))
}

// UpdateColsByCond 按补丁更新，列名先过 schema 校验再落库
func (br *RepoImpl[T]) UpdateColsByCond(ctx context.Context, patch Patch, cond any) (int64, error) {
	values, err := patch.toValues()
	if err != nil {
		return 0, err
	}
	if err := gormx.ValidateColumns(br.GetModel(), patch.Columns()); err != nil {
		return 0, err
	}
	return gormx.UpdateByCond(ctx, br.GetModel(), values, toQueryCond(cond))
}

func (br *RepoImpl[T]) Save(ctx context.Context, e T) error {
	if e.IsCreate() {
		return br.Insert(ctx, e)
	}
	return br.UpdateById(ctx, e)
}

func (br *RepoImpl[T]) DeleteById(ctx context.Context, id ...uint64) error {
	_, err := gormx.DeleteById(ctx, br.GetModel(), id...)
	return err
}

func (br *RepoImpl[T]) DeleteByCond(ctx context.Context, cond any) error {
	_, err := gormx.DeleteByCond(ctx, br.GetModel(), toQueryCond(cond))
	return err
}

func (br *RepoImpl[T]) RestoreByCond(ctx context.Context, cond any) (int64, error) {
	return gormx.Restore(ctx, br.GetModel(), toQueryCond(cond))
}

func (br *RepoImpl[T]) ExecBySQL(ctx context.Context, sql string, params ...any) error {
	_, err := gormx.ExecSQL(ctx, sql, params...)
	return err
}

func (br *RepoImpl[T]) SelectBySQL(ctx context.Context, sql string, res any, params ...any) error {
	return gormx.SelectBySQL(ctx, sql, res, params...)
}

func (br *RepoImpl[T]) GetById(id uint64, cols ...string) (T, error) {
	e := br.NewModel()
	return e, gormx.GetById(e, id, withIdColumn(cols)...)
}

func (br *RepoImpl[T]) GetByIds(ids []uint64, cols ...string) ([]T, error) {
	// 没有指定任何 id 时绝不能退化成「无条件查询」：那会把整张表当作这些 id 的结果返回
	if len(ids) == 0 {
		return []T{}, nil
	}
	var models []T
	return models, br.SelectByCondToAny(model.NewCond().In(model.IdColumn, ids).Columns(withIdColumn(cols)...), &models)
}

func (br *RepoImpl[T]) GetByCond(cond any) error {
	return gormx.GetByCond(br.GetModel(), toQueryCond(cond))
}

func (br *RepoImpl[T]) SelectByCondToAny(cond any, res any) error {
	return gormx.SelectByCond(br.GetModel(), toQueryCond(cond).Dest(res))
}

// SelectByCond 根据条件查询实体列表。
//
// 列投影有两个声明口，优先级：显式 cols 参数 > cond 里声明的 Columns。cols 为空时
// 完全沿用 cond 的投影（不会清空）；指定了 cols 则以它为准并自动补上主键列
func (br *RepoImpl[T]) SelectByCond(cond any, cols ...string) ([]T, error) {
	var models []T
	return models, br.SelectByCondToAny(projectCond(toQueryCond(cond), cols), &models)
}

// PageByCond 根据条件分页查询。列投影的优先级与 SelectByCond 一致：
// 显式 cols 参数 > cond 里声明的 Columns，cols 为空时沿用 cond 的投影
func (br *RepoImpl[T]) PageByCond(cond any, pageParam model.PageParam, cols ...string) (*model.PageResult[T], error) {
	var models []T
	return gormx.PageByCond(br.GetModel(), projectCond(toQueryCond(cond), cols), pageParam, models)
}

func (br *RepoImpl[T]) CountByCond(cond any) (int64, error) {
	return gormx.CountByCond(br.GetModel(), toQueryCond(cond))
}

func (br *RepoImpl[T]) Exists(ctx context.Context, cond any) (bool, error) {
	return gormx.Exists(ctx, br.GetModel(), toQueryCond(cond))
}

func (br *RepoImpl[T]) SelectByCondWithOffset(cond any, limit int, offset int) ([]T, error) {
	var models []T
	err := gormx.NewQuery(br.GetModel(), toQueryCond(cond)).GenGdb().Limit(limit).Offset(offset).Find(&models).Error
	if err != nil {
		return nil, err
	}
	return models, nil
}

// NewModel 新建模型实例
func (br *RepoImpl[T]) NewModel() T {
	br.initOnce.Do(br.initModelType)
	return reflect.New(br.modelType).Interface().(T)
}

// GetModel 获取表的模型实例
func (br *RepoImpl[T]) GetModel() T {
	br.initOnce.Do(br.initModelType)
	return br.model.(T)
}

// initModelType 由类型参数解析模型类型与模板实例，只会执行一次
func (br *RepoImpl[T]) initModelType() {
	var model T
	modelType := reflect.TypeOf(model)
	// 检查 model 是否为指针类型
	if modelType.Kind() == reflect.Pointer {
		// 获取指针指向的类型
		modelType = modelType.Elem()
	}
	br.modelType = modelType
	// 用作 GORM 的表/结构解析模板，字段值保持为零值
	br.model = reflect.New(modelType).Interface()
}

// 从上下文获取登录账号信息，并赋值至实体
func (br *RepoImpl[T]) fillBaseInfo(ctx context.Context, e T) T {
	e.FillBaseInfo(contextx.GetLoginAccount(ctx))
	// 主键生成策略（可选能力位，见 model.IdGenerator）：实体实现了它且主键为 0（本次是新增）
	// 时由它发号，否则保持数据库自增。顺序必须在 FillBaseInfo 之后：FillBaseInfo 正是以
	// 主键为 0 判定新建来填创建审计，先赋号会把新建识别成更新
	if gen, ok := any(e).(model.IdGenerator); ok && e.GetId() == 0 {
		e.SetId(gen.GenerateId())
	}
	return e
}

// normalizeUpdateValues 把更新值归一成 gormx 可用的形态：实体走副本补审计并清主键，map 原样透传。
// 形态非法的情况由 gormx.UpdateByCond 统一报错。
//
// 实体形态必须返回副本而不是原地修改：它需要「清主键」与「按更新填审计」两件事同时成立，而它们
// 对主键值的要求相反，只能在副本上做两步（先置非零让审计走更新分支，再清零让 UPDATE 不带主键）。
// 直接在调用方实体上做会把其主键清成 0——它之后再走 Save/UpdateById 就会被判成新建并插入重复数据
//
// map 形态不在这里补审计、也不在这里复制：修改审计只由 gormx.UpdateByCond 一处负责（只有它知道
// 该表真实有哪些审计列，也只有它守「不覆盖调用方显式写明的审计列」这条契约）。两处都补会让
// 同一条契约在实体路径成立、在 map 路径失效
func (br *RepoImpl[T]) normalizeUpdateValues(ctx context.Context, values any) any {
	e, ok := values.(T)
	if !ok {
		return values
	}
	entity := copyEntity(e)
	// 先置一个非零主键再填审计：使 IsCreate() 为假，只走「补修改审计」的分支。
	// 若先清主键，IsCreate() 为真会把创建人与创建时间也填成新值，而这些非零字段随后同样
	// 被写进 SET —— 一次按条件改标题就把该行的创建者改成当前操作人（列表按创建人过滤时，
	// 原创建者当场失去这条数据）
	entity.SetId(1)
	entity = br.fillBaseInfo(ctx, entity)
	// 主键必须为空：否则 UPDATE 会带上主键条件，并把 id 写进 SET 而改写主键列
	entity.SetId(0)
	return entity
}

// copyEntity 浅拷贝实体指针指向的结构体，返回独立副本
func copyEntity[T model.ModelI](e T) T {
	rv := reflect.ValueOf(e)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return e
	}
	cp := reflect.New(rv.Type().Elem())
	cp.Elem().Set(rv.Elem())
	return cp.Interface().(T)
}

// toQueryCond 统一转为*model.QueryCond
func toQueryCond(cond any) *model.QueryCond {
	if qc, ok := cond.(*model.QueryCond); ok {
		// typed-nil 指针也要归一成空条件，否则后续解引用会 panic
		if qc == nil {
			return model.NewCond()
		}
		return qc
	}
	return model.NewModelCond(cond)
}

// projectCond 在条件的副本上应用显式列参数。
//
// 条件是引用语义且常被构建一次、多处复用（先查明细再统计），就地写入投影会把本次
// 调用的列选择泄漏给后续每一次查询；cols 为空时不触碰原条件，沿用调用方声明的投影
func projectCond(cond *model.QueryCond, cols []string) *model.QueryCond {
	if len(cols) == 0 {
		return cond
	}
	return cond.Clone().Columns(withIdColumn(cols)...)
}

// withIdColumn 为实体投影列补上主键列。
//
// 只适用于返回实体本身的方法：调用方指定部分列时若漏掉主键，回来的实体主键为 0，
// 而 `IsCreate()` 正是以主键为 0 判定新建——拿它再走 Save/UpdateById 就会静默插入一条重复数据。
// 返回 PO/DTO 的投影方法（SelectByCondToAny）不适用，它们不依赖主键做存在性判断。
// 入参为空时原样返回：搭配 Columns 的零参 no-op 语义，cond 里已声明的投影得以保留
func withIdColumn(cols []string) []string {
	if len(cols) == 0 {
		return cols
	}
	for _, c := range cols {
		if strings.EqualFold(c, model.IdColumn) {
			return cols
		}
	}
	return append(append([]string{}, cols...), model.IdColumn)
}
