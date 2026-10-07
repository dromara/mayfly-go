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

func (d *dbRepoImpl) GetDbList(condition *entity.DbQuery, orderBy ...string) (*model.PageResult[*entity.DbListPO], error) {
    pd := model.NewCond().
        Eq("instance_id", condition.InstanceId).
        In("code", condition.Codes).
        Like("name", condition.Name)

    list := []*entity.DbListPO{}
    return gormx.PageByCond(d.GetModel(), pd, condition.PageParam, list)
}
```

## GORMX 常用操作

```go
// 条件构建
pd := model.NewCond().Eq("status", 1).In("id", ids).Like("name", keyword)

// 分页查询
result, err := gormx.PageByCond(repo.GetModel(), pd, pageParam, &list)

// 单条查询
err := gormx.GetByCond(repo.GetModel(), pd, &entity)

// 更新（回报真正被改动的行数，用于带前置状态条件的 CAS）
rows, err := gormx.UpdateByCond(repo.GetModel(), values, pd)
```
