package vo

import (
	tagentity "mayfly-go/internal/tag/domain/entity"
	"time"
)

type Redis struct {
	tagentity.ResourceTags
	Id                 *int64     `json:"id"`
	Code               string     `json:"code"`
	Name               *string    `json:"name"`
	Host               *string    `json:"host"`
	Db                 string     `json:"db"`
	Mode               *string    `json:"mode"`
	SshTunnelMachineId int        `json:"sshTunnelMachineId"` // ssh隧道机器id
	Remark             *string    `json:"remark"`
	FlowProcdefKey     string     `json:"flowProcdefKey"`
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
}

type KeyInfo struct {
	Key  string `json:"key"`
	Ttl  int    `json:"ttl"`
	Type string `json:"type"`
}
