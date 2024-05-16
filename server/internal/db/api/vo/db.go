package vo

import (
	"mayfly-go/internal/db/domain/entity"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"time"
)

type DbListVO struct {
	tagentity.ResourceTags

	Id              *int64                   `json:"id"`
	Code            string                   `json:"code"`
	Name            *string                  `json:"name"`
	GetDatabaseMode entity.DbGetDatabaseMode `json:"getDatabaseMode"` // 获取数据库方式
	Database        *string                  `json:"database"`
	Remark          *string                  `json:"remark"`

	InstanceId   *int64  `json:"instanceId"`
	AuthCertName string  `json:"authCertName"`
	InstanceName *string `json:"instanceName"`
	InstanceType *string `json:"type"`
	Host         string  `json:"host"`
	Port         int     `json:"port"`

	CreateTime *time.Time `json:"createTime"`
	Creator    *string    `json:"creator"`
	CreatorId  *int64     `json:"creatorId"`
	UpdateTime *time.Time `json:"updateTime"`
	Modifier   *string    `json:"modifier"`
	ModifierId *int64     `json:"modifierId"`
}

func (d DbListVO) GetCode() string {
	return d.Code
}
