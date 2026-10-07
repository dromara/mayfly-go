package mongodoc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ErrPipelineInvalid 聚合管道形状不合法。
var ErrPipelineInvalid = errors.New("mongodoc: pipeline must be a non-empty array of stages whose keys start with $")

// DecodePipeline 解析聚合管道。
//
// 要求是「非空数组 + 每项都是以 $ 开头的 stage 文档」：
//   - 数组：单个 stage 写成对象是常见笔误，直接接受会让调用方以为管道生效了；
//   - `$` 前缀：stage 名一律带 $，不带的大多是把 `$set` 写成了 `set`，
//     而 Mongo 对此报的是「unrecognized pipeline stage name」，在本层拦下能给出更准的提示。
func DecodePipeline(raw json.RawMessage) ([]bson.D, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, ErrPipelineInvalid
	}

	var stages bson.A
	if err := bson.UnmarshalExtJSON(trimmed, false, &stages); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrPipelineInvalid, err.Error())
	}
	if len(stages) == 0 {
		return nil, ErrPipelineInvalid
	}

	res := make([]bson.D, 0, len(stages))
	for _, item := range stages {
		stage, ok := item.(bson.D)
		if !ok || len(stage) == 0 || stage[0].Key == "" || stage[0].Key[0] != '$' {
			return nil, ErrPipelineInvalid
		}
		res = append(res, stage)
	}
	return res, nil
}

// PipelineCommand 组装用于分级判定的聚合命令文档。
//
// 鉴权与前端确认都依赖 Classify，而 Classify 的入参是命令文档，这里负责把管道还原成那个形状。
func PipelineCommand(collection string, pipeline []bson.D) bson.D {
	return bson.D{
		{Key: "aggregate", Value: collection},
		{Key: "pipeline", Value: pipeline},
	}
}
