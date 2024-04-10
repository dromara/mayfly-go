package structx

import "reflect"

// DiffFields 比较两个结构体中指定字段的属性值是否发生改变，并返回改变的字段名
func DiffFields[T any](t1 T, t2 T, fieldsToCompare ...string) []string {
	var changedFields []string

	oldValue := reflect.ValueOf(t1)
	newValue := reflect.ValueOf(t2)

	for _, fieldName := range fieldsToCompare {
		oldFieldValue := oldValue.FieldByName(fieldName)
		newFieldValue := newValue.FieldByName(fieldName)

		if !reflect.DeepEqual(oldFieldValue.Interface(), newFieldValue.Interface()) {
			changedFields = append(changedFields, fieldName)
		}
	}

	return changedFields
}
