package persistence

import (
	"fmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils"
	"strings"
)

type dbRepoImpl struct{}

func newDbRepo() repository.Db {
	return new(dbRepoImpl)
}

// 分页获取数据库信息列表
func (d *dbRepoImpl) GetDbList(condition *entity.DbQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult {
	sql := "SELECT d.* FROM t_db d WHERE 1 = 1 "

	values := make([]any, 0)
	if condition.Host != "" {
		sql = sql + " AND d.host LIKE ?"
		values = append(values, "%"+condition.Host+"%")
	}
	if condition.Database != "" {
		sql = sql + " AND d.database LIKE ?"
		values = append(values, "%"+condition.Database+"%")
	}
	if len(condition.TagIds) > 0 {
		sql = sql + " AND d.tag_id IN " + fmt.Sprintf("(%s)", strings.Join(utils.NumberArr2StrArr(condition.TagIds), ","))
	}
	if condition.TagPathLike != "" {
		sql = sql + " AND d.tag_path LIKE ?"
		values = append(values, "%"+condition.TagPathLike+"%")
	}
	sql = sql + " ORDER BY d.tag_path"
	return model.GetPageBySql(sql, pageParam, toEntity, values...)
}

func (d *dbRepoImpl) Count(condition *entity.DbQuery) int64 {
	where := make(map[string]any)
	if len(condition.TagIds) > 0 {
		where["tag_id"] = condition.TagIds
	}
	if condition.TagId != 0 {
		where["tag_id"] = condition.TagId
	}
	return model.CountByMap(new(entity.Db), where)
}

// 根据条件获取账号信息
func (d *dbRepoImpl) GetDb(condition *entity.Db, cols ...string) error {
	return model.GetBy(condition, cols...)
}

// 根据id获取
func (d *dbRepoImpl) GetById(id uint64, cols ...string) *entity.Db {
	db := new(entity.Db)
	if err := model.GetById(db, id, cols...); err != nil {
		return nil

	}
	return db
}

func (d *dbRepoImpl) Insert(db *entity.Db) {
	biz.ErrIsNil(model.Insert(db), "新增数据库信息失败")
}

func (d *dbRepoImpl) Update(db *entity.Db) {
	biz.ErrIsNil(model.UpdateById(db), "更新数据库信息失败")
}

func (d *dbRepoImpl) Delete(id uint64) {
	model.DeleteById(new(entity.Db), id)
}
