package entity

import (
	"mayfly-go/pkg/model"
)

// DbInstance 数据库实例信息
type DbInstance struct {
	model.Model
	model.ExtraData // 连接需要的其他额外参数（json字符串），如oracle数据库需要指定sid等

	Code               string  `json:"code" gorm:"size:32;not null;"`
	Name               string  `json:"name" gorm:"size:32;not null;"`
	Type               string  `json:"type" gorm:"size:32;not null;"` // 类型，mysql oracle等
	Host               string  `json:"host" gorm:"size:255;not null;"`
	Port               int     `json:"port"`
	Network            string  `json:"network" gorm:"size:20;"`
	Params             *string `json:"params" gorm:"size:255;comment:其他连接参数"` // 使用指针类型，可更新为零值（空字符串）
	Remark             *string `json:"remark" gorm:"size:255;"`
	SshTunnelMachineId int     `json:"sshTunnelMachineId"` // ssh隧道机器id

	// TLSMaterialEnc 证书材料密文（CA/客户端证书/私钥的 JSON 整体 AES 加密）。
	//
	// 与「档位/主机名/指纹」分开放：后者是可读配置，进 Extra；
	// 材料单个 3~5KB，Extra 是 varchar(2000) 装不下（同 db_transfer.go 中
	// 「表名清单独立成表而非进 Extra」的既有判断）。
	// json:"-" 必须保留：GetInstance 直接返回实体，否则密文会随详情接口下发。
	TLSMaterialEnc string `json:"-" gorm:"column:tls_material;type:mediumtext;comment:TLS证书材料密文"`
}

// 实例 TLS 的可读配置项（证书材料走 TLSMaterialEnc 列）。
// disable 以「键不存在」表达：基座 SetExtraValue 对零值即删键，缺键读出来就是空串。
const (
	ExtraKeyTLSMode        = "tlsMode"        // 档位：require / verify-ca / verify-full
	ExtraKeyTLSServerName  = "tlsServerName"  // 覆盖 SNI 与主机名校验（默认取原始 host）
	ExtraKeyTLSUseSystemCA = "tlsUseSystemCA" // 使用系统根证书（pg 的 verify-* 无自签 CA 时）
	ExtraKeyTLSFingerprint = "tlsFingerprint" // 当前材料指纹（展示 + 轮换判定）
)

func (d *DbInstance) TableName() string {
	return "t_db_instance"
}
