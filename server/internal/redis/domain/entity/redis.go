package entity

import (
	"mayfly-go/internal/common/utils"
	"mayfly-go/internal/redis/rdm"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/logx"
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
}

// ToRedisInfo 转换为redisInfo进行连接
func (r *Redis) ToRedisInfo(db int, authCert *tagentity.ResourceAuthCert, tagPath ...string) *rdm.RedisInfo {
	redisInfo := new(rdm.RedisInfo)
	_ = structx.Copy(redisInfo, r)
	redisInfo.Username = authCert.Username
	redisInfo.Password = authCert.Ciphertext
	redisInfo.Db = db
	redisInfo.CodePath = tagPath

	if redisInfo.Mode == rdm.SentinelMode {
		// // 密码替换为解密后的密码
		password, err := utils.PwdAesDecrypt(authCert.GetExtraString("redisNodePassword"))
		if err != nil {
			logx.Errorf("redis节点密码解密失败: %s", err.Error())
		} else {
			redisInfo.RedisNodePassword = password
		}
	}

	return redisInfo
}
