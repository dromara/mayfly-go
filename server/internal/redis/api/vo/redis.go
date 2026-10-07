package vo

import (
	"time"

	"mayfly-go/internal/redis/domain/entity"
)

type Redis struct {
	Id                 *int64     `json:"id"`
	Code               string     `json:"code"`
	Name               *string    `json:"name"`
	Host               *string    `json:"host"`
	Db                 string     `json:"db"`
	Mode               *string    `json:"mode"`
	SshTunnelMachineId int        `json:"sshTunnelMachineId"` // ssh隧道机器id
	Remark             *string    `json:"remark"`
	CreateTime         *time.Time `json:"createTime"`
	Creator            *string    `json:"creator"`
	CreatorId          *int64     `json:"creatorId"`
	UpdateTime         *time.Time `json:"updateTime"`
	Modifier           *string    `json:"modifier"`
	ModifierId         *int64     `json:"modifierId"`
}

func (r *Redis) GetCode() string {
	return r.Code
}

type Keys struct {
	Cursor map[string]uint64 `json:"cursor"`
	Keys   []string          `json:"keys"`
	DbSize int64             `json:"dbSize"`
	// 本批 key 的类型/过期摘要：树角标随扫描批次就位，前端无需再发独立摘要请求
	Summaries []*entity.KeySummary `json:"summaries"`
}
