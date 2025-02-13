package entity

import (
	"mayfly-go/internal/mongo/mgm"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/structx"
)

type Mongo struct {
	model.Model

	Code               string `json:"code" gorm:"size:32;comment:code"`             // code
	Name               string `json:"name" gorm:"not null;size:50;comment:名称"`      // 名称
	Uri                string `json:"uri" gorm:"not null;size:255;comment:连接uri"`   // 连接uri
	SshTunnelMachineId int    `json:"sshTunnelMachineId" gorm:"comment:ssh隧道的机器id"` // ssh隧道机器id
}

// 转换为mongoInfo进行连接
func (me *Mongo) ToMongoInfo(tagPath ...string) *mgm.MongoInfo {
	mongoInfo := new(mgm.MongoInfo)
	structx.Copy(mongoInfo, me)
	mongoInfo.CodePath = tagPath
	return mongoInfo
}
