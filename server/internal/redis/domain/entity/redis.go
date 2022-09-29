package entity

import (
	"mayfly-go/internal/common/utils"
	"mayfly-go/pkg/model"
)

type Redis struct {
	model.Model

	Host               string `orm:"column(host)" json:"host"`
	Mode               string `json:"mode"`
	Password           string `orm:"column(password)" json:"-"`
	Db                 string `orm:"column(database)" json:"db"`
	EnableSshTunnel    int8   `orm:"column(enable_ssh_tunnel)" json:"enableSshTunnel"`        // 是否启用ssh隧道
	SshTunnelMachineId uint64 `orm:"column(ssh_tunnel_machine_id)" json:"sshTunnelMachineId"` // ssh隧道机器id
	Remark             string
	ProjectId          uint64
	Project            string
	EnvId              uint64
	Env                string
}

const (
	RedisModeStandalone = "standalone"
	RedisModeCluster    = "cluster"
	RedisModeSentinel   = "sentinel"
)

func (r *Redis) PwdEncrypt() {
	// 密码替换为加密后的密码
	r.Password = utils.PwdAesEncrypt(r.Password)
}

func (r *Redis) PwdDecrypt() {
	// 密码替换为解密后的密码
	r.Password = utils.PwdAesDecrypt(r.Password)
}
