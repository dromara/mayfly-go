package structx

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"mayfly-go/pkg/utils/collx"
	"reflect"
	"strconv"
	"strings"
)

// Copy copy things，引用至copier
func Copy(toValue any, fromValue any) (err error) {
	var (
		isSlice bool
		amount  = 1
		from    = Indirect(reflect.ValueOf(fromValue))
		to      = Indirect(reflect.ValueOf(toValue))
	)

	if !to.CanAddr() {
		return errors.New("copy to value is unaddressable")
	}

	// Return is from value is invalid
	if !from.IsValid() {
		return
	}

	fromType := IndirectType(from.Type())
	toType := IndirectType(to.Type())

	// Just set it if possible to assign
	// And need to do copy anyway if the type is struct
	if fromType.Kind() != reflect.Struct && from.Type().AssignableTo(to.Type()) {
		to.Set(from)
		return
	}

	if fromType.Kind() != reflect.Struct || toType.Kind() != reflect.Struct {
		return
	}

	if to.Kind() == reflect.Slice {
		isSlice = true
		if from.Kind() == reflect.Slice {
			amount = from.Len()
		}
	}

	for i := 0; i < amount; i++ {
		var dest, source reflect.Value

		if isSlice {
			// source
			if from.Kind() == reflect.Slice {
				source = Indirect(from.Index(i))
			} else {
				source = Indirect(from)
			}
			// dest
			dest = Indirect(reflect.New(toType).Elem())
		} else {
			source = Indirect(from)
			dest = Indirect(to)
		}

		// check source
		if source.IsValid() {
			fromTypeFields := deepFields(fromType)
			//fmt.Printf("%#v", fromTypeFields)
			// Copy from field to field or method
			for _, field := range fromTypeFields {
				name := field.Name

				if fromField := source.FieldByName(name); fromField.IsValid() {
					// has field
					if toField := dest.FieldByName(name); toField.IsValid() {
						if toField.CanSet() {
							if !set(toField, fromField) {
								if err := Copy(toField.Addr().Interface(), fromField.Interface()); err != nil {
									return err
								}
							}
						}
					} else {
						// try to set to method
						var toMethod reflect.Value
						if dest.CanAddr() {
							toMethod = dest.Addr().MethodByName(name)
						} else {
							toMethod = dest.MethodByName(name)
						}

						if toMethod.IsValid() && toMethod.Type().NumIn() == 1 && fromField.Type().AssignableTo(toMethod.Type().In(0)) {
							toMethod.Call([]reflect.Value{fromField})
						}
					}
				}
			}

			// Copy from method to field
			for _, field := range deepFields(toType) {
				name := field.Name

				var fromMethod reflect.Value
				if source.CanAddr() {
					fromMethod = source.Addr().MethodByName(name)
				} else {
					fromMethod = source.MethodByName(name)
				}

				if fromMethod.IsValid() && fromMethod.Type().NumIn() == 0 && fromMethod.Type().NumOut() == 1 {
					if toField := dest.FieldByName(name); toField.IsValid() && toField.CanSet() {
						values := fromMethod.Call([]reflect.Value{})
						if len(values) >= 1 {
							set(toField, values[0])
						}
					}
				}
			}
		}
		if isSlice {
			if dest.Addr().Type().AssignableTo(to.Type().Elem()) {
				to.Set(reflect.Append(to, dest.Addr()))
			} else if dest.Type().AssignableTo(to.Type().Elem()) {
				to.Set(reflect.Append(to, dest))
			}
		}
	}
	return
}

// 对结构体的每个字段以及字段值执行doWith回调函数, 包括匿名属性的字段
func DoWithFields(str any, doWith func(fType reflect.StructField, fValue reflect.Value) error) error {
	t := IndirectType(reflect.TypeOf(str))
	if t.Kind() != reflect.Struct {
		return errors.New("非结构体")
	}

	fieldNum := t.NumField()
	v := Indirect(reflect.ValueOf(str))
	for i := 0; i < fieldNum; i++ {
		ft := t.Field(i)
		fv := v.Field(i)
		// 如果是匿名属性，则递归调用该方法
		if ft.Anonymous {
			DoWithFields(fv.Interface(), doWith)
			continue
		}
		err := doWith(ft, fv)
		if err != nil {
			return err
		}
	}
	return nil
}

