package mongodoc

import (
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ErrNoUpdateOperators 更新文档里出现了调用方自带的操作符。
//
// 更新语义（$set / $unset 的构造）由服务端负责，调用方只提交「期望的最终文档」。
// 否则调用方可以传 `{"$where": ...}`、`{"$currentDate": ...}` 之类本不应出现在这条通道里的
// 操作符，把「保存一份编辑后的文档」变成「执行一段服务端指令」。
var ErrNoUpdateOperators = errors.New("mongodoc: update document must not contain operators")

// BuildUpdateDiff 依据「服务端当前存储的文档」与「调用方提交的编辑后文档」生成更新指令。
//
// 为什么要在服务端做差异而不是直接 $set 整份文档：
//   - 旧实现只有 $set，从 JSON 里删掉一个 key 不会删除字段，用户看到的删除动作静默无效；
//   - 整份 $set 会把未改动的字段也重写一遍，与并发写互相覆盖的面积更大。
//
// 返回的第二个值表示「是否真的需要写入」：内容完全一致时为 false，调用方据此避免空写。
//
// 顶层 _id 一律从编辑后文档中剔除，主键不可变由服务端保证而不是依赖调用方自觉。
func BuildUpdateDiff(current, edited bson.D) (bson.D, bool, error) {
	target := StripID(edited)
	base := StripID(current)

	if err := assertNoOperators(target); err != nil {
		return nil, false, err
	}

	set := bson.D{}
	for _, e := range target {
		cur, exists := Lookup(base, e.Key)
		if exists && sameValue(cur, e.Value) {
			continue
		}
		set = append(set, bson.E{Key: e.Key, Value: e.Value})
	}

	unset := bson.D{}
	for _, e := range base {
		if _, exists := Lookup(target, e.Key); exists {
			continue
		}
		unset = append(unset, bson.E{Key: e.Key, Value: ""})
	}

	if len(set) == 0 && len(unset) == 0 {
		return nil, false, nil
	}

	update := bson.D{}
	if len(set) > 0 {
		update = append(update, bson.E{Key: "$set", Value: set})
	}
	if len(unset) > 0 {
		update = append(update, bson.E{Key: "$unset", Value: unset})
	}
	return update, true, nil
}

// assertNoOperators 拒绝顶层字段名以 $ 开头。只查顶层：更新指令的操作符只会被解释在顶层，
// 文档值里以 $ 开头的键是数据（Mongo 自身允许 5.0+ 的这类字段名）。
func assertNoOperators(doc bson.D) error {
	for _, e := range doc {
		if len(e.Key) > 0 && e.Key[0] == '$' {
			return ErrNoUpdateOperators
		}
	}
	return nil
}

// sameValue 以 BSON 规范字节比较字段值。
//
// 不能用 == 或 reflect.DeepEqual：int32(1) 与 int64(1) 数值相等但类型不同，
// 而「类型」正是本次重构要守住的那一层。按字节比较即「值与类型都相同才算没变」。
func sameValue(a, b any) bool {
	aBytes, err := bson.Marshal(bson.D{{Key: "v", Value: a}})
	if err != nil {
		return false
	}
	bBytes, err := bson.Marshal(bson.D{{Key: "v", Value: b}})
	if err != nil {
		return false
	}
	return string(aBytes) == string(bBytes)
}
