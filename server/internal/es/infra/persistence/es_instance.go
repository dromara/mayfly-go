package persistence

import (
	"mayfly-go/internal/es/domain/entity"
	"mayfly-go/internal/es/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type instanceRepoImpl struct {
	base.RepoImpl[*entity.EsInstance]
}

func NewInstanceRepo() repository.EsInstance {
	return &instanceRepoImpl{}
}

// 分页获取数据库信息列表
func (d *instanceRepoImpl) GetInstanceList(condition *entity.InstanceQuery, orderBy ...string) (*model.PageResult[*entity.EsInstance], error) {
	qd := model.NewCond().
		Eq("id", condition.Id).
		Eq("host", condition.Host).
		Like("name", condition.Name).
		Like("code", condition.Code).
		In("code", condition.Codes)

	keyword := condition.Keyword
	if keyword != "" {
		keyword = "%" + keyword + "%"
		qd.And("host like ? or name like ? or code like ?", keyword, keyword, keyword)
	}

	return d.PageByCond(qd, condition.PageParam)
}
