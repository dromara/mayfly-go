package api

import (
	dbapp "mayfly-go/internal/db/application"
	dbentity "mayfly-go/internal/db/domain/entity"
	machineapp "mayfly-go/internal/machine/application"
	machineentity "mayfly-go/internal/machine/domain/entity"
	mongoapp "mayfly-go/internal/mongo/application"
	mongoentity "mayfly-go/internal/mongo/domain/entity"
	redisapp "mayfly-go/internal/redis/application"
	redisentity "mayfly-go/internal/redis/domain/entity"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Index struct {
	TagApp     tagapp.TagTree
	MachineApp machineapp.Machine
	DbApp      dbapp.Db
	RedisApp   redisapp.Redis
	MongoApp   mongoapp.Mongo
}

func (i *Index) Count(rc *req.Ctx) {
	accountId := rc.GetLoginAccount().Id
	tagIds := i.TagApp.ListTagIdByAccountId(accountId)

	var mongoNum int64
	var redisNum int64
	var dbNum int64
	var machienNum int64

	if len(tagIds) > 0 {
		mongoNum = i.MongoApp.Count(&mongoentity.MongoQuery{TagIds: tagIds})
		machienNum = i.MachineApp.Count(&machineentity.MachineQuery{TagIds: tagIds})
		dbNum = i.DbApp.Count(&dbentity.DbQuery{TagIds: tagIds})
		redisNum = i.RedisApp.Count(&redisentity.RedisQuery{TagIds: tagIds})
	}
	rc.ResData = collx.M{
		"mongoNum":   mongoNum,
		"machineNum": machienNum,
		"dbNum":      dbNum,
		"redisNum":   redisNum,
	}
}
