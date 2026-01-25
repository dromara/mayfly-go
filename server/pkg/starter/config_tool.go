package starter

import (
	"fmt"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/structx"
	"reflect"
	"strconv"
	"strings"
)

// ConfigItem 配置项
type ConfigItem interface {
	// 如果不存在配置值，则设置默认值，并校验配置项
	ApplyDefaults() error
}

// ConfigError 配置错误类型
type ConfigError struct {
	Field   string // 字段名
	Message string // 错误描述
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("配置错误: [%s] %s", e.Field, e.Message)
}

// NewConfigError 创建配置错误
func NewConfigError(field, message string) error {
	return &ConfigError{
		Field:   field,
		Message: message,
	}
}

// LogDefaultValue 记录配置默认值日志
func LogDefaultValue(field, defaultValue any) {
	logx.Warnf("配置项 [%s] 未设置，使用默认值: %v", field, defaultValue)
}

// ApplyConfigDefaults 应用所有配置项的默认值，支持递归处理嵌套结构体和ConfigItem接口
func ApplyConfigDefaults(obj any, parentConfigPath ...string) error {
	parentPath := strings.Join(parentConfigPath, ".")

	configValue := reflect.ValueOf(obj).Elem()
	configType := configValue.Type()

	for i := 0; i < configValue.NumField(); i++ {
		field := configValue.Field(i)
		fieldType := configType.Field(i)

		// 检查字段是否为导出的字段
		if !field.CanSet() || !field.CanAddr() {
			continue
		}

		// 如果字段是指针类型，需要先创建实例
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				newValue := reflect.New(field.Type().Elem())
				field.Set(newValue)
			}
			field = field.Elem()
		}

		// 获取字段地址，用于调用方法
		fieldAddr := field.Addr()

		// 检查字段是否实现了 ConfigItem 接口
		if fieldAddr.Type().Implements(reflect.TypeOf((*ConfigItem)(nil)).Elem()) {
			// 实现了接口，调用 ApplyDefaults 方法
			method := fieldAddr.MethodByName("ApplyDefaults")
			if method.IsValid() {
				results := method.Call(nil)
				if len(results) > 0 {
					if err, ok := results[0].Interface().(error); ok && err != nil {
						return err
					}
				}
			}
		} else if field.Kind() == reflect.Struct {
			// 为嵌套结构体确定其路径组件
			structPath := getFieldName(fieldType)

			var currentPath string
			if parentPath != "" {
				currentPath = parentPath + "." + structPath
			} else {
				currentPath = structPath
			}

			if err := ApplyConfigDefaults(fieldAddr.Interface(), currentPath); err != nil {
				return err
			}
		} else {
			// 处理普通字段的默认值设置，传递父路径
			if err := applyConfigFieldDefaults(field, fieldType, parentPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// applyConfigFieldDefaults 处理单个字段的默认值设置和选项验证等
func applyConfigFieldDefaults(field reflect.Value, fieldType reflect.StructField, parentPath string) error {
	// 构建当前字段的完整配置路径
	currentPath := getFieldName(fieldType)
	if parentPath != "" {
		currentPath = parentPath + "." + currentPath
	}

	// 检查是否已设置值
	if structx.IsZeroValue(field) {
		defaultTag := fieldType.Tag.Get("default")
		if defaultTag == "" {
			return nil
		}

		if err := setFieldValue(field, defaultTag); err != nil {
			return fmt.Errorf("设置字段 [%s] 的默认值失败: %v", currentPath, err)
		}

		LogDefaultValue(currentPath, defaultTag)
	} else {
		// 检查选项验证
		optionsTag := fieldType.Tag.Get("options")
		if optionsTag != "" {
			if err := validateOptions(currentPath, field, fieldType, optionsTag); err != nil {
				return err
			}
		}
	}

	return nil
}

// setFieldValue 设置字段值
func setFieldValue(field reflect.Value, defaultValue string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(defaultValue)
	case reflect.Bool:
		val, err := strconv.ParseBool(defaultValue)
		if err != nil {
			return err
		}
		field.SetBool(val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(defaultValue, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(defaultValue, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(val)
	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(defaultValue, 64)
		if err != nil {
			return err
		}
		field.SetFloat(val)
	default:
		return fmt.Errorf("不支持的类型: %v", field.Kind())
	}
	return nil
}

// getFieldName 获取字段名称，用于日志输出
func getFieldName(field reflect.StructField) string {
	yamlTag := field.Tag.Get("yaml")
	if yamlTag != "" && yamlTag != "-" {
		return yamlTag
	}
	return strings.ToLower(field.Name)
}

// validateOptions 验证字段值是否在允许的选项范围内
func validateOptions(configPath string, field reflect.Value, typeField reflect.StructField, optionsTag string) error {
	// 解析选项列表
	options := strings.Split(optionsTag, ",")
	if len(options) == 0 {
		return nil
	}

	// 清理选项空格
	for i, option := range options {
		options[i] = strings.TrimSpace(option)
	}

	var fieldValue string
	switch field.Kind() {
	case reflect.String:
		fieldValue = field.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldValue = strconv.FormatInt(field.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fieldValue = strconv.FormatUint(field.Uint(), 10)
	case reflect.Bool:
		fieldValue = strconv.FormatBool(field.Bool())
	default:
		return fmt.Errorf("字段 %s 不支持选项验证，类型: %v", typeField.Name, field.Kind())
	}

	// 检查字段值是否在允许的选项中
	for _, option := range options {
		if fieldValue == option {
			return nil
		}
	}

	return NewConfigError(configPath, fmt.Sprintf("值 '%s' 不在允许的选项范围内: [%s]", fieldValue, strings.Join(options, ", ")))
}
