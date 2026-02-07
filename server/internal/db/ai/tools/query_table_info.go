package tools

import (
	"context"

	"mayfly-go/internal/db/application"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type QueryTableInfoParam struct {
	DbId      uint64 `json:"dbId" jsonschema_description:"数据库ID"`
	DbName    string `json:"dbName" jsonschema_description:"数据库名称"`
	TableName string `json:"tableName" jsonschema_description:"表名"`
}

type QueryTableInfoOutput struct {
	DDL string `json:"ddl" jsonschema_description:"表DDL"`
}

func GetQueryTableInfo() (tool.InvokableTool, error) {
	return utils.InferTool("QueryTableInfo",
		"当需要了解某个表结构时，请调用此工具。使用它来查询数据库表的DDL信息，包括表结构、字段定义、索引等。",
		func(ctx context.Context, param *QueryTableInfoParam) (*QueryTableInfoOutput, error) {
			conn, err := application.GetDbApp().GetDbConn(ctx, param.DbId, param.DbName)
			if err != nil {
				return nil, err
			}

			ddl, err := conn.GetMetadata().GetTableDDL(param.TableName, false)
			if err != nil {
				return nil, err
			}
			output := &QueryTableInfoOutput{DDL: ddl}
			return output, nil
		},
	)
}
