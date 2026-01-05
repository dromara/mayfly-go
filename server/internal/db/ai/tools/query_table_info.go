package tools

import (
	"context"

	"mayfly-go/internal/db/application"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/jsonx"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

func GetQueryTableInfo() tool.InvokableTool {
	return &QueryTableInfo{}
}

type QueryTableInfo struct {
}

var _ tool.InvokableTool = (*QueryTableInfo)(nil)

func (q QueryTableInfo) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "QueryTableInfo",
		Desc: "查询数据库表的详细信息，包括表结构、字段定义、索引等。当用户需要了解某个表的结构时使用此工具。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"dbId": {
				Type:     schema.Number,
				Desc:     "数据库ID",
				Required: true,
			},
			"dbName": {
				Type:     schema.String,
				Desc:     "数据库名称",
				Required: true,
			},
			"tableName": {
				Type:     schema.String,
				Desc:     "表名",
				Required: true,
			},
		}),
	}, nil
}

func (q QueryTableInfo) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	logx.Debugf("开始查询数据库表信息: %s", argumentsInJSON)
	m, err := jsonx.ToMap(argumentsInJSON)
	if err != nil {
		return "arguments json invalid", err
	}

	tableName := m.GetStr("tableName")
	conn, err := application.GetDbApp().GetDbConn(ctx, uint64(m.GetInt64("dbId")), m.GetStr("dbName"))
	if err != nil {
		return "获取数据库连接失败", err
	}

	return conn.GetMetadata().GetTableDDL(tableName, false)

}
