package mongodoc

import (
	"encoding/json"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestDecodeUpdateSpecAcceptsOperatorsAndPipeline(t *testing.T) {
	cases := []struct {
		name       string
		raw        string
		isPipeline bool
	}{
		{"$set 更新文档", `{"$set":{"status":"paid"}}`, false},
		{"多个操作符", `{"$set":{"a":1},"$inc":{"n":1},"$currentDate":{"at":true}}`, false},
		{"管道数组", `[{"$set":{"n":{"$add":["$n",1]}}}]`, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			spec, err := DecodeUpdateSpec(json.RawMessage(c.raw))
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if spec.IsPipeline() != c.isPipeline {
				t.Fatalf("isPipeline = %v, want %v", spec.IsPipeline(), c.isPipeline)
			}
			if spec.Value() == nil {
				t.Fatal("value must not be nil")
			}
		})
	}
}

// TestDecodeUpdateSpecRejectsReplacementDocument 拒绝「普通文档」作为批量更新体：
// 那种形状会被 driver 当成替换文档，一次按 filter 的批量写会把所有命中文档刷成同一份内容。
func TestDecodeUpdateSpecRejectsReplacementDocument(t *testing.T) {
	for _, raw := range []string{
		`{"status":"paid"}`,
		`{"_id":{"$oid":"507f1f77bcf86cd799439011"},"a":1}`,
		`{"$set":{"a":1},"extra":2}`,
		`{}`,
		`[]`,
		``,
		`null`,
		`"text"`,
	} {
		if _, err := DecodeUpdateSpec(json.RawMessage(raw)); !errors.Is(err, ErrUpdateSpecInvalid) {
			t.Fatalf("update body %q must be rejected, got %v", raw, err)
		}
	}
}

func TestDecodeUpdateSpecKeepsTypesAndOrder(t *testing.T) {
	spec, err := DecodeUpdateSpec(json.RawMessage(`{"$set":{"b":{"$numberLong":"9007199254740993"},"a":1}}`))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	doc, ok := spec.Value().(bson.D)
	if !ok {
		t.Fatalf("value type = %T, want bson.D", spec.Value())
	}
	set, ok := doc[0].Value.(bson.D)
	if !ok {
		t.Fatalf("$set value = %T, want bson.D", doc[0].Value)
	}
	if set[0].Key != "b" || set[1].Key != "a" {
		t.Fatalf("field order lost inside $set: %+v", set)
	}
	if _, ok := set[0].Value.(int64); !ok {
		t.Fatalf("$numberLong must stay int64, got %T", set[0].Value)
	}
}

func TestDecodePipeline(t *testing.T) {
	stages, err := DecodePipeline(json.RawMessage(`[{"$match":{"status":"paid"}},{"$group":{"_id":null,"total":{"$sum":"$amount"}}}]`))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(stages) != 2 || stages[0][0].Key != "$match" || stages[1][0].Key != "$group" {
		t.Fatalf("pipeline lost stage order/name: %+v", stages)
	}

	// 单对象写法、空数组、非 $ 开头的 stage 名都必须被拒
	for _, raw := range []string{`{"$match":{"a":1}}`, `[]`, `[{"match":{"a":1}}]`, `[{}]`, ``, `5`} {
		if _, err = DecodePipeline(json.RawMessage(raw)); !errors.Is(err, ErrPipelineInvalid) {
			t.Fatalf("pipeline %s must be rejected, got %v", raw, err)
		}
	}
}

func TestPipelineCommandClassifiesWriteStages(t *testing.T) {
	pipeline, err := DecodePipeline(json.RawMessage(`[{"$match":{"a":1}},{"$merge":{"into":"t2"}}]`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	name, level, err := Classify(PipelineCommand("orders", pipeline))
	if err != nil {
		t.Fatalf("classify: %v", err)
	}
	if name != "aggregate" || level != LevelDataSave {
		t.Fatalf("aggregate with $merge must be classified as data-writing, got %s/%s", name, level)
	}

	readOnly, err := DecodePipeline(json.RawMessage(`[{"$sort":{"a":-1}}]`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, level, err = Classify(PipelineCommand("orders", readOnly)); err != nil || level != LevelRead {
		t.Fatalf("read-only pipeline must stay read, got %v / %s", err, level)
	}
}
