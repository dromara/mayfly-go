package tools

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// probeParams 模拟工具入参 struct：属性名与必填项都靠 json / jsonschema tag 声明，
// 与 machinetool.CommandExecParam、dbtool.SQLExecParam 同形
type probeParams struct {
	DbId   int64  `json:"dbId"`
	Sql    string `json:"sql" jsonschema:"required"`
	Remark string `json:"remark" jsonschema:"required"`
	Skip   string `json:"-"`
}

// TestDeclareParamSchemaReflectsTags 入参形状必须由反射得出，而不是再手写一份清单。
//
// 手写清单写歪一个键（实测：提案存的是 cmd 而工具收 command）只会表现为「校验永远不报错、
// 但批准后执行不了」，编译与静态检查全绿，所以真源只能是工具自己的 struct
// declareProbeSchema 注册探针工具的入参形状，并在用例结束时从全局 paramSchemas 删除。
//
// DeclareParamSchema 写的是包级全局表；测试注入的探针名若不清理，会残留到整个进程结束，
// 任何遍历 paramSchemas 的守卫/用例都会看到它们（同类污染曾让守卫如实误报内置高危工具）
func declareProbeSchema(t *testing.T, toolName string, sample any) {
	t.Helper()
	DeclareParamSchema(toolName, sample)
	t.Cleanup(func() {
		paramSchemaMu.Lock()
		defer paramSchemaMu.Unlock()
		delete(paramSchemas, toolName)
	})
}

func TestDeclareParamSchemaReflectsTags(t *testing.T) {
	declareProbeSchema(t, "ProbeTool", probeParams{})

	schema, ok := ParamSchemaOf("ProbeTool")
	require.True(t, ok, "声明后必须可查回")
	// 属性名取 json tag（模型看到的是 dbId，不是 Go 字段名 DbId）；json:"-" 的字段不参与
	require.Equal(t, []string{"dbId", "sql", "remark"}, schema.Props)
	// 必填项取 jsonschema:"required" 标记
	require.Equal(t, []string{"sql", "remark"}, schema.Required)
}

// TestValidateToolParamsRejectsMismatchedParams 提案参数与工具入参不同形时必须拒绝。
func TestValidateToolParamsRejectsMismatchedParams(t *testing.T) {
	declareProbeSchema(t, "ProbeTool", probeParams{})

	// 真实缺陷形态：键名是模型自己编的（cmd），工具根本没有这个入参。
	// 必须断言失败原因：只断 Error 的话，「缺必填」检查会顺带满足断言，
	// 未知键检查被删也测不出来（实测变异存活过一次）
	unknownErr := ValidateToolParams("ProbeTool", `{"dbId":5,"cmd":"true","sql":"select 1","remark":"r"}`)
	require.Error(t, unknownErr, "未知参数键必须拒绝")
	require.Contains(t, unknownErr.Error(), "unknown parameter", "必须是因为未知键被拒，而不是其他检查顺带拦下")
	require.Contains(t, unknownErr.Error(), "cmd", "报错要指名是哪个键，否则无法定位模型写错在哪")

	// 缺必填项：执行时必然失败，登记/下发许可时就该拦住
	missingErr := ValidateToolParams("ProbeTool", `{"dbId":5,"sql":"select 1"}`)
	require.Error(t, missingErr, "缺必填 remark 应拒绝")
	require.Contains(t, missingErr.Error(), "miss parameter")
	require.Contains(t, missingErr.Error(), "remark")
	// 合法形状放行
	require.NoError(t, ValidateToolParams("ProbeTool", `{"dbId":5,"sql":"select 1","remark":"核对行数"}`))
	// 只给必填项、省略可选属性：现在必须拒绝。
	//
	// 因为放行走的是「批准参数与执行实收参数逐项相等」，而实收参数由工具参数 struct marshal 得出、
	// 必然含全部属性；登记侧允许省略就会做出「批了但永远不放行」的提案（真机核实的矛盾）
	omitErr := ValidateToolParams("ProbeTool", `{"sql":"select 1","remark":"r"}`)
	require.Error(t, omitErr, "省略可选属性应在登记时就拒绝")
	require.Contains(t, omitErr.Error(), "dbId", "报错要点名缺哪个属性，模型才能补齐")
}

// TestValidateToolParamsSkipsUndeclaredTools 未声明形状的工具跳过校验，不猜它的参数。
//
// MCP/插件工具由运行期注册，若在这里保守拒绝，等于让所有外部工具的提案都不可执行
func TestValidateToolParamsSkipsUndeclaredTools(t *testing.T) {
	require.NoError(t, ValidateToolParams("SomeMcpTool", `{"anything":1}`), "未声明形状的工具不校验参数")
}

// TestValidateToolParamsMalformedInput 非对象参数按非法处理且不得 panic。
func TestValidateToolParamsMalformedInput(t *testing.T) {
	declareProbeSchema(t, "ProbeTool", probeParams{})

	require.Error(t, ValidateToolParams("ProbeTool", `not-json`))
	require.Error(t, ValidateToolParams("ProbeTool", `[1,2,3]`), "数组不是入参形状")
	// 空参数：工具若有必填项则空必然非法；若无必填项（未声明）走上面的跳过分支
	require.Error(t, ValidateToolParams("ProbeTool", ``), "有必填项的工具不能提交空参数")
}

// TestDeclareParamSchemaIgnoresJunkSamples 脏输入不产生半成品声明。
func TestDeclareParamSchemaIgnoresJunkSamples(t *testing.T) {
	DeclareParamSchema("", probeParams{})
	DeclareParamSchema("ProbeToolNilSample", nil)
	DeclareParamSchema("ProbeToolNotStruct", "not-a-struct")

	for _, name := range []string{"ProbeToolNilSample", "ProbeToolNotStruct"} {
		_, ok := ParamSchemaOf(name)
		require.False(t, ok, "%s 不应产生声明", name)
	}
}
