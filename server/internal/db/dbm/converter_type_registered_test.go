package dbm

// 异构迁移「转换产出类型 → 类型注册表」闭环守卫（无环境依赖）。
//
// 每个方言的 CommonTypeConverter 在任意列形态下产出的目标类型，必须能在该方言自己的
// 类型注册表中按 Name 命中，且命中的 DataType 与产出对象一致。
//
// 为何必须钉死：dump/导入生成 INSERT 时，值转 SQL 是按「目标列类型名」查注册表取 SQLValue
// （如 mssql/sqlgen.go 的 dbi.GetDbDataType(DbTypeMssql, column.DataType).DataType.SQLValue(v)），
// 查不到会**静默回退** DefaultDbDataType（普通字符串字面量）。本机实测：mssql 的
// varchar(max)/nvarchar(max)/varbinary(max) 只由异构转换产出、曾漏注册，使 BLOB 迁入
// SQL Server 时值被输出为 '00ff' 字面量（报 Implicit conversion from varchar to varbinary(max)），
// 大文本列同样失去方言专有的字面量语义（如 mssql 的 N'' Unicode 前缀）。
//
// 本用例用反射遍历转换器的全部公共类型方法，新增方言、新增 CT 或新增目标类型时自动纳入校验。

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

// converterColumnShapes 转换方法入参的代表性列形态：长度/精度组合覆盖各转换器的分支边界
// （定长上限、(max) 退化、无精度、带小数位）
var converterColumnShapes = []dbi.Column{
	{ColumnName: "c"},
	{ColumnName: "c", CharMaxLength: 1},
	{ColumnName: "c", CharMaxLength: 4000},
	{ColumnName: "c", CharMaxLength: 8000},
	{ColumnName: "c", CharMaxLength: 65535},
	{ColumnName: "c", CharMaxLength: 1 << 30},
	{ColumnName: "c", NumPrecision: 30, NumScale: 10},
	{ColumnName: "c", NumPrecision: 3},
}

func TestConverterOutputTypesRegistered(t *testing.T) {
	for _, dt := range expectedDbTypes {
		dt := dt
		converter := dbi.GetMeta(dt).GetCommonTypeConverter()
		require.NotNil(t, converter, "[%s] 未提供公共类型转换器", dt)

		cv := reflect.ValueOf(converter)
		for i := 0; i < cv.NumMethod(); i++ {
			methodName := cv.Type().Method(i).Name
			t.Run(string(dt)+"/"+methodName, func(t *testing.T) {
				for _, shape := range converterColumnShapes {
					shape := shape
					col := shape
					out := cv.Method(i).Call([]reflect.Value{reflect.ValueOf(&col)})
					require.Len(t, out, 1, "[%s] 转换方法 %s 返回值个数异常", dt, methodName)
					require.False(t, out[0].IsNil(), "[%s] 转换方法 %s 对列 %+v 返回nil（异构迁移会静默丢列类型）",
						dt, methodName, shape)
					target, ok := out[0].Interface().(*dbi.DbDataType)
					require.True(t, ok, "[%s] 转换方法 %s 返回类型异常", dt, methodName)
					require.NotEmpty(t, target.Name, "[%s] 转换方法 %s 返回无名类型", dt, methodName)

					// 按名回查注册表：命中默认类型即意味着值转SQL丢失该方言的专有字面量语义
					resolved := dbi.GetDbDataType(dt, target.Name)
					assert.NotSame(t, dbi.DefaultDbDataType, resolved,
						"[%s] %s产出的类型[%s]未在类型注册表中，INSERT值转SQL将静默回退默认字符串类型",
						dt, methodName, target.Name)
					// 注册表以小写类型名为键，但Name保留方言原始大小写（如oracle的BLOB），故按大小写无关比较
					assert.True(t, strings.EqualFold(target.Name, resolved.Name),
						"[%s] 类型[%s]按名回查命中了 %s", dt, target.Name, resolved.Name)
					assert.Same(t, target.DataType, resolved.DataType,
						"[%s] 类型[%s]注册对象的值转换者与产出者不一致（SQLValue/Valuer会按注册对象执行）",
						dt, target.Name)
				}
			})
		}
	}
}
