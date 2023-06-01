package utils

import (
	"reflect"
	"strconv"
)

func GetString4Map(m map[string]any, key string) string {
	return m[key].(string)
}

func GetInt4Map(m map[string]any, key string) int {
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
	m map[string]any
}

func MapBuilder(key string, value any) *mapBuilder {
	mb := new(mapBuilder)
	mb.m = make(map[string]any, 4)
	mb.m[key] = value
	return mb
}

func (mb *mapBuilder) Put(key string, value any) *mapBuilder {
	mb.m[key] = value
	return mb
}

func (mb *mapBuilder) ToMap() map[string]any {
	return mb.m
}
