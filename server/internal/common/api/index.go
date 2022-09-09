package api

import (
	dbapp "mayfly-go/internal/db/application"
	dbentity "mayfly-go/internal/db/domain/entity"
	machineentity "mayfly-go/internal/machine/domain/entity"
	projectentity "mayfly-go/internal/project/domain/entity"
	redisentity "mayfly-go/internal/redis/domain/entity"

	machineapp "mayfly-go/internal/machine/application"
	projectapp "mayfly-go/internal/project/application"
	redisapp "mayfly-go/internal/redis/application"
	"mayfly-go/pkg/ctx"
)

type Index struct {
	ProjectApp projectapp.Project
	MachineApp machineapp.Machine
	DbApp      dbapp.Db
	RedisApp   redisapp.Redis
}

func (i *Index) Count(rc *ctx.ReqCtx) {
	rc.ResData = map[string]interface{}{
		"projectNum": i.ProjectApp.Count(new(projectentity.Project)),
		"machineNum": i.MachineApp.Count(new(machineentity.Machine)),
		"dbNum":      i.DbApp.Count(new(dbentity.Db)),
		"redisNum":   i.RedisApp.Count(new(redisentity.Redis)),
	}
}
