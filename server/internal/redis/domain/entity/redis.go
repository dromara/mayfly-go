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

	Code               string `json:"code" gorm:"size:32;not null;"`                // code
	Name               string `json:"name" gorm:"size:255;not null;"`               // 名称
	Host               string `json:"host" gorm:"size:255;not null;"`               // 主机地址
	Mode               string `json:"mode" gorm:"size:32;"`                         // 模式
	Db                 string `json:"db" gorm:"size:64;comment:库号: 多个库用,分割"`        // 库号: 多个库用,分割
	SshTunnelMachineId int    `json:"sshTunnelMachineId" gorm:"comment:ssh隧道的机器id"` // ssh隧道机器id
	Remark             string `json:"remark" gorm:"size:255;"`
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