func deepFields(reflectType reflect.Type) []reflect.StructField {
	var fields []reflect.StructField

	if reflectType = IndirectType(reflectType); reflectType.Kind() == reflect.Struct {
		for i := 0; i < reflectType.NumField(); i++ {
			v := reflectType.Field(i)
			if v.Anonymous {
				fields = append(fields, deepFields(v.Type)...)
			} else {
				fields = append(fields, v)
			}
		}
	}

	return fields
}

func Indirect(reflectValue reflect.Value) reflect.Value {
	for reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}
	return reflectValue
}

func IndirectType(reflectType reflect.Type) reflect.Type {
	for reflectType.Kind() == reflect.Ptr || reflectType.Kind() == reflect.Slice {
		reflectType = reflectType.Elem()
	}
	return reflectType
}

func set(to, from reflect.Value) bool {
	if from.IsValid() {
		if to.Kind() == reflect.Ptr {
			//set `to` to nil if from is nil
			if from.Kind() == reflect.Ptr && from.IsNil() {
				to.Set(reflect.Zero(to.Type()))
				return true
			} else if to.IsNil() {
				to.Set(reflect.New(to.Type().Elem()))
			}
			to = to.Elem()
		}

		if from.Type().ConvertibleTo(to.Type()) {
			to.Set(from.Convert(to.Type()))
		} else if scanner, ok := to.Addr().Interface().(sql.Scanner); ok {
			err := scanner.Scan(from.Interface())
			if err != nil {
				return false
			}
		} else if from.Kind() == reflect.Ptr {
			return set(to, from.Elem())
		} else {
			return false
		}
	}
	return true
}

func Map2Struct(m map[string]any, s any) error {
	toValue := Indirect(reflect.ValueOf(s))
	if !toValue.CanAddr() {
		return errors.New("to value is unaddressable")
	}

	innerStructMaps := getInnerStructMaps(m)
	if len(innerStructMaps) != 0 {
		for k, v := range innerStructMaps {
			var fieldV reflect.Value
			if strings.Contains(k, ".") {
				fieldV = getFiledValueByPath(k, toValue)
			} else {
				fieldV = toValue.FieldByName(k)
			}

			if !fieldV.CanSet() || !fieldV.CanAddr() {
				continue
			}
			fieldT := fieldV.Type().Elem()
			if fieldT.Kind() != reflect.Struct {
				return errors.New(k + "不是结构体")
			}
			// 如果值为nil，则默认创建一个并赋值
			if fieldV.IsNil() {
				fieldV.Set(reflect.New(fieldT))
			}
			err := Map2Struct(v, fieldV.Addr().Interface())
			if err != nil {
				return err
			}
		}
	}
	var err error
	for k, v := range m {
		if v == nil {
			continue
		}
		k = strings.Title(k)
		// 如果key含有下划线，则将其转为驼峰
		if strings.Contains(k, "_") {
			k = Case2Camel(k)
		}

		fieldV := toValue.FieldByName(k)
		if !fieldV.CanSet() {
			continue
		}

		err = decode(k, v, fieldV)
		if err != nil {
			return err
		}
	}

	return nil
}

func Maps2Structs(maps []map[string]any, structs any) error {
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
		err := Map2Struct(maps[i], valSlice.Index(i).Addr().Interface())
		if err != nil {
			return err
		}
	}
	structsV.Set(valSlice)
	return nil
}

func getFiledValueByPath(path string, value reflect.Value) reflect.Value {
	split := strings.Split(path, ".")
	for _, v := range split {
		if value.Type().Kind() == reflect.Ptr {
			// 如果值为nil，则创建并赋值
			if value.IsNil() {
				value.Set(reflect.New(IndirectType(value.Type())))
			}
			value = value.Elem()
		}
		value = value.FieldByName(v)
	}
	return value
}

func getInnerStructMaps(m map[string]any) map[string]map[string]any {
	key2map := make(map[string]map[string]any)
	for k, v := range m {
		if !strings.Contains(k, ".") {
			continue
		}
		lastIndex := strings.LastIndex(k, ".")
		prefix := k[0:lastIndex]
		m2 := key2map[prefix]
		if m2 == nil {
			key2map[prefix] = collx.M{k[lastIndex+1:]: v}
		} else {
			m2[k[lastIndex+1:]] = v
		}
		delete(m, k)
	}
	return key2map
}

//  decode等方法摘抄自mapstructure库

