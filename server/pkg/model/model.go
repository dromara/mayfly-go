package model

import (
	"time"
)

const (
	IdColumn              = "id"
	DeletedColumn         = "is_deleted" // 删除字段
	DeleteTimeColumn      = "delete_time"
	ModelDeleted     int8 = 1
	ModelUndeleted   int8 = 0
)

// 含有删除字段模型
type DeletedModel struct {
	Id         uint64     `json:"id"`
	IsDeleted  int8       `json:"-" gorm:"column:is_deleted;default:0"`
	DeleteTime *time.Time `json:"-"`
}

// 基础实体模型，数据表最基础字段，每张表必备字段
type Model struct {
	DeletedModel

	CreateTime *time.Time `json:"createTime"`
	CreatorId  uint64     `json:"creatorId"`
	Creator    string     `json:"creator"`
	UpdateTime *time.Time `json:"updateTime"`
	ModifierId uint64     `json:"modifierId"`
	Modifier   string     `json:"modifier"`
}

// 设置基础信息. 如创建时间，修改时间，创建者，修改者信息
func (m *Model) SetBaseInfo(account *LoginAccount) {
	nowTime := time.Now()
	isCreate := m.Id == 0
	if isCreate {
		m.IsDeleted = ModelUndeleted
		m.CreateTime = &nowTime
	}
	m.UpdateTime = &nowTime

	if account == nil {
		return
	}
	id := account.Id
	name := account.Username
	if isCreate {
		m.CreatorId = id
		m.Creator = name
	}
	m.Modifier = name
	m.ModifierId = id
}
