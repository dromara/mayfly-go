package model

import (
	"time"
)

const (
	IdColumn         = "id"
	DeletedColumn    = "is_deleted" // 删除字段
	DeleteTimeColumn = "delete_time"
	ModifierColumn   = "modifier"
	ModifierIdColumn = "modifier_id"
	UpdateTimeColumn = "update_time"
	CreatorColumn    = "creator"
	CreatorIdColumn  = "creator_id"
	CreateTimeColumn = "create_time"

	ModelDeleted   int8 = 1
	ModelUndeleted int8 = 0
)

// 实体接口
type ModelI interface {
	// GetId 获取实体主键
	GetId() uint64

	// SetId 设置id
	SetId(id uint64)

	// IsCreate 是否为新建该实体模型, 默认 id == 0 为新建
	IsCreate() bool

	// FillBaseInfo 使用当前登录账号信息赋值实体结构体的基础信息
	//
	// 如创建时间，修改时间，创建者，修改者信息等。account 为 nil 表示无操作者上下文
	// （启动兜底、后台任务等）：此时创建人回落系统账号，修改人保持不动
	FillBaseInfo(account *LoginAccount)

	// LogicDelete 是否为逻辑删除
	LogicDelete() bool
}

// 各基础模型的「有哪些字段」由嵌入结构决定（保持既有嵌入链，实体与迁移脚本的字面量构造不受影响），
// 而「怎么填」只有下面 applyCreate/applyUpdate 两份实现。
//
// 此前六个基础模型各自写一份 FillBaseInfo，创建人回落、更新时间刷新、无操作者时是否动修改人这几条
// 规则在六份实现之间有细微分叉：选了哪个基类就等于选了哪套审计行为，且改一处容易漏改其余几处。

// createFields 可被填充创建信息的字段集合
type createFields interface {
	setCreate(createdAt time.Time, creatorId uint64, creator string)
}

// updateFields 可被填充修改信息的字段集合
// account 为 nil 表示无操作者上下文，由各模型决定保留还是覆盖修改人，不用零值去猜
type updateFields interface {
	setUpdate(updatedAt *time.Time, account *LoginAccount)
}

// applyCreate 填充创建信息：仅在新建时调用，无操作者上下文时创建人记为系统账号
func applyCreate(target createFields, account *LoginAccount) {
	if account == nil {
		account = SysAccount
	}
	target.setCreate(time.Now(), account.Id, account.Username)
}

// applyUpdate 填充修改信息。
//
// 更新时间总是刷新；没有操作者上下文时不动修改人——调用方常是「先查出实体再改几个字段」，
// 实体上带着的就是当前修改人，覆盖成空反而丢信息
func applyUpdate(target updateFields, account *LoginAccount) {
	now := time.Now()
	target.setUpdate(&now, account)
}

type IdModel struct {
	Id uint64 `json:"id" gorm:"primarykey;AUTO_INCREMENT"`
}

func (m *IdModel) GetId() uint64 {
	return m.Id
}

func (m *IdModel) SetId(id uint64) {
	m.Id = id
}

func (m *IdModel) IsCreate() bool {
	return m.Id == 0
}

func (m *IdModel) FillBaseInfo(account *LoginAccount) {
	// 主键由数据库自增生成，除主键外该模型没有其他基础信息可填充
}

func (m *IdModel) LogicDelete() bool {
	return false
}

// 含有删除字段模型
type DeletedModel struct {
	IdModel
	IsDeleted  int8       `json:"-" gorm:"column:is_deleted;not null;default:0;"`
	DeleteTime *time.Time `json:"-"`
}

func (m *DeletedModel) FillBaseInfo(account *LoginAccount) {
	if m.Id == 0 {
		m.IsDeleted = ModelUndeleted
	}
}

func (m *DeletedModel) LogicDelete() bool {
	return true
}

// CreateModelNLD 含有创建等信息，但不包含逻辑删除信息
type CreateModelNLD struct {
	IdModel
	CreateTime *time.Time `json:"createTime" gorm:"not null;"`
	CreatorId  uint64     `json:"creatorId" gorm:"not null;"`
	Creator    string     `json:"creator" gorm:"size:32;not null;"`
}

func (m *CreateModelNLD) setCreate(createdAt time.Time, creatorId uint64, creator string) {
	m.CreateTime = &createdAt
	m.CreatorId = creatorId
	m.Creator = creator
}

func (m *CreateModelNLD) FillBaseInfo(account *LoginAccount) {
	if !m.IsCreate() {
		return
	}
	applyCreate(m, account)
}

// 含有删除、创建字段模型
type CreateModel struct {
	DeletedModel
	CreateTime *time.Time `json:"createTime" gorm:"not null;"`
	CreatorId  uint64     `json:"creatorId" gorm:"not null;"`
	Creator    string     `json:"creator" gorm:"size:32;not null;"`
}

func (m *CreateModel) setCreate(createdAt time.Time, creatorId uint64, creator string) {
	m.CreateTime = &createdAt
	m.CreatorId = creatorId
	m.Creator = creator
}

func (m *CreateModel) FillBaseInfo(account *LoginAccount) {
	if !m.IsCreate() {
		return
	}
	m.IsDeleted = ModelUndeleted
	applyCreate(m, account)
}

// 基础实体模型，数据表最基础字段，不包含逻辑删除
type ModelNLD struct {
	CreateModelNLD

	UpdateTime *time.Time `json:"updateTime" gorm:"not null;"`
	ModifierId uint64     `json:"modifierId" gorm:"not null;"`
	Modifier   string     `json:"modifier" gorm:"size:32;not null;"`
}

func (m *ModelNLD) setUpdate(updatedAt *time.Time, account *LoginAccount) {
	m.UpdateTime = updatedAt
	if account == nil {
		// 无操作者上下文时保留实体自带的修改人，只刷新更新时间
		return
	}
	m.ModifierId = account.Id
	m.Modifier = account.Username
}

// 设置基础信息. 如创建时间，修改时间，创建者，修改者信息
func (m *ModelNLD) FillBaseInfo(account *LoginAccount) {
	if m.IsCreate() {
		applyCreate(m, account)
		// 新建时修改人即创建人，无操作者上下文时同为系统账号
		applyUpdate(m, orSysAccount(account))
		return
	}
	applyUpdate(m, account)
}

// 基础实体模型，数据表最基础字段，尽量每张表都包含这些字段
type Model struct {
	CreateModel

	UpdateTime *time.Time `json:"updateTime" gorm:"not null;"`
	ModifierId uint64     `json:"modifierId" gorm:"not null;"`
	Modifier   string     `json:"modifier" gorm:"size:32;not null;"`
}

func (m *Model) setUpdate(updatedAt *time.Time, account *LoginAccount) {
	m.UpdateTime = updatedAt
	if account == nil {
		// 无操作者上下文时保留实体自带的修改人，只刷新更新时间
		return
	}
	m.ModifierId = account.Id
	m.Modifier = account.Username
}

// 设置基础信息. 如创建时间，修改时间，创建者，修改者信息
func (m *Model) FillBaseInfo(account *LoginAccount) {
	if m.IsCreate() {
		m.IsDeleted = ModelUndeleted
		applyCreate(m, account)
		applyUpdate(m, orSysAccount(account))
		return
	}
	applyUpdate(m, account)
}

func orSysAccount(account *LoginAccount) *LoginAccount {
	if account == nil {
		return SysAccount
	}
	return account
}
