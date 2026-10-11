package base

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"time"
)

const (
	// TxWarnHoldDuration 事务持有时间超过该值即告警：多数事务应在数毫秒内完成，
	// 持有时间长通常意味着事务里夹了远程调用或慢 SQL，必须暴露出来而不是静默承受
	TxWarnHoldDuration = 500 * time.Millisecond
	// TxErrorHoldDuration 事务持有时间超过该值按错误记录
	TxErrorHoldDuration = 5 * time.Second
)

// 基础application接口
type App[T model.ModelI] interface {

	// Insert 新增一个实体
	Insert(ctx context.Context, e T) error

	// BatchInsert 批量新增实体
	BatchInsert(ctx context.Context, models []T) error

	// InsertIfAbsent 新增实体，命中给定唯一列冲突时跳过且不报错
	InsertIfAbsent(ctx context.Context, e T, conflictColumns ...string) error

	// UpdateById 根据实体id更新实体信息
	UpdateById(ctx context.Context, e T) error

	// UpdateByCond 更新满足条件的数据
	//  - values 需为模型结构体指针或map(更新零值等)
	//  - cond 可为*model.QueryCond也可以为普通查询model
	UpdateByCond(ctx context.Context, values any, cond any) error

	// UpdateColsByCond 按补丁更新满足条件的数据：零值会真实写入，并支持原子自增与表达式赋值
	UpdateColsByCond(ctx context.Context, patch Patch, cond any) error

	// DeleteById 根据实体主键删除实体
	DeleteById(ctx context.Context, id ...uint64) error

	// DeleteByCond 根据条件进行删除
	//  - cond 可为*model.QueryCond也可以为普通查询model
	DeleteByCond(ctx context.Context, cond any) error

	// RestoreByCond 撤销满足条件的实体的逻辑删除状态，语义与恢复前置条件见 Repo.RestoreByCond
	RestoreByCond(ctx context.Context, cond any) error

	// Save 保存实体，实体IsCreate返回true则新增，否则更新
	Save(ctx context.Context, e T) error

	// GetById 根据实体id查询
	GetById(id uint64, cols ...string) (T, error)

	// GetByIds 根据实体id数组查询
	GetByIds(ids []uint64, cols ...string) ([]T, error)

	// GetByCond 根据实体条件查询实体信息(获取单个实体)
	//  - cond 可为*model.QueryCond也可以为普通查询model
	GetByCond(cond any) error

	// ListByCondToAny 根据条件查询数据映射至res
	//  - cond 可为*model.QueryCond也可以为普通查询model
	ListByCondToAny(cond any, res any) error

	// ListByCond 根据条件查询
	//  - cond 可为*model.QueryCond也可以为普通查询model
	ListByCond(cond any, cols ...string) ([]T, error)

	// PageByCond 根据指定条件分页查询
	//  - cond 可为*model.QueryCond也可以为普通查询model
	PageByCond(cond any, pageParam model.PageParam, cols ...string) (*model.PageResult[T], error)

	// CountByCond 根据指定条件统计model表的数量。SQL 出错时返回 error，不与「统计为 0」混淆
	CountByCond(cond any) (int64, error)

	// Exists 根据指定条件判断是否存在数据。
	//
	// 带 ctx 以便在事务内看到本事务尚未提交的写入，「先查重再写入」必须用它而不是
	// CountByCond(...) > 0
	Exists(ctx context.Context, cond any) (bool, error)

	//  CursorByCond 根据指定条件遍历model表数据
	CursorByCond(cond any, handler func(T) error) error

	// Tx 执行事务操作
	Tx(ctx context.Context, funcs ...func(context.Context) error) (err error)
}

var _ (App[*model.Model]) = (*AppImpl[*model.Model, *RepoImpl[*model.Model]])(nil)

// 基础application接口实现
type AppImpl[T model.ModelI, R Repo[T]] struct {
	Repo R `inject:"T"` // repo接口, 根据类型进行注入
}

