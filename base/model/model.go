package model

import (
	"mayfly-go/base/global"
	"strconv"

	"strings"
	"time"
)

type Model struct {
	Id         uint64     `json:"id"`
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

// 根据id获取实体对象。model需为指针类型（需要将查询出来的值赋值给model）
//
// 若error不为nil则为不存在该记录
func GetById(model interface{}, id uint64, cols ...string) error {
	return global.Db.Select(cols).Where("id = ?", id).First(model).Error
}

// 根据id更新model，更新字段为model中不为空的值，即int类型不为0，ptr类型不为nil这类字段值
func UpdateById(model interface{}) error {
	return global.Db.Model(model).Updates(model).Error
}

// 根据id删除model
func DeleteById(model interface{}, id uint64) error {
	return global.Db.Delete(model, "id = ?", id).Error
}

// 插入model
func Insert(model interface{}) error {
	return global.Db.Create(model).Error
}

// 获取满足model中不为空的字段值条件的所有数据.
//
// @param list为数组类型 如 var users []*User，可指定为非model结构体，即只包含需要返回的字段结构体
func ListBy(model interface{}, list interface{}, cols ...string) {
	global.Db.Model(model).Select(cols).Where(model).Find(list)
}

// 获取满足model中不为空的字段值条件的单个对象。model需为指针类型（需要将查询出来的值赋值给model）
//
// 若 error不为nil，则为不存在该记录
func GetBy(model interface{}, cols ...string) error {
	return global.Db.Select(cols).Where(model).First(model).Error
}

// 获取满足conditionModel中不为空的字段值条件的单个对象。model需为指针类型（需要将查询出来的值赋值给model）
//	@param toModel  需要查询的字段
// 若 error不为nil，则为不存在该记录
func GetByConditionTo(conditionModel interface{}, toModel interface{}) error {
	return global.Db.Model(conditionModel).Where(conditionModel).First(toModel).Error
}

// 获取分页结果
func GetPage(pageParam *PageParam, conditionModel interface{}, toModels interface{}, orderBy ...string) PageResult {
	var count int64
	global.Db.Model(conditionModel).Where(conditionModel).Count(&count)
	if count == 0 {
		return PageResult{Total: 0, List: []string{}}
	}
	page := pageParam.PageNum
	pageSize := pageParam.PageSize
	var orderByStr string
	if orderBy == nil {
		orderByStr = "id desc"
	} else {
		orderByStr = strings.Join(orderBy, ",")
	}
	global.Db.Model(conditionModel).Where(conditionModel).Order(orderByStr).Limit(pageSize).Offset((page - 1) * pageSize).Find(toModels)
	return PageResult{Total: count, List: toModels}
}

// 根据sql获取分页对象
func GetPageBySql(sql string, param *PageParam, toModel interface{}, args ...interface{}) PageResult {
	db := global.Db
	selectIndex := strings.Index(sql, "SELECT ") + 7
	fromIndex := strings.Index(sql, " FROM")
	selectCol := sql[selectIndex:fromIndex]
	countSql := strings.Replace(sql, selectCol, "COUNT(*) AS total ", 1)
	// 查询count
	var count int
	db.Raw(countSql, args...).Scan(&count)
	if count == 0 {
		return PageResult{Total: 0, List: []string{}}
	}
	// 分页查询
	limitSql := sql + " LIMIT " + strconv.Itoa(param.PageNum-1) + ", " + strconv.Itoa(param.PageSize)
	db.Raw(limitSql).Scan(toModel)
	return PageResult{Total: int64(count), List: toModel}
}

func GetListBySql(sql string, params ...interface{}) []map[string]interface{} {
	var maps []map[string]interface{}
	global.Db.Raw(sql, params).Scan(&maps)
	return maps
}
