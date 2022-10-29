package persistence

import (
	"fmt"
	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/internal/mongo/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils"
	"strings"
)

type mongoRepoImpl struct{}

func newMongoRepo() repository.Mongo {
	return new(mongoRepoImpl)
}

// 分页获取数据库信息列表
func (d *mongoRepoImpl) GetList(condition *entity.MongoQuery, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	sql := "SELECT d.* FROM t_mongo d WHERE 1=1  "

	if len(condition.TagIds) > 0 {
		sql = sql + " AND d.tag_id IN " + fmt.Sprintf("(%s)", strings.Join(utils.NumberArr2StrArr(condition.TagIds), ","))
	}
	if condition.TagPathLike != "" {
		sql = sql + " AND d.tag_path LIKE '" + condition.TagPathLike + "%'"
	}
	sql = sql + " ORDER BY d.tag_path"
	return model.GetPageBySql(sql, pageParam, toEntity)
}

func (d *mongoRepoImpl) Count(condition *entity.MongoQuery) int64 {
	where := make(map[string]interface{})
	if len(condition.TagIds) > 0 {
		where["tag_id"] = condition.TagIds
	}
	if condition.TagId != 0 {
		where["tag_id"] = condition.TagId
	}
	return model.CountByMap(new(entity.Mongo), where)
}

// 根据条件获取
func (d *mongoRepoImpl) Get(condition *entity.Mongo, cols ...string) error {
	return model.GetBy(condition, cols...)
}

// 根据id获取
func (d *mongoRepoImpl) GetById(id uint64, cols ...string) *entity.Mongo {
	db := new(entity.Mongo)
	if err := model.GetById(db, id, cols...); err != nil {
		return nil

	}
	return db
}

func (d *mongoRepoImpl) Insert(db *entity.Mongo) {
	biz.ErrIsNil(model.Insert(db), "新增mongo信息失败")
}

func (d *mongoRepoImpl) Update(db *entity.Mongo) {
	biz.ErrIsNil(model.UpdateById(db), "更新mongo信息失败")
}

func (d *mongoRepoImpl) Delete(id uint64) {
	model.DeleteById(new(entity.Mongo), id)
}
