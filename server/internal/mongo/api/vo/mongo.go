package vo

import (
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
)

type Mongo struct {
	model.Model
	tagentity.ResourceTags

	Code               string `orm:"column(code)" json:"code"`
	Name               string `orm:"column(name)" json:"name"`
	Uri                string `orm:"column(uri)" json:"uri"`
	SshTunnelMachineId int    `orm:"column(ssh_tunnel_machine_id)" json:"sshTunnelMachineId"` // ssh隧道机器id
}

func (m *Mongo) GetCode() string {
	return m.Code
}
