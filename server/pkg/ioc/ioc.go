package ioc

import (
	"context"
	"errors"
	"fmt"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/structx"
	"reflect"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

const (
	InjectTag           = "inject"
	InjectMethodPrefix  = "Inject"
	ByTypeComponentName = "T" // 根据类型注入的组件名
)

// 容器
type Container struct {
	mu         sync.RWMutex
	components map[string]*Component
}

func NewContainer() *Container {
	return &Container{
		components: make(map[string]*Component),
	}
}

// 注册实例至实例容器
func (c *Container) Register(bean any, opts ...ComponentOption) {
	c.mu.Lock()
	defer c.mu.Unlock()

	component := NewComponent(bean, opts...)

	componentName := component.Name
	indirectCType := structx.IndirectType(component.GetType())
	// 组件名为空，则取组件类型`包名路径.名称`作为组件名
	if componentName == "" {
		componentName = fmt.Sprintf("%s.%s", indirectCType.PkgPath(), indirectCType.Name())
		component.Name = componentName
	}

	if _, ok := c.components[componentName]; ok {
		logx.Warnf("the component name [%s] has been registered to the container. Repeat the registration...", componentName)
	}

	logx.Debugf("ioc register : %s = %s.%s", componentName, indirectCType.PkgPath(), indirectCType.Name())
	c.components[componentName] = component
}

// Inject 注册对象实例的字段含有注入标签或者Setter方法，则注入对应组件实例
func (c *Container) Inject(obj any) error {
	objValue := reflect.ValueOf(obj)
	if structx.Indirect(objValue).Kind() != reflect.Struct {
		return nil
	}

	ctx := contextx.NewTraceId()
	if err := c.injectWithField(ctx, objValue); err != nil {
		return err
	}
	if err := c.injectWithMethod(ctx, objValue); err != nil {
		return err
	}
	return nil
}

// 对所有组件实例执行Inject。即为实例字段注入依赖的组件实例
func (c *Container) InjectComponents() error {
	componentsGroups := collx.ArraySplit[*Component](collx.MapValues(c.components), 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errGroup, _ := errgroup.WithContext(ctx)

	for _, components := range componentsGroups {
		errGroup.Go(func() error {
			for _, v := range components {
				if err := c.Inject(v.Value); err != nil {
					cancel() // 取消所有协程的执行
					return err
				}
			}
			return nil
		})
	}

	if err := errGroup.Wait(); err != nil {
		return err
	}

	return nil
}

// Get 根据组件实例名，获取对应实例信息
func (c *Container) Get(name string) (any, error) {
	if comp, err := c.GetComponent(name); err == nil {
		return comp.Value, nil
	} else {
		return nil, err
	}
}

// GetComponent 根据组件名，获取对应组件信息
func (c *Container) GetComponent(name string) (*Component, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	component, ok := c.components[name]
	if !ok {
		return nil, errors.New("component not found: " + name)
	}

	return component, nil
}

// GetByType 根据组件实例类型获取组件实例
func (c *Container) GetByType(fieldType reflect.Type) (any, error) {
	if comp, err := c.GetComponentByType(fieldType); err == nil {
		return comp.Value, nil
	} else {
		return nil, err
	}
}

// GetComponentByType 根据组件实例类型获取组件信息
func (c *Container) GetComponentByType(fieldType reflect.Type) (*Component, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, component := range c.components {
		if component.GetType().AssignableTo(fieldType) {
			return component, nil
		}
	}

	return nil, errors.New("component type not found: " + fmt.Sprintf("%s.%s", fieldType.PkgPath(), fieldType.Name()))
}

// GetBeansByType 根据组件实例类型获取所有对应类型的组件实例
func (c *Container) GetBeansByType(fieldType reflect.Type) []any {
	return collx.ArrayMap(c.GetComponentsByType(fieldType), func(comp *Component) any { return comp.Value })
}

// GetComponentsByType 根据组件实例类型获取指定类型的所有组件信息
func (c *Container) GetComponentsByType(fieldType reflect.Type) []*Component {
	c.mu.RLock()
	defer c.mu.RUnlock()

	components := make([]*Component, 0)
	for _, component := range c.components {
		if component.GetType().AssignableTo(fieldType) {
			components = append(components, component)
		}
	}

	return components
}

// injectWithField 根据实例字段的inject:"xxx"标签进行依赖注入
func (c *Container) injectWithField(context context.Context, objValue reflect.Value) error {
	objValue = structx.Indirect(objValue)
	objType := objValue.Type()

	logx.DebugfContext(context, "start ioc inject with field: %s.%s", objType.PkgPath(), objType.Name())

	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		fieldValue := objValue.Field(i)

		// 检查字段是否是通过组合包含在当前结构体中的，即嵌套结构体
		if field.Anonymous && structx.IndirectType(field.Type).Kind() == reflect.Struct {
			c.injectWithField(context, fieldValue)
			continue
		}

		componentName, ok := field.Tag.Lookup(InjectTag)
		if !ok {
			continue
		}

		// 如果组件名为指定的根据类型注入值，则根据类型注入
		if componentName == ByTypeComponentName {
			if err := c.injectByType(context, objType, field, fieldValue); err != nil {
				return err
			}
			continue
		}

		if err := c.injectByName(context, objType, field, fieldValue, componentName); err != nil {
			return err
		}
	}

	return nil
}

// injectByName 根据实例组件名进行依赖注入
func (c *Container) injectByName(context context.Context, structType reflect.Type, field reflect.StructField, fieldValue reflect.Value, componentName string) error {
	// inject tag字段名为空则默认为字段名
	if componentName == "" {
		componentName = field.Name
	}

	injectInfo := fmt.Sprintf("ioc field inject by name => [%s -> %s.%s#%s]", componentName, structType.PkgPath(), structType.Name(), field.Name)

	component, err := c.GetComponent(componentName)
	if err != nil {
		return fmt.Errorf("%s error: %s", injectInfo, err.Error())
	}

	// 判断字段类型与需要注入的组件类型是否为可赋值关系
	componentType := component.GetType()
	if !componentType.AssignableTo(field.Type) {
		indirectComponentType := structx.IndirectType(componentType)
		return fmt.Errorf("%s error: injection types are inconsistent(Expected type -> %s.%s, Component type -> %s.%s)", injectInfo, field.Type.PkgPath(), field.Type.Name(), indirectComponentType.PkgPath(), indirectComponentType.Name())
	}

	logx.DebugfContext(context, "ioc field inject by name => [%s (%s) -> %s.%s#%s]", componentName, getComponentValueDesc(componentType), structType.PkgPath(), structType.Name(), field.Name)

	if err := setFieldValue(fieldValue, component.Value); err != nil {
		return fmt.Errorf("%s error: %s", injectInfo, err.Error())
	}

	return nil
}

// injectByType 根据实例类型进行依赖注入
func (c *Container) injectByType(context context.Context, structType reflect.Type, field reflect.StructField, fieldValue reflect.Value) error {
	fieldType := field.Type

	injectInfo := fmt.Sprintf("ioc field inject by type => [%s.%s -> %s.%s#%s]", fieldType.PkgPath(), fieldType.Name(), structType.PkgPath(), structType.Name(), field.Name)

	component, err := c.GetComponentByType(fieldType)
	if err != nil {
		return fmt.Errorf("%s error: %s", injectInfo, err.Error())
	}

	logx.DebugfContext(context, "ioc field inject by type => [%s.%s (%s) -> %s.%s#%s]", fieldType.PkgPath(), fieldType.Name(), getComponentValueDesc(component.GetType()), structType.PkgPath(), structType.Name(), field.Name)

	if err := setFieldValue(fieldValue, component.Value); err != nil {
		return fmt.Errorf("%s error: %s", injectInfo, err.Error())
	}

	return nil
}

// 根据实例的Inject方法进行依赖注入
func (c *Container) injectWithMethod(context context.Context, objValue reflect.Value) error {
	objType := objValue.Type()

	for i := 0; i < objType.NumMethod(); i++ {
		method := objType.Method(i)

		methodName := method.Name
		// 不是以指定方法名前缀开头的函数，则默认跳过
		if !strings.HasPrefix(methodName, InjectMethodPrefix) {
			continue
		}

		// 获取组件名，InjectTestApp -> TestApp
		componentName := methodName[6:]

		injectInfo := fmt.Sprintf("ioc method inject [%s.%s#%s(%s)]", objType.Elem().PkgPath(), objType.Elem().Name(), methodName, componentName)
		logx.DebugfContext(context, injectInfo)

		if method.Type.NumIn() != 2 {
			logx.WarnfContext(context, "%s error: the method cannot be injected if it does not have one parameter", injectInfo)
			continue
		}

		component, err := c.Get(componentName)
		if err != nil {
			return fmt.Errorf("%s error: %s", injectInfo, err.Error())
		}

		componentType := reflect.TypeOf(component)
		// 期望的组件类型，即参数入参类型
		expectedComponentType := method.Type.In(1)
		if !componentType.AssignableTo(expectedComponentType) {
			componentType = structx.IndirectType(componentType)
			return fmt.Errorf("%s error: injection types are inconsistent(Expected type -> %s.%s, Component type -> %s.%s)", injectInfo, expectedComponentType.PkgPath(), expectedComponentType.Name(), componentType.PkgPath(), componentType.Name())
		}

		method.Func.Call([]reflect.Value{objValue, reflect.ValueOf(component)})
	}

	return nil
}

func setFieldValue(fieldValue reflect.Value, component any) error {
	if !fieldValue.IsValid() || !fieldValue.CanSet() {
		// 不可导出变量处理
		fieldPtrValue := reflect.NewAt(fieldValue.Type(), fieldValue.Addr().UnsafePointer())
		fieldValue = fieldPtrValue.Elem()
		if !fieldValue.IsValid() || !fieldValue.CanSet() {
			return errors.New("the field is invalid or a non-exportable type")
		}
	}

	fieldValue.Set(reflect.ValueOf(component))
	return nil
}

func getComponentValueDesc(componentValueType reflect.Type) string {
	if componentValueType.Kind() == reflect.Ptr {
		componentValueType = structx.IndirectType(componentValueType)
		return fmt.Sprintf("*%s.%s", componentValueType.PkgPath(), componentValueType.Name())
	}
	return fmt.Sprintf("%s.%s", componentValueType.PkgPath(), componentValueType.Name())
}
