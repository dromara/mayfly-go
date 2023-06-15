package model

import (
	"fmt"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/global"
	"strconv"

	"strings"
	"time"

	"gorm.io/gorm"
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

func Tx(funcs ...func(db *gorm.DB) error) (err error) {
	tx := global.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			err = fmt.Errorf("%v", err)
		}
	}()
	for _, f := range funcs {
		err = f(tx)
		if err != nil {
			tx.Rollback()
			return
		}
	}
	err = tx.Commit().Error
	return
}

// 根据id获取实体对象。model需为指针类型（需要将查询出来的值赋值给model）
//
// 若error不为nil则为不存在该记录
// @param model  数据库映射实体模型
func GetById(model any, id uint64, cols ...string) error {
	return global.Db.Select(cols).Where("id = ?", id).First(model).Error
}

// 根据map条件查询列表，map中的值如果为数组，则使用in查询
// @param model  数据库映射实体模型
func GetByIdIn(model any, list any, ids []uint64, orderBy ...string) {
	var orderByStr string
	if orderBy == nil {
		orderByStr = "id desc"
	} else {
		orderByStr = strings.Join(orderBy, ",")
	}
	global.Db.Model(model).Where("id in (?)", ids).Order(orderByStr).Find(list)
}

// 根据map指定条件查询列表
func SelectByMap(model any, list any, where map[string]any, orderBy ...string) {
	var orderByStr string
	if orderBy == nil {
		orderByStr = "id desc"
	} else {
		orderByStr = strings.Join(orderBy, ",")
	}
	global.Db.Model(model).Where(where).Order(orderByStr).Find(list)
}

// 根据model指定条件统计数量
func CountBy(model any) int64 {
	var count int64
	global.Db.Model(model).Where(model).Count(&count)
	return count
}

// 根据map为条件统计数量，map中的值如果为数组，则使用in查询
// @param model  数据库映射实体模型
// @param where  条件map
func CountByMap(model any, where map[string]any) int64 {
	var count int64
	global.Db.Model(model).Where(where).Count(&count)
	return count
}

// 根据统计sql返回统计数量
func CountBySql(sql string) int64 {
	var count int64
	global.Db.Raw(sql).Scan(&count)
	return count
}

// 根据id更新model，更新字段为model中不为空的值，即int类型不为0，ptr类型不为nil这类字段值
// @param model  数据库映射实体模型
func UpdateById(model any) error {
	return global.Db.Model(model).Updates(model).Error
}

// 根据id删除model
// @param model  数据库映射实体模型
func DeleteById(model any, id uint64) error {
	return global.Db.Delete(model, "id = ?", id).Error
}

// 根据条件删除
// @param model  数据库映射实体模型
func DeleteByCondition(model any) error {
	return global.Db.Where(model).Delete(model).Error
}

// 插入model
// @param model  数据库映射实体模型
func Insert(model any) error {
	return global.Db.Create(model).Error
}

// 获取满足model中不为空的字段值条件的所有数据.
//
// @param list为数组类型 如 var users *[]User，可指定为非model结构体，即只包含需要返回的字段结构体
func ListBy(model any, list any, cols ...string) {
	global.Db.Model(model).Select(cols).Where(model).Order("id desc").Find(list)
}

// 获取满足model中不为空的字段值条件的所有数据.
//
// @param list为数组类型 如 var users *[]User，可指定为非model结构体
// @param model  数据库映射实体模型
func ListByOrder(model any, list any, order ...string) {
	var orderByStr string
	if order == nil {
		orderByStr = "id desc"
	} else {
		orderByStr = strings.Join(order, ",")
	}
	global.Db.Model(model).Where(model).Order(orderByStr).Find(list)
}

// 获取满足model中不为空的字段值条件的单个对象。model需为指针类型（需要将查询出来的值赋值给model）
//
// 若 error不为nil，则为不存在该记录
// @param model  数据库映射实体模型
func GetBy(model any, cols ...string) error {
	return global.Db.Select(cols).Where(model).First(model).Error
}

// 获取满足conditionModel中不为空的字段值条件的单个对象。model需为指针类型（需要将查询出来的值赋值给model）
//
//	@param toModel  需要查询的字段
//
// 若 error不为nil，则为不存在该记录
func GetByConditionTo(conditionModel any, toModel any) error {
	return global.Db.Model(conditionModel).Where(conditionModel).First(toModel).Error
}

// 获取分页结果
func GetPage(pageParam *PageParam, model any, conditionModel any, toModels any, orderBy ...string) *PageResult {
	var count int64
	err := global.Db.Model(model).Where(conditionModel).Count(&count).Error
	biz.ErrIsNilAppendErr(err, " 查询错误：%s")
	if count == 0 {
		return &PageResult{Total: 0, List: []string{}}
	}

	page := pageParam.PageNum
	pageSize := pageParam.PageSize
	var orderByStr string
	if orderBy == nil {
		orderByStr = "id desc"
	} else {
		orderByStr = strings.Join(orderBy, ",")
	}
	err = global.Db.Model(model).Where(conditionModel).Order(orderByStr).Limit(pageSize).Offset((page - 1) * pageSize).Find(toModels).Error
	biz.ErrIsNil(err, "查询失败")
	return &PageResult{Total: count, List: toModels}
}

// 根据sql获取分页对象
func GetPageBySql(sql string, param *PageParam, toModel any, args ...any) *PageResult {
	db := global.Db
	selectIndex := strings.Index(sql, "SELECT ") + 7
	fromIndex := strings.Index(sql, " FROM")
	selectCol := sql[selectIndex:fromIndex]
	countSql := strings.Replace(sql, selectCol, "COUNT(*) AS total ", 1)
	// 查询count
	var count int
	err := db.Raw(countSql, args...).Scan(&count).Error
	biz.ErrIsNilAppendErr(err, "查询失败: %s")
	if count == 0 {
		return &PageResult{Total: 0, List: []string{}}
	}
	// 分页查询
	limitSql := sql + " LIMIT " + strconv.Itoa((param.PageNum-1)*param.PageSize) + ", " + strconv.Itoa(param.PageSize)
	err = db.Raw(limitSql, args...).Scan(toModel).Error
	biz.ErrIsNil(err, "查询失败: %s")
	return &PageResult{Total: int64(count), List: toModel}
}

func GetListBySql(sql string, params ...any) []map[string]any {
	var maps []map[string]any
	global.Db.Raw(sql, params...).Scan(&maps)
	return maps
}

func GetListBySql2Model(sql string, toEntity any, params ...any) error {
	return global.Db.Raw(sql, params...).Find(toEntity).Error
}

func ExecSql(sql string, params ...any) {
	global.Db.Exec(sql, params...)
}
