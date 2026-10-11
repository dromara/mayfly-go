---
trigger: always_on
---

# Infrastructure 层规范

## Repository 实现

```go
package persistence

type dbRepoImpl struct {
    base.RepoImpl[*entity.Db]
}

func newDbRepo() repository.Db {
    return &dbRepoImpl{}
}

// 写入口一律以 ctx 为首参（连接与事务只经 ctx 传播）。读入口目前仍是一批不收 ctx 的历史方法
// （GetById/GetByCond/SelectByCond/PageByCond/CountByCond）：它们走全局连接池，既看不到本事务
// 尚未提交的写入（正确性问题：查重与差异比较会按旧数据放行），也会在事务占满连接池时等不到
// 空闲连接（元数据库默认是本地文件库，部署把 max-open-conns 调小后就是长时间不返回）。
// 因此两条硬性约束：
//   1. 需要事务内可见性的判断（如查重）用收 ctx 的入口（Exists），不要用历史读；
//   2. 历史读不得写在 `Tx` 事务闭包里，要么提到事务之前，要么用 AfterCommit 放到提交后执行
//      （架构守卫 TestNoCtxlessReadsInsideTxClosure 会拦住新增违规）
//
// 本示例是一个普通列表投影读，因此不收 ctx：它不要求事务内可见性，也不会出现在事务里
func (d *dbListRepoImpl) GetDbList(condition *entity.DbQuery, pageParam model.PageParam) (*model.PageResult[*entity.DbListPO], error) {
    pd := model.NewCond().
        Eq("instance_id", condition.InstanceId). // 可选过滤：值为零值时自动跳过
        In("code", condition.Codes).
        Like("name", condition.Name).
        OrderByDesc("id")

    // 投影到 PO 走 Query 的分页查询：目标切片决定查回来的类型，
    // 基座接口只承载实体形态，不为投影退回 any
    return gormx.NewQuery(d.GetModel(), pd).PageQuery(pageParam, []entity.DbListPO{})
}
```

## GORMX 常用操作

连接与事务只经 `ctx` 传播：`gormx.DB(ctx)` 是「用哪条连接」的唯一决策点，
因此基座与业务层都不存在 `XxxWithDb` 形式的方法变体。

```go
// 条件构建（条件按追加顺序保存，同一条逻辑查询生成的 SQL 文本恒定）
pd := model.NewCond().Eq("status", 1).In("id", ids).Like("name", keyword)

// 单条查询：结果由 dest 承担（历史入口，无 ctx）
err := repo.GetByCond(pd) // 未找到时返回包装了 model.ErrNotFound 的错误，用 model.IsNotFound 判定

// 分页查询
page, err := repo.PageByCond(pd, pageParam)

// 存在性判断：带 ctx，因此在事务内能看到自己尚未提交的写入
// 「先查重再写入」必须用它，不要用 CountByCond(...) > 0（读不到本事务的写入）
exists, err := repo.Exists(ctx, model.NewCond().Eq("name", name))

// 统计：SQL 出错必须回报，不与「统计为 0」混淆
count, err := repo.CountByCond(pd)

// 条件更新：回报真正被改动的行数（用于带前置状态条件的 CAS）
// 判读边界：未开 clientFoundRows，把相同值写回去也是 0 行，
// 只有 SET 的值必然不同于旧值时（如条件带 status=旧状态），0 行才等价于前置条件不成立
rows, err := repo.UpdateByCond(ctx, &entity.Db{Status: newStatus},
    model.NewCond().Eq("id", id).Eq("status", oldStatus))

// 需要写零值（清空字段）或数据库侧原子自增时，改用更新补丁
err := app.UpdateColsByCond(ctx,
    base.NewPatch().Set("remark", "").Inc("call_count", 1),
    model.NewCond().Eq("id", id))

// 撤销逻辑删除（回收站恢复、重新启用同名记录）：必须声明包含已删除数据
err := app.RestoreByCond(ctx,
    model.NewCond().IncludeDeleted().Eq("target_type", targetType).Eq("target_id", id))
```

