package entity

import (
	"mayfly-go/internal/redis/rdm"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/structx"
)

type Redis struct {
	model.Model

	Code               string `orm:"column(code)" json:"code"`
	Name               string `orm:"column(name)" json:"name"`
	Host               string `orm:"column(host)" json:"host"`
	Mode               string `json:"mode"`
	Db                 string `orm:"column(database)" json:"db"`
	SshTunnelMachineId int    `orm:"column(ssh_tunnel_machine_id)" json:"sshTunnelMachineId"` // ssh隧道机器id
	Remark             string
	FlowProcdefKey     *string `json:"flowProcdefKey"` // 审批流-流程定义key（有值则说明关键操作需要进行审批执行）,使用指针为了方便更新空字符串(取消流程审批)
}

// ToRedisInfo 转换为redisInfo进行连接
func (r *Redis) ToRedisInfo(db int, authCert *tagentity.ResourceAuthCert, tagPath ...string) *rdm.RedisInfo {
	redisInfo := new(rdm.RedisInfo)
	_ = structx.Copy(redisInfo, r)
	redisInfo.Username = authCert.Username
	redisInfo.Password = authCert.Ciphertext
	redisInfo.Db = db
	redisInfo.TagPath = tagPath
	return redisInfo
}
