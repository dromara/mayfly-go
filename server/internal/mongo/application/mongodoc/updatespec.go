package mongodoc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ErrUpdateSpecInvalid 批量更新的更新体形状不合法。
var ErrUpdateSpecInvalid = errors.New("mongodoc: invalid update specification, expected an update document with $ operators or an aggregation pipeline")

// UpdateSpec 批量更新的更新体。
//
// 只接受两种形状：
//   - 全 `$` 前缀键的更新文档（`{"$set":{...}}`、`{"$inc":{...}}` 等）；
//   - 聚合管道数组（MongoDB 4.2+ 的支持「按管道更新」）。
//
// 之所以不接受「普通文档」：那样会被 driver 当成替换文档，一条按 filter 命中的批量更新
// 会把所有命中文档替换成同一份内容 —— 与调用方「改字段」的直觉完全相反，属于高危静默破坏。
type UpdateSpec struct {
	update     bson.D
	pipeline   bson.A
	isPipeline bool
}

// DecodeUpdateSpec 解析并校验更新体。
func DecodeUpdateSpec(raw json.RawMessage) (*UpdateSpec, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, fmt.Errorf("%w: content is empty", ErrUpdateSpecInvalid)
	}

	if trimmed[0] == '[' {
		var pipeline bson.A
		if err := bson.UnmarshalExtJSON(trimmed, false, &pipeline); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrUpdateSpecInvalid, err.Error())
		}
		if len(pipeline) == 0 {
			return nil, fmt.Errorf("%w: pipeline is empty", ErrUpdateSpecInvalid)
		}
		for _, stage := range pipeline {
			if _, ok := stage.(bson.D); !ok {
				return nil, fmt.Errorf("%w: pipeline stage must be a document", ErrUpdateSpecInvalid)
			}
		}
		return &UpdateSpec{pipeline: pipeline, isPipeline: true}, nil
	}

	// 标量与字符串不可能是更新体：先按形状拦住，才能报出「更新体必须是 $ 操作符文档」
	// 这个准确说法，而不是「不是合法 JSON」这种误导人去找逗号的话
	if trimmed[0] != '{' {
		return nil, fmt.Errorf("%w: content is neither an update document nor a pipeline array", ErrUpdateSpecInvalid)
	}

	doc, err := Decode(trimmed)
	if err != nil {
		return nil, err
	}
	if len(doc) == 0 {
		return nil, fmt.Errorf("%w: update document is empty", ErrUpdateSpecInvalid)
	}
	for _, e := range doc {
		if len(e.Key) == 0 || e.Key[0] != '$' {
			return nil, fmt.Errorf("%w: field %q is not an update operator", ErrUpdateSpecInvalid, e.Key)
		}
	}
	return &UpdateSpec{update: doc}, nil
}

// Value 交给 driver 的更新体（两种形状的 driver 入参形态不同）。
func (s *UpdateSpec) Value() any {
	if s.isPipeline {
		return mongoPipeline(s.pipeline)
	}
	return s.update
}

// IsPipeline 报告更新体是否为聚合管道形态。
func (s *UpdateSpec) IsPipeline() bool { return s.isPipeline }

// mongoPipeline driver 要求管道是 []bson.D，bson.A 会被拒
func mongoPipeline(pipeline bson.A) []bson.D {
	res := make([]bson.D, 0, len(pipeline))
	for _, stage := range pipeline {
		res = append(res, stage.(bson.D))
	}
	return res
}