func decode(name string, input any, outVal reflect.Value) error {
	var inputVal reflect.Value
	if input != nil {
		inputVal = reflect.ValueOf(input)

		// We need to check here if input is a typed nil. Typed nils won't
		// match the "input == nil" below so we check that here.
		if inputVal.Kind() == reflect.Ptr && inputVal.IsNil() {
			input = nil
		}
	}

	if !inputVal.IsValid() {
		// If the input value is invalid, then we just set the value
		// to be the zero value.
		outVal.Set(reflect.Zero(outVal.Type()))
		return nil
	}

	var err error
	outputKind := getKind(outVal)
	switch outputKind {
	case reflect.Int:
		err = decodeInt(name, input, outVal)
	case reflect.Uint:
		err = decodeUint(name, input, outVal)
	case reflect.Float32:
		err = decodeFloat(name, input, outVal)
	case reflect.String:
		err = decodeString(name, input, outVal)
	case reflect.Ptr:
		_, err = decodePtr(name, input, outVal)
	default:
		// If we reached this point then we weren't able to decode it
		return fmt.Errorf("%s: unsupported type: %s", name, outputKind)
	}
	return err
}

func decodeInt(name string, data any, val reflect.Value) error {
	dataVal := reflect.Indirect(reflect.ValueOf(data))
	dataKind := getKind(dataVal)
	dataType := dataVal.Type()

	switch {
	case dataKind == reflect.Int:
		val.SetInt(dataVal.Int())
	case dataKind == reflect.Uint:
		val.SetInt(int64(dataVal.Uint()))
	case dataKind == reflect.Float32:
		val.SetInt(int64(dataVal.Float()))
	case dataKind == reflect.Bool:
		if dataVal.Bool() {
			val.SetInt(1)
		} else {
			val.SetInt(0)
		}
	case dataKind == reflect.String:
		i, err := strconv.ParseInt(dataVal.String(), 0, val.Type().Bits())
		if err == nil {
			val.SetInt(i)
		} else {
			return fmt.Errorf("cannot parse '%s' as int: %s", name, err)
		}
	case dataType.PkgPath() == "encoding/json" && dataType.Name() == "Number":
		jn := data.(json.Number)
		i, err := jn.Int64()
		if err != nil {
			return fmt.Errorf(
				"error decoding json.Number into %s: %s", name, err)
		}
		val.SetInt(i)
	default:
		return fmt.Errorf(
			"'%s' expected type '%s', got unconvertible type '%s'",
			name, val.Type(), dataVal.Type())
	}

	return nil
}

func decodeUint(name string, data any, val reflect.Value) error {
	dataVal := reflect.Indirect(reflect.ValueOf(data))
	dataKind := getKind(dataVal)
	dataType := dataVal.Type()

	switch {
	case dataKind == reflect.Int:
		i := dataVal.Int()
		if i < 0 {
			return fmt.Errorf("cannot parse '%s', %d overflows uint",
				name, i)
		}
		val.SetUint(uint64(i))
	case dataKind == reflect.Uint:
		val.SetUint(dataVal.Uint())
	case dataKind == reflect.Float32:
		f := dataVal.Float()
		if f < 0 {
			return fmt.Errorf("cannot parse '%s', %f overflows uint",
				name, f)
		}
		val.SetUint(uint64(f))
	case dataKind == reflect.Bool:
		if dataVal.Bool() {
			val.SetUint(1)
		} else {
			val.SetUint(0)
		}
	case dataKind == reflect.String:
		i, err := strconv.ParseUint(dataVal.String(), 0, val.Type().Bits())
		if err == nil {
			val.SetUint(i)
		} else {
			return fmt.Errorf("cannot parse '%s' as uint: %s", name, err)
		}
	case dataType.PkgPath() == "encoding/json" && dataType.Name() == "Number":
		jn := data.(json.Number)
		i, err := jn.Int64()
		if err != nil {
			return fmt.Errorf(
				"error decoding json.Number into %s: %s", name, err)
		}
		if i < 0 {
			return fmt.Errorf("cannot parse '%s', %d overflows uint",
				name, i)
		}
		val.SetUint(uint64(i))
	default:
		return fmt.Errorf(
			"'%s' expected type '%s', got unconvertible type '%s'",
			name, val.Type(), dataVal.Type())
	}

	return nil
}

