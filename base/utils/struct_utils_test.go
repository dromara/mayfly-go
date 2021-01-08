package utils

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

type Src struct {
	Id         *int64    `json:"id"`
	Username   string    `json:"username"`
	CreateTime time.Time `json:"time"`
	UpdateTime time.Time
	Inner      *SrcInner
}

type SrcInner struct {
	Name string
	Desc string
	Id   int64
	Dest *Dest
}

type Dest struct {
	Username   string
	Id         int64
	CreateTime time.Time
	Inner      *DestInner
}

type DestInner struct {
	Desc string
}

func TestDeepFields(t *testing.T) {
	////src := Src{Username: "test", Id: 1000, CreateTime: time.Now()}
	//si := SrcInner{Desc: "desc"}
	//src.Inner = &si
	////src.Id = 1222
	//dest := new(Dest)
	//err := structutils.Copy(dest, src)
	//if err != nil {
	//	fmt.Println(err.Error())
	//} else {
	//	fmt.Println(dest)
	//}

}

func TestGetFieldNames(t *testing.T) {
	//names := structutils.GetFieldNames(new(Src))
	//fmt.Println(names)
}

func TestMaps2Structs(t *testing.T) {
	mapInstance := make(map[string]interface{})
	mapInstance["Username"] = "liang637210"
	mapInstance["Id"] = 28
	mapInstance["CreateTime"] = time.Now()
	mapInstance["Creator"] = "createor"
	mapInstance["Inner.Id"] = 10
	mapInstance["Inner.Name"] = "hahah"
	mapInstance["Inner.Desc"] = "inner desc"
	mapInstance["Inner.Dest.Username"] = "inner dest uername"
	mapInstance["Inner.Dest.Inner.Desc"] = "inner dest inner desc"

	mapInstance2 := make(map[string]interface{})
	mapInstance2["Username"] = "liang6372102"
	mapInstance2["Id"] = 282
	mapInstance2["CreateTime"] = time.Now()
	mapInstance2["Creator"] = "createor2"
	mapInstance2["Inner.Id"] = 102
	mapInstance2["Inner.Name"] = "hahah2"
	mapInstance2["Inner.Desc"] = "inner desc2"
	mapInstance2["Inner.Dest.Username"] = "inner dest uername2"
	mapInstance2["Inner.Dest.Inner.Desc"] = "inner dest inner desc2"

	maps := make([]map[string]interface{}, 2)
	maps[0] = mapInstance
	maps[1] = mapInstance2
	res := new([]Src)
	err := Maps2Structs(maps, res)
	if err != nil {
		fmt.Println(err)
	}
}

func TestMap2Struct(t *testing.T) {
	mapInstance := make(map[string]interface{})
	mapInstance["Username"] = "liang637210"
	mapInstance["Id"] = 12
	mapInstance["CreateTime"] = time.Now()
	mapInstance["Creator"] = "createor"
	mapInstance["Inner.Id"] = nil
	mapInstance["Inner.Name"] = "hahah"
	mapInstance["Inner.Desc"] = "inner desc"
	mapInstance["Inner.Dest.Username"] = "inner dest uername"
	mapInstance["Inner.Dest.Inner.Desc"] = "inner dest inner desc"

	//innerMap := make(map[string]interface{})
	//innerMap["Name"] = "Innername"

	//a := new(Src)
	////a.Inner = new(SrcInner)
	//
	//stime := time.Now().UnixNano()
	//for i := 0; i < 1000000; i++ {
	//	err := structutils.Map2Struct(mapInstance, a)
	//	if err != nil {
	//		fmt.Println(err)
	//	}
	//}
	//etime := time.Now().UnixNano()
	//fmt.Println(etime - stime)
	//if err != nil {
	//	fmt.Println(err)
	//} else {
	//	fmt.Println(a)
	//}

	s := new(Src)
	//name, b := structutils.IndirectType(reflect.TypeOf(s)).FieldByName("Inner")
	//if structutils.IndirectType(name.Type).Kind() != reflect.Struct {
	//	fmt.Println(name.Name + "不是结构体")
	//} else {
	//	//innerType := name.Type
	//	innerValue := structutils.Indirect(reflect.ValueOf(s)).FieldByName("Inner")
	//	//if innerValue.IsValid() && innerValue.IsNil() {
	//	//	innerValue.Set(reflect.New(innerValue.Type().Elem()))
	//	//}
	//	if !innerValue.IsValid() {
	//		fmt.Println("is valid")
	//	} else {
	//		//innerValue.Set(reflect.New(innerValue.Type()))
	//		fmt.Println(innerValue.CanSet())
	//		fmt.Println(innerValue.CanAddr())
	//		//mapstructure.Decode(innerMap, innerValue.Addr().Interface())
	//	}
	//
	//}
	//
	//fmt.Println(name, b)
	//将 map 转换为指定的结构体
	// if err := decode(mapInstance, &s); err != nil {
	// 	fmt.Println(err)
	// }
	fmt.Printf("map2struct后得到的 struct 内容为:%v", s)
}

func getPrefixKeyMap(m map[string]interface{}) map[string]map[string]interface{} {
	key2map := make(map[string]map[string]interface{})
	for k, v := range m {
		if !strings.Contains(k, ".") {
			continue
		}
		lastIndex := strings.LastIndex(k, ".")
		prefix := k[0:lastIndex]
		m2 := key2map[prefix]
		if m2 == nil {
			key2map[prefix] = map[string]interface{}{k[lastIndex+1:]: v}
		} else {
			m2[k[lastIndex+1:]] = v
		}
		delete(m, k)
	}
	return key2map
}

func TestReflect(t *testing.T) {
	type dog struct {
		LegCount int
	}
	// 获取dog实例的反射值对象
	valueOfDog := reflect.ValueOf(&dog{}).Elem()

	// 获取legCount字段的值
	vLegCount := valueOfDog.FieldByName("LegCount")

	fmt.Println(vLegCount.CanSet())
	fmt.Println(vLegCount.CanAddr())
	// 尝试设置legCount的值(这里会发生崩溃)
	vLegCount.SetInt(4)
}

func TestTemplateResolve(t *testing.T) {
	d := make(map[string]string)
	d["Name"] = "黄先生"
	d["Age"] = "23jlfdsjf"
	resolve := TemplateResolve("{{.Name}} is name, and {{.Age}} is age", d)
	fmt.Println(resolve)

}
