package vo

import "time"

type InstanceListVO struct {
	Id         *int64     `json:"id"`
	Name       *string    `json:"name"`
	Host       *string    `json:"host"`
	Port       *int       `json:"port"`
	Type       *string    `json:"type"`
	Params     *string    `json:"params"`
	Sid        *string    `json:"sid"`
	Username   *string    `json:"username"`
	Remark     *string    `json:"remark"`
	CreateTime *time.Time `json:"createTime"`
	Creator    *string    `json:"creator"`
	CreatorId  *int64     `json:"creatorId"`

	UpdateTime *time.Time `json:"updateTime"`
	Modifier   *string    `json:"modifier"`
	ModifierId *int64     `json:"modifierId"`

	SshTunnelMachineId int `json:"sshTunnelMachineId"`
}
