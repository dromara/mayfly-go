package keyvalue

import (
	"mayfly-go/internal/redis/domain/entity"

	"mayfly-go/pkg/utils/collx"
)

// 视角描述符的声明式构造助手：让每个处理器的文件读起来像「该类型的 UI 契约」，
// 而契约与命令实现同处一个文件，新增类型不需要触碰其他文件

// consoleKeyPlaceholder 命令模板里的 key 占位符：ConsoleHints 写的是「GET {key}」这类模板，
// 前端渲染快捷命令时换成当前 key 名，因此模板里的带参命令必须含该占位符
const consoleKeyPlaceholder = "{key}"

func form(fields ...entity.FormField) *entity.FormSchema {
	return &entity.FormSchema{Version: 1, Cols: 1, Fields: fields}
}

func textField(prop, label string, required bool) entity.FormField {
	field := entity.FormField{Prop: prop, Label: label, Type: "input"}
	if required {
		field.Rules = &entity.FormRules{Required: true}
	}
	return field
}

func textFieldWith(prop, label, placeholder string, required bool) entity.FormField {
	field := textField(prop, label, required)
	field.Placeholder = placeholder
	return field
}

func numberField(prop, label string, required bool) entity.FormField {
	field := entity.FormField{Prop: prop, Label: label, Type: "number"}
	if required {
		field.Rules = &entity.FormRules{Required: true}
	}
	return field
}

func numberFieldWith(prop, label, description string, required bool) entity.FormField {
	field := numberField(prop, label, required)
	field.Description = description
	return field
}

func textAreaField(prop, label string, rows int, required bool) entity.FormField {
	field := entity.FormField{Prop: prop, Label: label, Type: "textarea", Rows: rows}
	if required {
		field.Rules = &entity.FormRules{Required: true}
	}
	return field
}

func selectField(prop, label string, options ...entity.FormOption) entity.FormField {
	return entity.FormField{Prop: prop, Label: label, Type: "select", Options: options, DefaultValue: options[0].Value}
}

func switchField(prop, label string) entity.FormField {
	return entity.FormField{Prop: prop, Label: label, Type: "switch"}
}

func option(value any, label string) entity.FormOption {
	return entity.FormOption{Value: value, Label: label}
}

func column(field, label, kind string, width int) entity.Column {
	return entity.Column{Field: field, Label: label, Value: kind, Width: width}
}

func sortableColumn(field, label, kind string, width int) entity.Column {
	col := column(field, label, kind, width)
	col.Sortable = true
	return col
}

func op(name, label string, write bool, schema *entity.FormSchema) entity.OpSpec {
	return entity.OpSpec{Name: name, Label: label, Write: write, Form: schema}
}

// withoutColumn / withoutFormField / withoutOp 按名摘除描述符里的列、表单项与操作，
// 供「按实例能力裁剪描述符」的视角使用：一律返回新切片，绝不改动进程级共享的描述符数组

func withoutColumn(columns []entity.Column, field string) []entity.Column {
	return collx.ArrayFilter(columns, func(col entity.Column) bool { return col.Field != field })
}

func withoutFormField(schema *entity.FormSchema, prop string) *entity.FormSchema {
	if schema == nil {
		return nil
	}
	trimmed := *schema
	trimmed.Fields = collx.ArrayFilter(schema.Fields, func(field entity.FormField) bool { return field.Prop != prop })
	return &trimmed
}

func withoutOp(ops []entity.OpSpec, name string) []entity.OpSpec {
	return collx.ArrayFilter(ops, func(item entity.OpSpec) bool { return item.Name != name })
}

func withDefault(desc *entity.ViewDescriptor) *entity.ViewDescriptor {
	desc.Default = true
	return desc
}