func decodeFloat(name string, data any, val reflect.Value) error {
	dataVal := reflect.Indirect(reflect.ValueOf(data))
	dataKind := getKind(dataVal)
	dataType := dataVal.Type()

	switch {
	case dataKind == reflect.Int:
		val.SetFloat(float64(dataVal.Int()))
	case dataKind == reflect.Uint:
		val.SetFloat(float64(dataVal.Uint()))
	case dataKind == reflect.Float32:
		val.SetFloat(dataVal.Float())
	case dataKind == reflect.Bool:
		if dataVal.Bool() {
			val.SetFloat(1)
		} else {
			val.SetFloat(0)
		}
	case dataKind == reflect.String:
		f, err := strconv.ParseFloat(dataVal.String(), val.Type().Bits())
		if err == nil {
			val.SetFloat(f)
		} else {
			return fmt.Errorf("cannot parse '%s' as float: %s", name, err)
		}
	case dataType.PkgPath() == "encoding/json" && dataType.Name() == "Number":
		jn := data.(json.Number)
		i, err := jn.Float64()
		if err != nil {
			return fmt.Errorf(
				"error decoding json.Number into %s: %s", name, err)
		}
		val.SetFloat(i)
	default:
		return fmt.Errorf(
			"'%s' expected type '%s', got unconvertible type '%s'",
			name, val.Type(), dataVal.Type())
	}

	return nil
}

func decodeString(name string, data any, val reflect.Value) error {
	dataVal := reflect.Indirect(reflect.ValueOf(data))
	dataKind := getKind(dataVal)

	converted := true
	switch {
	case dataKind == reflect.String:
		val.SetString(dataVal.String())
	case dataKind == reflect.Bool:
		if dataVal.Bool() {
			val.SetString("1")
		} else {
			val.SetString("0")
		}
	case dataKind == reflect.Int:
		val.SetString(strconv.FormatInt(dataVal.Int(), 10))
	case dataKind == reflect.Uint:
		val.SetString(strconv.FormatUint(dataVal.Uint(), 10))
	case dataKind == reflect.Float32:
		val.SetString(strconv.FormatFloat(dataVal.Float(), 'f', -1, 64))
	case dataKind == reflect.Slice,
		dataKind == reflect.Array:
		dataType := dataVal.Type()
		elemKind := dataType.Elem().Kind()
		switch elemKind {
		case reflect.Uint8:
			var uints []uint8
			if dataKind == reflect.Array {
				uints = make([]uint8, dataVal.Len(), dataVal.Len())
				for i := range uints {
					uints[i] = dataVal.Index(i).Interface().(uint8)
				}
			} else {
				uints = dataVal.Interface().([]uint8)
			}
			val.SetString(string(uints))
		default:
			converted = false
		}
	default:
		converted = false
	}

	if !converted {
		return fmt.Errorf(
			"'%s' expected type '%s', got unconvertible type '%s'",
			name, val.Type(), dataVal.Type())
	}

	return nil
}

func decodePtr(name string, data any, val reflect.Value) (bool, error) {
	// If the input data is nil, then we want to just set the output
	// pointer to be nil as well.
	isNil := data == nil
	if !isNil {
		switch v := reflect.Indirect(reflect.ValueOf(data)); v.Kind() {
		case reflect.Chan,
			reflect.Func,
			reflect.Interface,
			reflect.Map,
			reflect.Ptr,
			reflect.Slice:
			isNil = v.IsNil()
		}
	}
	if isNil {
		if !val.IsNil() && val.CanSet() {
			nilValue := reflect.New(val.Type()).Elem()
			val.Set(nilValue)
		}

		return true, nil
	}

	// Create an element of the concrete (non pointer) type and decode
	// into that. Then set the value of the pointer to this type.
	valType := val.Type()
	valElemType := valType.Elem()
	if val.CanSet() {
		realVal := val
		if realVal.IsNil() {
			realVal = reflect.New(valElemType)
		}

		if err := decode(name, data, reflect.Indirect(realVal)); err != nil {
			return false, err
		}

		val.Set(realVal)
	} else {
		if err := decode(name, data, reflect.Indirect(val)); err != nil {
			return false, err
		}
	}
	return false, nil
}

func getKind(val reflect.Value) reflect.Kind {
	kind := val.Kind()

	switch {
	case kind >= reflect.Int && kind <= reflect.Int64:
		return reflect.Int
	case kind >= reflect.Uint && kind <= reflect.Uint64:
		return reflect.Uint
	case kind >= reflect.Float32 && kind <= reflect.Float64:
		return reflect.Float32
	default:
		return kind
	}
}

// 下划线写法转为驼峰写法
func Case2Camel(name string) string {
	name = strings.Replace(name, "_", " ", -1)
	name = strings.Title(name)
	return strings.Replace(name, " ", "", -1)
}
