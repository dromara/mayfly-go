package api

import (
	"mayfly-go/internal/common/consts"
	dbapp "mayfly-go/internal/db/application"
	machineapp "mayfly-go/internal/machine/application"
	mongoapp "mayfly-go/internal/mongo/application"
	redisapp "mayfly-go/internal/redis/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Index struct {
	TagApp     tagapp.TagTree     `inject:"TagTreeApp"`
	MachineApp machineapp.Machine `inject:""`
	DbApp      dbapp.Db           `inject:""`
	RedisApp   redisapp.Redis     `inject:""`
	MongoApp   mongoapp.Mongo     `inject:""`
}

func (i *Index) Count(rc *req.Ctx) {
	accountId := rc.GetLoginAccount().Id

	mongoNum := len(i.TagApp.GetAccountResourceCodes(accountId, consts.TagResourceTypeMongo, ""))
	machienNum := len(i.TagApp.GetAccountResourceCodes(accountId, consts.TagResourceTypeMachine, ""))
	dbNum := len(i.TagApp.GetAccountResourceCodes(accountId, consts.TagResourceTypeDb, ""))
	redisNum := len(i.TagApp.GetAccountResourceCodes(accountId, consts.TagResourceTypeRedis, ""))

	rc.ResData = collx.M{
		"mongoNum":   mongoNum,
		"machineNum": machienNum,
		"dbNum":      dbNum,
		"redisNum":   redisNum,
	}
}
