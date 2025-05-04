package vo

import (
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
	"time"
)

type InstanceListVO struct {
	model.ExtraData
	tagentity.AuthCerts // 授权凭证信息
	tagentity.ResourceTags

	Id         *int64     `json:"id"`
	Code       string     `json:"code"`
	Name       *string    `json:"name"`
	Host       *string    `json:"host"`
	Port       *int       `json:"port"`
	Type       *string    `json:"type"`
	Params     string     `json:"params"`
	Remark     *string    `json:"remark"`
	CreateTime *time.Time `json:"createTime"`
	Creator    *string    `json:"creator"`
	CreatorId  *int64     `json:"creatorId"`

	UpdateTime *time.Time `json:"updateTime"`
	Modifier   *string    `json:"modifier"`
	ModifierId *int64     `json:"modifierId"`

	SshTunnelMachineId int `json:"sshTunnelMachineId"`
}

func (i *InstanceListVO) GetCode() string {
	return i.Code
}
