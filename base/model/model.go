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
	return global.Db.Debug().Select(cols).Where("id = ?", id).First(model).Error
}

// 根据id更新model，更新字段为model中不为空的值，即int类型不为0，ptr类型不为nil这类字段值
func UpdateById(model interface{}) error {
	return global.Db.Model(model).Updates(model).Error
}

// 根据id删除model
func DeleteById(model interface{}, id uint64) error {
	// return QuerySetter(model).Filter("Id", id).Delete()
	return global.Db.Delete(model).Error
}

// 插入model
func Insert(model interface{}) error {
	return global.Db.Create(model).Error
}

// 获取满足model中不为空的字段值条件的所有数据.
//
// @param list为数组类型 如 var users []*User，可指定为非model结构体，即只包含需要返回的字段结构体
func ListBy(model interface{}, list interface{}, cols ...string) {
	global.Db.Debug().Model(model).Select(cols).Where(model).Find(list)
}

// 获取满足model中不为空的字段值条件的单个对象。model需为指针类型（需要将查询出来的值赋值给model）
//
// 若 error不为nil，则为不存在该记录
func GetBy(model interface{}, cols ...string) error {
	return global.Db.Debug().Select(cols).Where(model).First(model).Error
}

// 获取满足conditionModel中不为空的字段值条件的单个对象。model需为指针类型（需要将查询出来的值赋值给model）
//	@param toModel  需要查询的字段
// 若 error不为nil，则为不存在该记录
func GetByConditionTo(conditionModel interface{}, toModel interface{}) error {
	return global.Db.Debug().Model(conditionModel).Where(conditionModel).First(toModel).Error
}

// 获取分页结果
func GetPage(pageParam *PageParam, conditionModel interface{}, toModels interface{}, orderBy ...string) PageResult {
	var count int64
	global.Db.Debug().Model(conditionModel).Where(conditionModel).Count(&count)
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
	global.Db.Debug().Model(conditionModel).Where(conditionModel).Order(orderByStr).Limit(pageSize).Offset((page - 1) * pageSize).Find(toModels)
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

// 获取所有列表数据
// model为数组类型 如 var users []*User
// func GetList(seter orm.QuerySeter, model interface{}, toModel interface{}) {
// 	_, _ = seter.All(model, getFieldNames(toModel)...)
// 	err := utils.Copy(toModel, model)
// 	biz.BizErrIsNil(err, "实体转换错误")
// }

// func getOrm() orm.Ormer {
// 	return orm.NewOrm()
// }

// // 结果模型缓存
// var resultModelCache = make(map[string][]string)

// // 获取实体对象的字段名
// func getFieldNames(obj interface{}) []string {
// 	objType := indirectType(reflect.TypeOf(obj))
// 	cacheKey := objType.PkgPath() + "." + objType.Name()
// 	cache := resultModelCache[cacheKey]
// 	if cache != nil {
// 		return cache
// 	}
// 	cache = getFieldNamesByType("", reflect.TypeOf(obj))
// 	resultModelCache[cacheKey] = cache
// 	return cache
// }

// func indirectType(reflectType reflect.Type) reflect.Type {
// 	for reflectType.Kind() == reflect.Ptr || reflectType.Kind() == reflect.Slice {
// 		reflectType = reflectType.Elem()
// 	}
// 	return reflectType
// }

// func getFieldNamesByType(namePrefix string, reflectType reflect.Type) []string {
// 	var fieldNames []string

// 	if reflectType = indirectType(reflectType); reflectType.Kind() == reflect.Struct {
// 		for i := 0; i < reflectType.NumField(); i++ {
// 			t := reflectType.Field(i)
// 			tName := t.Name
// 			// 判断结构体字段是否为结构体，是的话则跳过
// 			it := indirectType(t.Type)
// 			if it.Kind() == reflect.Struct {
// 				itName := it.Name()
// 				// 如果包含Time或time则表示为time类型，无需递归该结构体字段
// 				if !strings.Contains(itName, "BaseModel") && !strings.Contains(itName, "Time") &&
// 					!strings.Contains(itName, "time") {
// 					fieldNames = append(fieldNames, getFieldNamesByType(tName+"__", it)...)
// 					continue
// 				}
// 			}

// 			if t.Anonymous {
// 				fieldNames = append(fieldNames, getFieldNamesByType("", t.Type)...)
// 			} else {
// 				fieldNames = append(fieldNames, namePrefix+tName)
// 			}
// 		}
// 	}

// 	return fieldNames
// }

// func ormParams2Struct(maps []orm.Params, structs interface{}) error {
// 	structsV := reflect.Indirect(reflect.ValueOf(structs))
// 	valType := structsV.Type()
// 	valElemType := valType.Elem()
// 	sliceType := reflect.SliceOf(valElemType)

// 	length := len(maps)

// 	valSlice := structsV
// 	if valSlice.IsNil() {
// 		// Make a new slice to hold our result, same size as the original data.
// 		valSlice = reflect.MakeSlice(sliceType, length, length)
// 	}

// 	for i := 0; i < length; i++ {
// 		err := utils.Map2Struct(maps[i], valSlice.Index(i).Addr().Interface())
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	structsV.Set(valSlice)
// 	return nil
// }
