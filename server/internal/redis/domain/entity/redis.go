package entity

import (
	"errors"
	"mayfly-go/internal/common/utils"
	"mayfly-go/internal/redis/rdm"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/structx"
)

type Redis struct {
	model.Model

	Code               string `orm:"column(code)" json:"code"`
	Name               string `orm:"column(name)" json:"name"`
	Host               string `orm:"column(host)" json:"host"`
	Mode               string `json:"mode"`
	Username           string `json:"username"`
	Password           string `orm:"column(password)" json:"-"`
	Db                 string `orm:"column(database)" json:"db"`
	SshTunnelMachineId int    `orm:"column(ssh_tunnel_machine_id)" json:"sshTunnelMachineId"` // ssh隧道机器id
	Remark             string
}

func (r *Redis) PwdEncrypt() error {
	// 密码替换为加密后的密码
	password, err := utils.PwdAesEncrypt(r.Password)
	if err != nil {
		return errors.New("加密 Redis 密码失败")
	}
	r.Password = password
	return nil
}

func (r *Redis) PwdDecrypt() error {
	// 密码替换为解密后的密码
	password, err := utils.PwdAesDecrypt(r.Password)
	if err != nil {
		return errors.New("解密 Redis 密码失败")
	}
	r.Password = password
	return nil
}

// ToRedisInfo 转换为redisInfo进行连接
func (r *Redis) ToRedisInfo(db int, tagPath ...string) *rdm.RedisInfo {
	redisInfo := new(rdm.RedisInfo)
	_ = structx.Copy(redisInfo, r)
	redisInfo.Db = db
	redisInfo.TagPath = tagPath
	return redisInfo
}
