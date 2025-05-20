package entity

import "time"

type DbListPO struct {
	Id              *int64            `json:"id"`
	Code            string            `json:"code"`
	Name            *string           `json:"name"`
	GetDatabaseMode DbGetDatabaseMode `json:"getDatabaseMode"` // 获取数据库方式
	Database        *string           `json:"database"`
	Remark          *string           `json:"remark"`
	InstanceId      uint64            `json:"instanceId"`
	AuthCertName    string            `json:"authCertName"`
	CreateTime      *time.Time        `json:"createTime"`
	Creator         *string           `json:"creator"`
	CreatorId       *int64            `json:"creatorId"`
	UpdateTime      *time.Time        `json:"updateTime"`
	Modifier        *string           `json:"modifier"`
	ModifierId      *int64            `json:"modifierId"`
}
