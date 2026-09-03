package persistence

import (
	"mayfly-go/internal/milvus/domain/entity"
	"mayfly-go/internal/milvus/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type milvusRepoImpl struct {
	base.RepoImpl[*entity.Milvus]
}

var _ repository.Milvus = (*milvusRepoImpl)(nil)

func newMilvusRepo() repository.Milvus {
	return &milvusRepoImpl{}
}

// 分页获取 Milvus 实例列表
func (r *milvusRepoImpl) GetList(condition *entity.MilvusQuery, orderBy ...string) (*model.PageResult[*entity.Milvus], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		Eq("code", condition.Code).
		In("code", condition.Codes)

	keyword := condition.Keyword
	if keyword != "" {
		keyword = "%" + keyword + "%"
		qd.And("(name like ? or code like ?)", keyword, keyword)
	}
	return r.PageByCond(qd, condition.PageParam)
}