// 获取repo
func (ai *AppImpl[T, R]) GetRepo() R {
	return ai.Repo
}

// 新增一个实体 (单纯新增，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) Insert(ctx context.Context, e T) error {
	return ai.GetRepo().Insert(ctx, e)
}

// InsertIfAbsent 新增实体，命中唯一列冲突时跳过（幂等投递用），不回报冲突与否
func (ai *AppImpl[T, R]) InsertIfAbsent(ctx context.Context, e T, conflictColumns ...string) error {
	return ai.GetRepo().InsertIfAbsent(ctx, e, conflictColumns...)
}

// 批量新增实体 (单纯新增，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) BatchInsert(ctx context.Context, es []T) error {
	return ai.GetRepo().BatchInsert(ctx, es)
}

// 根据实体id更新实体信息 (单纯更新，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) UpdateById(ctx context.Context, e T) error {
	return ai.GetRepo().UpdateById(ctx, e)
}

// UpdateByCond 更新满足条件的数据 (单纯更新，不做其他业务逻辑处理)
//   - values 需为模型结构体指针或map(更新零值等)
//   - cond 可为*model.QueryCond也可以为普通查询model
//
// 应用层只报执行是否出错，不回传改动行数：0 行对多数条件更新是正常态（无在途数据、幂等重写），
// 回一个「是否改到行」的布尔值会把无事发生编码成失败、迫使每个调用点丢弃它。需要按行数判
// 「没抢到」的流转，应在业务具名方法里直接调仓储的行数入口（Repo.UpdateByCond）
func (ai *AppImpl[T, R]) UpdateByCond(ctx context.Context, values any, cond any) error {
	_, err := ai.GetRepo().UpdateByCond(ctx, values, cond)
	return err
}

func (ai *AppImpl[T, R]) RestoreByCond(ctx context.Context, cond any) error {
	_, err := ai.GetRepo().RestoreByCond(ctx, cond)
	return err
}

// UpdateColsByCond 按补丁更新满足条件的数据 (单纯更新，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) UpdateColsByCond(ctx context.Context, patch Patch, cond any) error {
	_, err := ai.GetRepo().UpdateColsByCond(ctx, patch, cond)
	return err
}

// 保存实体，实体IsCreate返回true则新增，否则更新
func (ai *AppImpl[T, R]) Save(ctx context.Context, e T) error {
	return ai.GetRepo().Save(ctx, e)
}

// 根据实体主键删除实体 (单纯删除实体，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) DeleteById(ctx context.Context, id ...uint64) error {
	return ai.GetRepo().DeleteById(ctx, id...)
}

// 根据指定条件删除实体 (单纯删除实体，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) DeleteByCond(ctx context.Context, cond any) error {
	return ai.GetRepo().DeleteByCond(ctx, cond)
}

// 根据实体id查询
func (ai *AppImpl[T, R]) GetById(id uint64, cols ...string) (T, error) {
	return ai.GetRepo().GetById(id, cols...)
}

func (ai *AppImpl[T, R]) GetByIds(ids []uint64, cols ...string) ([]T, error) {
	return ai.GetRepo().GetByIds(ids, cols...)
}

// 根据实体条件查询实体信息
func (ai *AppImpl[T, R]) GetByCond(cond any) error {
	return ai.GetRepo().GetByCond(cond)
}

func (ai *AppImpl[T, R]) ListByCondToAny(cond any, res any) error {
	return ai.GetRepo().SelectByCondToAny(cond, res)
}

func (ai *AppImpl[T, R]) ListByCond(cond any, cols ...string) ([]T, error) {
	return ai.GetRepo().SelectByCond(cond, cols...)
}

func (ai *AppImpl[T, R]) PageByCond(cond any, pageParam model.PageParam, cols ...string) (*model.PageResult[T], error) {
	return ai.GetRepo().PageByCond(cond, pageParam, cols...)
}

