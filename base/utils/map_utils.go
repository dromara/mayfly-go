package utils

import (
	"reflect"
	"strconv"
)

func GetString4Map(m map[string]interface{}, key string) string {
	return m[key].(string)
}

func GetInt4Map(m map[string]interface{}, key string) int {
	i := m[key]
	iKind := reflect.TypeOf(i).Kind()
	if iKind == reflect.Int {
		return i.(int)
	}
	if iKind == reflect.String {
		i, _ := strconv.Atoi(i.(string))
		return i
	}
	return 0
}

// map构造器
type mapBuilder struct {
	m map[string]interface{}
}

func MapBuilder(key string, value interface{}) *mapBuilder {
	mb := new(mapBuilder)
	mb.m = make(map[string]interface{}, 4)
	mb.m[key] = value
	return mb
}

func (mb *mapBuilder) Put(key string, value interface{}) *mapBuilder {
	mb.m[key] = value
	return mb
}

func (mb *mapBuilder) ToMap() map[string]interface{} {
	return mb.m
}
