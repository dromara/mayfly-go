package mongodoc

import (
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestBuildUpdateDiffNoChange(t *testing.T) {
	current := bson.D{{Key: FieldID, Value: bson.NewObjectID()}, {Key: "name", Value: "x"}, {Key: "n", Value: int32(1)}}

	update, changed, err := BuildUpdateDiff(current, current)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if changed {
		t.Fatalf("identical document should produce no write, got %+v", update)
	}
	if update != nil {
		t.Fatalf("no-change update must be nil, got %+v", update)
	}
}

func TestBuildUpdateDiffSetAndUnset(t *testing.T) {
	id := bson.NewObjectID()
	created := bson.NewDateTimeFromTime(time.Unix(1700000000, 0))

	current := bson.D{{Key: FieldID, Value: id}, {Key: "name", Value: "old"}, {Key: "keep", Value: "same"}, {Key: "dropped", Value: int32(1)}}
	// 编辑后：改 name、删掉 dropped、新增 created
	edited := bson.D{{Key: FieldID, Value: id}, {Key: "name", Value: "new"}, {Key: "keep", Value: "same"}, {Key: "created", Value: created}}

	update, changed, err := BuildUpdateDiff(current, edited)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !changed {
		t.Fatal("expect changed")
	}

	set := firstSection(update, "$set")
	if set == nil {
		t.Fatalf("expect $set section, got %+v", update)
	}
	if keys := fieldKeys(set); len(keys) != 2 || keys[0] != "name" || keys[1] != "created" {
		t.Fatalf("$set fields = %v, want [name created]（未改动字段不得被重写）", keys)
	}
	if set[1].Value != created {
		t.Fatalf("$set lost the new field value: %+v", set[1])
	}

	unset := firstSection(update, "$unset")
	if unset == nil || len(unset) != 1 || unset[0].Key != "dropped" {
		t.Fatalf("$unset = %+v, want [dropped]（从 JSON 删 key 必须真的删除字段）", unset)
	}
}

// TestBuildUpdateDiffIDImmutable 编辑后文档里改掉或保留的 _id 都不会进入更新指令：
// 主键不可变由服务端保证，不依赖前端记得删掉它。
func TestBuildUpdateDiffIDImmutable(t *testing.T) {
	id := bson.NewObjectID()
	current := bson.D{{Key: FieldID, Value: id}, {Key: "a", Value: int32(1)}}

	if _, changed, err := BuildUpdateDiff(current, bson.D{{Key: FieldID, Value: id}, {Key: "a", Value: int32(1)}}); err != nil || changed {
		t.Fatalf("same document must not trigger a write, got changed=%v err=%v", changed, err)
	}

	// 即使编辑后换上了形状完全不同的 _id，主键改动也被忽略，只按其余字段判定差异
	update, changed, err := BuildUpdateDiff(current, bson.D{{Key: FieldID, Value: "hacked-id"}, {Key: "a", Value: int32(1)}, {Key: "b", Value: int32(2)}})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !changed {
		t.Fatal("expect changed for new field b")
	}
	if hasField(update, FieldID) {
		t.Fatalf("_id must never be written: %+v", update)
	}
	if keys := fieldKeys(firstSection(update, "$set")); len(keys) != 1 || keys[0] != "b" {
		t.Fatalf("$set = %v, want [b]", keys)
	}
}

// TestBuildUpdateDiffTypeChangeIsChange 值相等但类型不同必须视为改动。
//
// 这是本次重构的语义底线：旧实现里 int64 读出来是数字、写回去成 int32 会被认为「什么都没变」，
// 于是类型损坏连一次写入都不触发，用户完全无从察觉。
func TestBuildUpdateDiffTypeChangeIsChange(t *testing.T) {
	current := bson.D{{Key: FieldID, Value: bson.NewObjectID()}, {Key: "n", Value: int64(9007199254740993)}}
	edited := bson.D{{Key: FieldID, Value: bson.NewObjectID()}, {Key: "n", Value: 9007199254740993.0}}

	update, changed, err := BuildUpdateDiff(current, edited)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !changed {
		t.Fatal("int64 -> double must be treated as a modification")
	}
	set := firstSection(update, "$set")
	if _, ok := set[0].Value.(float64); !ok {
		t.Fatalf("$set must carry the submitted type, got %T", set[0].Value)
	}
}

// TestBuildUpdateDiffNumericEqualitySameType 同类型同值才算没变，避免每次保存都产生空写。
func TestBuildUpdateDiffNumericEqualitySameType(t *testing.T) {
	current := bson.D{{Key: FieldID, Value: bson.NewObjectID()}, {Key: "n", Value: int32(7)}, {Key: "s", Value: "a"}, {Key: "b", Value: true}, {Key: "nil", Value: nil}}

	edited := bson.D{{Key: FieldID, Value: bson.NewObjectID()}, {Key: "n", Value: int32(7)}, {Key: "s", Value: "a"}, {Key: "b", Value: true}, {Key: "nil", Value: nil}}
	edited[0].Value = current[0].Value

	if _, changed, err := BuildUpdateDiff(current, edited); err != nil {
		t.Fatalf("unexpected err: %v", err)
	} else if changed {
		t.Fatal("same typed values should not trigger a write")
	}
}

func TestBuildUpdateDiffRejectsOperatorKeys(t *testing.T) {
	for _, key := range []string{"$where", "$currentDate", "$set"} {
		current := bson.D{{Key: FieldID, Value: bson.NewObjectID()}}
		edited := bson.D{{Key: FieldID, Value: bson.NewObjectID()}, {Key: key, Value: "x"}}

		_, _, err := BuildUpdateDiff(current, edited)
		if !errors.Is(err, ErrNoUpdateOperators) {
			t.Fatalf("operator key %q must be rejected, got %v", key, err)
		}
	}

	// 字段名不带 $ 前缀时，值里含 $ 键属于数据，不得被误判为操作符
	current := bson.D{{Key: FieldID, Value: bson.NewObjectID()}}
	edited := bson.D{{Key: FieldID, Value: bson.NewObjectID()}, {Key: "expr", Value: bson.D{{Key: "$gt", Value: int32(1)}}}}
	if _, changed, err := BuildUpdateDiff(current, edited); err != nil || !changed {
		t.Fatalf("subdocument data must be accepted, got changed=%v err=%v", changed, err)
	}
}

// TestBuildUpdateDiffUnsetOnly 只删字段时不应产生 $set 段。
func TestBuildUpdateDiffUnsetOnly(t *testing.T) {
	current := bson.D{{Key: FieldID, Value: bson.NewObjectID()}, {Key: "a", Value: int32(1)}, {Key: "b", Value: int32(2)}}
	edited := bson.D{{Key: FieldID, Value: current[0].Value}, {Key: "a", Value: int32(1)}}

	update, changed, err := BuildUpdateDiff(current, edited)
	if err != nil || !changed {
		t.Fatalf("unexpected changed=%v err=%v", changed, err)
	}
	if firstSection(update, "$set") != nil {
		t.Fatalf("unset-only diff must not contain $set: %+v", update)
	}
	if unset := firstSection(update, "$unset"); len(unset) != 1 || unset[0].Key != "b" {
		t.Fatalf("$unset = %+v, want [b]", unset)
	}
}

func firstSection(update bson.D, key string) bson.D {
	for _, e := range update {
		if e.Key == key {
			if section, ok := e.Value.(bson.D); ok {
				return section
			}
		}
	}
	return nil
}

func hasField(doc bson.D, key string) bool {
	_, ok := Lookup(doc, key)
	return ok
}

func fieldKeys(doc bson.D) []string {
	keys := make([]string, 0, len(doc))
	for _, e := range doc {
		keys = append(keys, e.Key)
	}
	return keys
}
