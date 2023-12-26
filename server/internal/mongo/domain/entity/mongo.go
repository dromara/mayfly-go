package entity

import (
	"mayfly-go/internal/mongo/mgm"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/structx"
)

type Mongo struct {
	model.Model

	Code               string `orm:"column(code)" json:"code"`
	Name               string `orm:"column(name)" json:"name"`
	Uri                string `orm:"column(uri)" json:"uri"`
	SshTunnelMachineId int    `orm:"column(ssh_tunnel_machine_id)" json:"sshTunnelMachineId"` // ssh隧道机器id
}

// 转换为mongoInfo进行连接
func (me *Mongo) ToMongoInfo(tagPath ...string) *mgm.MongoInfo {
	mongoInfo := new(mgm.MongoInfo)
	structx.Copy(mongoInfo, me)
	mongoInfo.TagPath = tagPath
	return mongoInfo
}
