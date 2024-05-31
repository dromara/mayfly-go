package ioc

import (
	"context"
	"errors"
	"fmt"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/structx"
	"reflect"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
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
	cType := structx.IndirectType(reflect.TypeOf(component.Value))
	// 组件名为空，则取组件类型名称作为组件名
	if componentName == "" {
		componentName = cType.Name()
		component.Name = componentName
	}

	if _, ok := c.components[componentName]; ok {
		logx.Warnf("组件名[%s]已经注册至容器, 重复注册...", componentName)
	}

	logx.Debugf("ioc register : %s = %s.%s", componentName, cType.PkgPath(), cType.Name())
	c.components[componentName] = component
}

// 注册对象实例的字段含有inject:"xxx"标签或者Setter方法，则注入对应组件实例
func (c *Container) Inject(obj any) error {
	objValue := reflect.ValueOf(obj)
	if structx.Indirect(objValue).Kind() != reflect.Struct {
		return nil
	}

	if err := c.injectWithField(objValue); err != nil {
		return err
	}
	if err := c.injectWithMethod(objValue); err != nil {
		return err
	}
	return nil
}

// 对所有组件实例执行Inject。即为实例字段注入依赖的组件实例
func (c *Container) InjectComponents() error {
	componentsGroups := collx.ArraySplit[*Component](collx.MapValues(c.components), 3)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errGroup, _ := errgroup.WithContext(ctx)

	for _, components := range componentsGroups {
		components := components // 创建局部变量以在闭包中使用
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

// 根据组件实例名，获取对应实例信息
func (c *Container) Get(name string) (any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	component, ok := c.components[name]
	if !ok {
		return nil, errors.New("component not found: " + name)
	}

	return component.Value, nil
}

// 根据实例字段的inject:"xxx"标签进行依赖注入
func (c *Container) injectWithField(objValue reflect.Value) error {
	objValue = structx.Indirect(objValue)
	objType := objValue.Type()

	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)

		componentName, ok := field.Tag.Lookup("inject")
		if !ok {
			continue
		}
		// inject tag字段名为空则默认为字段名
		if componentName == "" {
			componentName = field.Name
		}

		injectInfo := fmt.Sprintf("ioc field inject [%s -> %s.%s#%s]", componentName, objType.PkgPath(), objType.Name(), field.Name)
		logx.Debugf(injectInfo)

		component, err := c.Get(componentName)
		if err != nil {
			return fmt.Errorf("%s error: %s", injectInfo, err.Error())
		}

		// 判断字段类型与需要注入的组件类型是否为可赋值关系
		componentType := reflect.TypeOf(component)
		if !componentType.AssignableTo(field.Type) {
			componentType = structx.IndirectType(componentType)
			return fmt.Errorf("%s error: 注入类型不一致(期望类型->%s.%s, 组件类型->%s.%s)", injectInfo, field.Type.PkgPath(), field.Type.Name(), componentType.PkgPath(), componentType.Name())
		}

		fieldValue := objValue.Field(i)
		if !fieldValue.IsValid() || !fieldValue.CanSet() {
			// 不可导出变量处理
			fieldPtrValue := reflect.NewAt(fieldValue.Type(), fieldValue.Addr().UnsafePointer())
			fieldValue = fieldPtrValue.Elem()
			if !fieldValue.IsValid() || !fieldValue.CanSet() {
				return fmt.Errorf("%s error: 字段无效或为不可导出类型", injectInfo)
			}
		}

		fieldValue.Set(reflect.ValueOf(component))
	}

	return nil
}

// 根据实例的Inject方法进行依赖注入
func (c *Container) injectWithMethod(objValue reflect.Value) error {
	objType := objValue.Type()

	for i := 0; i < objType.NumMethod(); i++ {
		method := objType.Method(i)

		methodName := method.Name
		// 不是以Inject开头的函数，则默认跳过
		if !strings.HasPrefix(methodName, "Inject") {
			continue
		}

		// 获取组件名，InjectTestApp -> TestApp
		componentName := methodName[6:]

		injectInfo := fmt.Sprintf("ioc method inject [%s.%s#%s(%s)]", objType.Elem().PkgPath(), objType.Elem().Name(), methodName, componentName)
		logx.Debugf(injectInfo)

		if method.Type.NumIn() != 2 {
			logx.Warnf("%s error: 方法入参不为1个, 无法进行注入", injectInfo)
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
			return fmt.Errorf("%s error: 注入类型不一致(期望类型->%s.%s, 组件类型->%s.%s)", injectInfo, expectedComponentType.PkgPath(), expectedComponentType.Name(), componentType.PkgPath(), componentType.Name())
		}

		method.Func.Call([]reflect.Value{objValue, reflect.ValueOf(component)})
	}

	return nil
}
