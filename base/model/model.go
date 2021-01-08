package model

import (
	"errors"
	"mayfly-go/base/ctx"
	"mayfly-go/base/utils"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/siddontang/go/log"
)

type Model struct {
	Id         uint64     `orm:"column(id);auto" json:"id"`
	CreateTime *time.Time `orm:"column(create_time);type(datetime);null" json:"createTime"`
	CreatorId  uint64     `orm:"column(creator_id)" json:"creatorId"`
	Creator    string     `orm:"column(creator)" json:"creator"`
	UpdateTime *time.Time `orm:"column(update_time);type(datetime);null" json:"updateTime"`
	ModifierId uint64     `orm:"column(modifier_id)" json:"modifierId"`
	Modifier   string     `orm:"column(modifier)" json:"modifier"`
}

// 设置基础信息. 如创建时间，修改时间，创建者，修改者信息
func (m *Model) SetBaseInfo(account *ctx.LoginAccount) {
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

// 获取orm querySeter
func QuerySetter(table interface{}) orm.QuerySeter {
	return getOrm().QueryTable(table)
}

// 根据id获取实体对象。model需为指针类型（需要将查询出来的值赋值给model）
//
// 若error不为nil则为不存在该记录
func GetById(model interface{}, id uint64, cols ...string) error {
	return QuerySetter(model).Filter("Id", id).One(model, cols...)
}

// 根据id更新model，更新字段为model中不为空的值，即int类型不为0，ptr类型不为nil这类字段值
func UpdateById(model interface{}) (int64, error) {
	var id uint64
	params := orm.Params{}
	err := utils.DoWithFields(model, func(ft reflect.StructField, fv reflect.Value) error {
		if utils.IsBlank(fv) {
			return nil
		}
		if ft.Name == "Id" {
			if id = fv.Uint(); id == 0 {
				return errors.New("根据id更新model时Id不能为0")
			}
			return nil
		}
		params[ft.Name] = fv.Interface()
		return nil
	})
	if err != nil {
		return 0, err
	}
	return QuerySetter(model).Filter("Id", id).Update(params)
}

// 根据id删除model
func DeleteById(model interface{}, id uint64) (int64, error) {
	return QuerySetter(model).Filter("Id", id).Delete()
}

// 插入model
func Insert(model interface{}) (int64, error) {
	return getOrm().Insert(model)
}

// 获取满足model中不为空的字段值条件的所有数据.
//
// @param list为数组类型 如 var users []*User
func ListByCondition(model interface{}, list interface{}) {
	qs := QuerySetter(model)
	utils.DoWithFields(model, func(ft reflect.StructField, fv reflect.Value) error {
		if !utils.IsBlank(fv) {
			qs = qs.Filter(ft.Name, fv.Interface())
		}
		return nil
	})
	qs.All(list)
}

// 获取满足model中不为空的字段值条件的单个对象。model需为指针类型（需要将查询出来的值赋值给model）
//
// 若 error不为nil，则为不存在该记录
func GetByCondition(model interface{}, cols ...string) error {
	qs := QuerySetter(model)
	utils.DoWithFields(model, func(ft reflect.StructField, fv reflect.Value) error {
		if !utils.IsBlank(fv) {
			qs = qs.Filter(ft.Name, fv.Interface())
		}
		return nil
	})
	return qs.One(model, cols...)
}

// 获取分页结果
func GetPage(seter orm.QuerySeter, pageParam *PageParam, models interface{}, toModels interface{}) PageResult {
	count, _ := seter.Count()
	if count == 0 {
		return PageResult{Total: 0, List: nil}
	}
	_, qerr := seter.Limit(pageParam.PageSize, pageParam.PageNum-1).All(models, getFieldNames(toModels)...)
	BizErrIsNil(qerr, "查询错误")
	err := utils.Copy(toModels, models)
	BizErrIsNil(err, "实体转换错误")
	return PageResult{Total: count, List: toModels}
}

// 根据sql获取分页对象
func GetPageBySql(sql string, toModel interface{}, param *PageParam, args ...interface{}) PageResult {
	selectIndex := strings.Index(sql, "SELECT ") + 7
	fromIndex := strings.Index(sql, " FROM")
	selectCol := sql[selectIndex:fromIndex]
	countSql := strings.Replace(sql, selectCol, "COUNT(*) AS total ", 1)
	// 查询count
	o := getOrm()
	type TotalRes struct {
		Total int64
	}
	var totalRes TotalRes
	_ = o.Raw(countSql, args).QueryRow(&totalRes)
	total := totalRes.Total
	if total == 0 {
		return PageResult{Total: 0, List: nil}
	}
	// 分页查询
	limitSql := sql + " LIMIT " + strconv.Itoa(param.PageNum-1) + ", " + strconv.Itoa(param.PageSize)
	var maps []orm.Params
	_, err := o.Raw(limitSql, args).Values(&maps)
	if err != nil {
		panic(errors.New("查询错误 : " + err.Error()))
	}
	e := ormParams2Struct(maps, toModel)
	if e != nil {
		panic(e)
	}
	return PageResult{Total: total, List: toModel}
}

func GetListBySql(sql string, params ...interface{}) *[]orm.Params {
	var maps []orm.Params
	_, err := getOrm().Raw(sql, params).Values(&maps)
	if err != nil {
		log.Error("根据sql查询数据列表失败：%s", err.Error())
	}
	return &maps
}

// 获取所有列表数据
// model为数组类型 如 var users []*User
func GetList(seter orm.QuerySeter, model interface{}, toModel interface{}) {
	_, _ = seter.All(model, getFieldNames(toModel)...)
	err := utils.Copy(toModel, model)
	BizErrIsNil(err, "实体转换错误")
}

// 根据toModel结构体字段查询单条记录，并将值赋值给toModel
func GetOne(seter orm.QuerySeter, model interface{}, toModel interface{}) error {
	err := seter.One(model, getFieldNames(toModel)...)
	if err != nil {
		return err
	}
	cerr := utils.Copy(toModel, model)
	BizErrIsNil(cerr, "实体转换错误")
	return nil
}

// 根据实体以及指定字段值查询实体，若字段数组为空，则默认用id查
func GetBy(model interface{}, fs ...string) error {
	err := getOrm().Read(model, fs...)
	if err != nil {
		if err == orm.ErrNoRows {
			return errors.New("该数据不存在")
		} else {
			return errors.New("查询失败")
		}
	}
	return nil
}

func Update(model interface{}, fs ...string) error {
	_, err := getOrm().Update(model, fs...)
	if err != nil {
		return errors.New("数据更新失败")
	}
	return nil
}

func Delete(model interface{}, fs ...string) error {
	_, err := getOrm().Delete(model, fs...)
	if err != nil {
		return errors.New("数据删除失败")
	}
	return nil
}

func getOrm() orm.Ormer {
	return orm.NewOrm()
}

// 结果模型缓存
var resultModelCache = make(map[string][]string)

// 获取实体对象的字段名
func getFieldNames(obj interface{}) []string {
	objType := indirectType(reflect.TypeOf(obj))
	cacheKey := objType.PkgPath() + "." + objType.Name()
	cache := resultModelCache[cacheKey]
	if cache != nil {
		return cache
	}
	cache = getFieldNamesByType("", reflect.TypeOf(obj))
	resultModelCache[cacheKey] = cache
	return cache
}

func indirectType(reflectType reflect.Type) reflect.Type {
	for reflectType.Kind() == reflect.Ptr || reflectType.Kind() == reflect.Slice {
		reflectType = reflectType.Elem()
	}
	return reflectType
}

func getFieldNamesByType(namePrefix string, reflectType reflect.Type) []string {
	var fieldNames []string

	if reflectType = indirectType(reflectType); reflectType.Kind() == reflect.Struct {
		for i := 0; i < reflectType.NumField(); i++ {
			t := reflectType.Field(i)
			tName := t.Name
			// 判断结构体字段是否为结构体，是的话则跳过
			it := indirectType(t.Type)
			if it.Kind() == reflect.Struct {
				itName := it.Name()
				// 如果包含Time或time则表示为time类型，无需递归该结构体字段
				if !strings.Contains(itName, "BaseModel") && !strings.Contains(itName, "Time") &&
					!strings.Contains(itName, "time") {
					fieldNames = append(fieldNames, getFieldNamesByType(tName+"__", it)...)
					continue
				}
			}

			if t.Anonymous {
				fieldNames = append(fieldNames, getFieldNamesByType("", t.Type)...)
			} else {
				fieldNames = append(fieldNames, namePrefix+tName)
			}
		}
	}

	return fieldNames
}

func ormParams2Struct(maps []orm.Params, structs interface{}) error {
	structsV := reflect.Indirect(reflect.ValueOf(structs))
	valType := structsV.Type()
	valElemType := valType.Elem()
	sliceType := reflect.SliceOf(valElemType)

	length := len(maps)

	valSlice := structsV
	if valSlice.IsNil() {
		// Make a new slice to hold our result, same size as the original data.
		valSlice = reflect.MakeSlice(sliceType, length, length)
	}

	for i := 0; i < length; i++ {
		err := utils.Map2Struct(maps[i], valSlice.Index(i).Addr().Interface())
		if err != nil {
			return err
		}
	}
	structsV.Set(valSlice)
	return nil
}
