package vo

import (
	"mayfly-go/internal/db/domain/entity"
	"time"
)

type DbListVO struct {
	Id              *int64                   `json:"id"`
	Code            string                   `json:"code"`
	Name            *string                  `json:"name"`
	GetDatabaseMode entity.DbGetDatabaseMode `json:"getDatabaseMode"` // 获取数据库方式
	Database        *string                  `json:"database"`
	Remark          *string                  `json:"remark"`
	InstanceId      uint64                   `json:"instanceId"`
	AuthCertName    string                   `json:"authCertName"`

	InstanceCode string `json:"instanceCode" gorm:"-"`
	InstanceType string `json:"type" gorm:"-"`
	Host         string `json:"host" gorm:"-"`
	Port         int    `json:"port" gorm:"-"`

	CreateTime *time.Time `json:"createTime"`
	Creator    *string    `json:"creator"`
	CreatorId  *int64     `json:"creatorId"`
	UpdateTime *time.Time `json:"updateTime"`
	Modifier   *string    `json:"modifier"`
	ModifierId *int64     `json:"modifierId"`
}
