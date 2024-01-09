package model

import (
	"time"
)

type IdGenType int

const (
	IdColumn         = "id"
	DeletedColumn    = "is_deleted" // 删除字段
	DeleteTimeColumn = "delete_time"

	ModelDeleted   int8 = 1
	ModelUndeleted int8 = 0

	IdGenTypeNone      IdGenType = 0 // 数据库处理
	IdGenTypeTimestamp IdGenType = 1 // 当前时间戳
)

// 实体接口
type ModelI interface {

	// 是否为新建该实体模型, 默认 id == 0 为新建
	IsCreate() bool

	// 使用当前登录账号信息赋值实体结构体的基础信息
	//
	// 如创建时间，修改时间，创建者，修改者信息等
	FillBaseInfo(idGenType IdGenType, account *LoginAccount)
}

type IdModel struct {
	Id uint64 `json:"id"`
}

func (m *IdModel) IsCreate() bool {
	return m.Id == 0
}

func (m *IdModel) FillBaseInfo(idGenType IdGenType, account *LoginAccount) {
	// 存在id，则赋值
	if !m.IsCreate() {
		return
	}
	m.Id = GetIdByGenType(idGenType)
}

// 含有删除字段模型
type DeletedModel struct {
	IdModel
	IsDeleted  int8       `json:"-" gorm:"column:is_deleted;default:0"`
	DeleteTime *time.Time `json:"-"`
}

func (m *DeletedModel) FillBaseInfo(idGenType IdGenType, account *LoginAccount) {
	if m.Id == 0 {
		m.IdModel.FillBaseInfo(idGenType, account)
		m.IsDeleted = ModelUndeleted
	}
}

// 含有删除、创建字段模型
type CreateModel struct {
	DeletedModel
	CreateTime *time.Time `json:"createTime"`
	CreatorId  uint64     `json:"creatorId"`
	Creator    string     `json:"creator"`
}

func (m *CreateModel) FillBaseInfo(idGenType IdGenType, account *LoginAccount) {
	if !m.IsCreate() {
		return
	}

	m.DeletedModel.FillBaseInfo(idGenType, account)
	nowTime := time.Now()
	m.CreateTime = &nowTime
	if account != nil {
		m.CreatorId = account.Id
		m.Creator = account.Username
	}
}

// 基础实体模型，数据表最基础字段，尽量每张表都包含这些字段
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
func (m *Model) FillBaseInfo(idGenType IdGenType, account *LoginAccount) {
	nowTime := time.Now()
	isCreate := m.IsCreate()
	if isCreate {
		m.IsDeleted = ModelUndeleted
		m.CreateTime = &nowTime
		m.IdModel.FillBaseInfo(idGenType, account)
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

// 根据id生成类型，生成id
func GetIdByGenType(genType IdGenType) uint64 {
	if genType == IdGenTypeTimestamp {
		return uint64(time.Now().Unix())
	}
	return 0
}