// 根据指定条件统计model表的数量, cond为条件可以为map等
func (ai *AppImpl[T, R]) CountByCond(cond any) (int64, error) {
	return ai.GetRepo().CountByCond(cond)
}

func (ai *AppImpl[T, R]) Exists(ctx context.Context, cond any) (bool, error) {
	return ai.GetRepo().Exists(ctx, cond)
}

// CursorByCond 按主键游标分批遍历满足条件的数据。
//
// 用游标而非 offset 翻页：遍历期间有数据被删除或插入时，offset 会跳过或重复处理部分行，
// 而按主键游标推进只依赖「已处理过的主键」这一事实，与并发写入无关；深翻页也不再随偏移量线性变慢
func (ai *AppImpl[T, R]) CursorByCond(cond any, handler func(T) error) error {
	return CursorByCond(ai.GetRepo(), cond, DefaultCursorBatchSize, handler)
}

// DefaultCursorBatchSize 游标遍历的单批条数
const DefaultCursorBatchSize = 200

// ErrCursorOrder 游标遍历只支持按主键推进。
//
// 调用方指定了其他排序时不能默默改成主键排序：那会让它拿到的数据顺序与自己的声明不符，
// 必须报错让调用方决定
var ErrCursorOrder = errors.New("cursor iteration does not support custom order by")

// CursorByCond 按主键游标分批遍历仓储数据，batch 为单批条数
func CursorByCond[T model.ModelI](repo Repo[T], cond any, batch int, handler func(T) error) error {
	if batch <= 0 {
		batch = DefaultCursorBatchSize
	}
	origin := toQueryCond(cond)
	if len(origin.GetOrderBy()) > 0 {
		return ErrCursorOrder
	}
	// 游标按主键推进：投影若不含主键，每批回来的实体主键恒为 0，游标边界永远不动，
	// 同一批数据会被无限重复处理。在副本上补主键，不污染调用方条件
	if cols := origin.GetSelectColumns(); len(cols) > 0 {
		origin = origin.Clone().Columns(withIdColumn(cols)...)
	}

	lastId := uint64(0)
	for {
		// 每批基于副本追加游标条件，不污染调用方传入的条件对象
		batchCond := origin.Clone()
		if lastId > 0 {
			batchCond = batchCond.Lt(model.IdColumn, lastId)
		}
		list, err := repo.SelectByCondWithOffset(batchCond, batch, 0)
		if err != nil {
			return err
		}
		if len(list) == 0 {
			return nil
		}

		for _, item := range list {
			if err := handler(item); err != nil {
				return err
			}
		}

		// 默认按主键降序返回，本批最后一条即最小主键，作为下一批的游标边界
		lastId = list[len(list)-1].GetId()
		if len(list) < batch {
			return nil
		}
	}
}

// Tx 执行事务操作：以同一个事务依次执行 funcs，任一环节失败则整体回滚。
//
// ctx 已绑定事务时直接加入外层事务，不开新事务也不负责提交回滚：只有事务创建者才有权结束
// 事务。内层若自己 recover，会把程序缺陷降级成「业务失败」并让外层拿不到统一告警入口
func (ai *AppImpl[T, R]) Tx(ctx context.Context, funcs ...func(context.Context) error) (err error) {
	return Tx(ctx, funcs...)
}

// Tx 在 ctx 未绑定事务时开启一个新事务并依次执行 funcs。
//
// 事务只覆盖本地元数据库的可回滚写入：远程调用（目标实例 SQL、机器/Redis 命令、模型调用等）
// 不得放在事务里，它们既不可回滚，又会把行锁持有时间拉长到秒级甚至分钟级。这类动作请用
// AfterCommit 挂到提交之后，或配合状态位落库后由后台任务重驱动
func Tx(ctx context.Context, funcs ...func(context.Context) error) (err error) {
	return TxWith(ctx, nil, funcs...)
}