## 硬性约束

- **列名与排序片段不可参数化**（它们是标识符而非值），因此 `OrderBy` 只接受
  「列名[.列名] [ASC|DESC]」，`Patch` 的列名先过白名单再按 schema 校验；非法输入在生成
  SQL 之前就被拒绝，不会拼进语句。
- **逻辑删除列与主键不可通过 `Patch` 改写**，基座直接返回 `ErrReservedPatchColumn`。
- **修改审计只由基座一处决定，且两条路口径一致**：实体形态由 `fillBaseInfo` 走更新分支填充
  （条件更新在副本上必须先置非零主键再填审计，否则 `IsCreate()` 为真会把创建人与创建时间也写进
  SET）；map/补丁形态由 `gormx.UpdateByCond` 补齐，并按实体真实拥有的列过滤（`CreateModelNLD`
  没有 `update_time`，无条件写入会生成不存在的列）。`Patch` 自身不再补审计，所校验的列与最终写进
  SQL 的列是同一份。仓储层**不对 map 形态二次补审计**：只有 `gormx.UpdateByCond` 知道该表真实有哪些
  审计列，也只有它守「不覆盖调用方已显式写明的审计列」这条约定（历史数据回填、导入类调用方需要自己
  给时间与操作人）；两处都补会让同一条约定在实体路径成立、在 map 路径失效。`UpdateByCond` 也不就地
  改写入参 map（它是导出函数），归一成副本后再补。
- **破坏性操作必须带调用方声明的条件**：`DeleteByCond`/`UpdateByCond`/`UpdateColsByCond`/
  `RestoreByCond` 在生成 SQL 之前先过 `gormx.ErrCondRequired`。「有没有条件」只看调用方声明的部分
  （条件片段，或条件模型里的非零字段），基座自动附加的软删过滤不算——拿它当「有条件」会让整表操作
  看起来合法。不依赖驱动层的全表操作拦截：那是实现细节，`AllowGlobalUpdate` 一开即失效，且报错文案
  说不清是调用方漏传了条件。确实需要整表操作时走显式的裸 SQL 入口，让意图写在代码里。
- **外部副作用写在 `base.AfterCommit(ctx, action)` 里，不写在事务闭包里**：动作在提交成功后才执行，
  回滚路径一个都不执行；动作 ctx 由基座自己剥离事务连接与取消信号，调用方不用管。动作失败**不回滚
  数据**，错误以 `ErrAfterCommit` 包装（`errors.Is` 可判），调用方只能按该动作自身的语义补偿，不得
  当成整体失败重试——那会把已生效的副作用再做一遍。动作里的 panic 由基座收敛成 error，不会从已提交
  的事务里穿出去。事务持有时间看门狗在提交那一刻定格，提交后动作的耗时另行统计：这段时长里行锁与
  连接都已释放，把它算进「事务持有时间」会让告警长期假阳性。
- **自动附加的软删过滤不占用调用方的条件集合**：调用方已显式就 `is_deleted` 表达过条件，
  或声明了 `IncludeDeleted()` 时不再自动附加，因此「查已删除数据」有确定入口。
- **基座不提供 `XxxWithDb` 变体，也不提供 `(bool, error)` 式的「按主键更新且判断改没改到」入口**：
  后者在本仓的 changed-rows 口径下会把「行存在但值未变」判成「行不存在」。需要强一致存在性时用
  `Exists(ctx, ...)`（事务内可见自己的写入），需要并发安全流转时把前置状态写进更新条件并按行数判赢。
- **集合类条件区分「没传」与「传了空集合」**：`In(nil)` 跳过条件，`In(空非nil切片)` 生成恒假
  条件（绝不退化为放开全表）；`NotIn` 两种情况都跳过（没有排除项即不排除任何行）。
