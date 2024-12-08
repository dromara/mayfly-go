package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type syslogRepoImpl struct {
	base.RepoImpl[*entity.SysLog]
}

func newSyslogRepo() repository.Syslog {
	return &syslogRepoImpl{}
}

func (m *syslogRepoImpl) GetPageList(condition *entity.SysLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := model.NewCond().Like("description", condition.Description).
		Eq("creator_id", condition.CreatorId).Eq("type", condition.Type).OrderBy(orderBy...)
	return m.PageByCondToAny(qd, pageParam, toEntity)
}