// ErrAfterCommit 事务已提交，但登记在其后的提交后动作执行失败。
//
// 调用方必须能把它与「事务失败」区分开：此时数据已经生效，按整体失败去重试会造成重复副作用，
// 只能按该动作自身的语义去补偿
var ErrAfterCommit = errors.New("transaction committed but a post-commit action failed")

// TxWith 开启事务并可选指定隔离级别/只读等选项
func TxWith(ctx context.Context, opts *sql.TxOptions, funcs ...func(context.Context) error) (err error) {
	if gormx.InTx(ctx) {
		// 加入外层事务：不注册 recover、也不负责提交回滚，结束权只属于创建者
		for _, f := range funcs {
			if err := f(ctx); err != nil {
				return err
			}
		}
		return nil
	}

	txCtx, tx, err := gormx.Begin(ctx, opts)
	if err != nil {
		return err
	}
	txCtx, afterQueue := withAfterCommitQueue(txCtx)

	beginAt := time.Now()
	// defer 后进先出：先注册「结束事务」，再注册 panic 收口，因此 recover 先执行并把错误写入
	// 具名返回值，结束事务时能看到它。回滚失败必须合并进返回错误，否则调用方会以为已经回滚干净
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback().Error; rbErr != nil {
				err = errors.Join(err, fmt.Errorf("rollback transaction failed: %w", rbErr))
			}
			reportTxHold(time.Since(beginAt), len(funcs))
			return
		}
		if commitErr := tx.Commit().Error; commitErr != nil {
			err = commitErr
			reportTxHold(time.Since(beginAt), len(funcs))
			return
		}
		// 看门狗量的是「事务持有时间」，必须在提交那一刻定格：提交后动作按设计就可以是秒级的
		// 外部副作用，把它们算进来会让告警长期假阳性，而假阳性的告警连带着把「真的在事务里夹了
		// 远程调用」那一次也一起忽略掉
		reportTxHold(time.Since(beginAt), len(funcs))

		// 提交成功才执行提交后动作；回滚路径上一个都不会执行
		actionBeginAt := time.Now()
		actionErr := runAfterCommit(txCtx, afterQueue)
		if actionErr != nil {
			// 两个 %w 而不是 %v：调用方需要 errors.Is 到具体哪个动作失败才能按该动作自身的语义补偿
			err = fmt.Errorf("%w: %w", ErrAfterCommit, actionErr)
		}
		reportAfterCommitHold(time.Since(actionBeginAt), actionErr)
	}()
	defer gox.Recover(func(e error) { err = e })

	for _, f := range funcs {
		if err = f(txCtx); err != nil {
			return
		}
	}
	return
}

// reportTxHold 上报事务持有时间。
//
// 持有时间远超正常值，几乎总是因为事务里夹了外部调用或慢 SQL：它会把行锁与连接一起占用，
// 必须先可观测，否则这类问题只能等线上卡死才发现
func reportTxHold(held time.Duration, funcCount int) {
	switch {
	case held >= TxErrorHoldDuration:
		logx.Errorf("事务持有 %v（%d 个步骤）已超过 %v，请检查事务内是否存在外部调用或慢 SQL", held, funcCount, TxErrorHoldDuration)
	case held >= TxWarnHoldDuration:
		logx.Warnf("事务持有 %v（%d 个步骤）偏长，请检查事务内是否有非必要的外部调用", held, funcCount)
	}
}

// reportAfterCommitHold 上报提交后动作的耗时。
//
// 与事务持有时间分开统计：这段时长里事务已经提交、行锁与连接都已释放，它慢不代表事务慢。
// 但它同样需要可观测——动作里通常是对目标实例的远程调用，卡在那里会直接拖长请求耗时
func reportAfterCommitHold(held time.Duration, err error) {
	if err != nil {
		logx.Errorf("提交后动作执行失败（耗时 %v），事务已提交不可回滚，请按动作自身语义补偿：%v", held, err)
		return
	}
	if held >= TxErrorHoldDuration {
		logx.Warnf("提交后动作耗时 %v，已直接计入请求耗时，建议改为异步执行", held)
	}
}
