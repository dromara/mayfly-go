package persistence

import (
	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/internal/mongo/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type mongoRepoImpl struct {
	base.RepoImpl[*entity.Mongo]
}

func newMongoRepo() repository.Mongo {
	return &mongoRepoImpl{}
}

// 分页获取数据库信息列表
func (d *mongoRepoImpl) GetList(condition *entity.MongoQuery, orderBy ...string) (*model.PageResult[*entity.Mongo], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		Eq("code", condition.Code).
		In("code", condition.Codes)

	keyword := condition.Keyword
	if keyword != "" {
		keyword = "%" + keyword + "%"
		qd.And("name like ? or code like ?", keyword, keyword)
	}
	return d.PageByCond(qd, condition.PageParam)
}
