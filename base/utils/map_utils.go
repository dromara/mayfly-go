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
