package entity

import (
	"mayfly-go/pkg/model"
)

// MachineHostKey 主机公钥指纹信任库。
//
// 以「原始目标地址 ip:port」为信任键而非机器 id：直连、跳板链与未落库的测试连接
// 都以地址定位目标主机；机器换址后旧记录成为孤儿数据，由管理页删除，
// 新地址重新走首次信任流程（对新地址而言没有历史记录，语义正确）。
//
// 不建唯一索引：信任记录随实体走软删除，唯一索引会与软删行冲突（撤销信任后
// 重新首次信任会撞历史行），查重由应用层负责。
type MachineHostKey struct {
	model.Model

	HostAddr    string `json:"hostAddr" gorm:"size:100;not null;index;comment:主机地址 ip:port"` // 信任键：连接改写前的原始目标地址
	KeyType     string `json:"keyType" gorm:"size:50;not null;comment:公钥类型"`                 // 如 ssh-ed25519、ssh-rsa
	Fingerprint string `json:"fingerprint" gorm:"size:100;not null;comment:SHA256指纹"`
	Remark      string `json:"remark" gorm:"size:255;comment:备注"`
}
