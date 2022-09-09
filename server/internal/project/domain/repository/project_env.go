package repository

import "mayfly-go/internal/project/domain/entity"

type ProjectEnv interface {
	// 获取项目环境列表
	ListEnv(condition *entity.ProjectEnv, toEntity interface{}, orderBy ...string)

	Save(entity *entity.ProjectEnv)

	DeleteEnvs(projectId uint64)

	DeleteEnv(envId uint64)
}
