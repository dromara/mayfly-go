package dto

import (
	"mayfly-go/internal/redis/domain/entity"
	tagentity "mayfly-go/internal/tag/domain/entity"
)

type SaveRedis struct {
	Redis        *entity.Redis
	AuthCert     *tagentity.ResourceAuthCert
	TagCodePaths []string
}

type RunCmd struct {
	Id     uint64 `json:"id"`
	Db     int    `json:"db"`
	Cmd    []any  `json:"cmd"`
	Remark string
}
