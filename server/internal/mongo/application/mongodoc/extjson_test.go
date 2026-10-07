package mongodoc

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestNeedsExtJSON(t *testing.T) {
	oid := bson.NewObjectID()

	cases := []struct {
		name string
		doc  bson.D
		want bool
	}{
		{"纯标量文档走 plain", bson.D{{Key: "a", Value: "x"}, {Key: "b", Value: true}, {Key: "c", Value: int32(1)}, {Key: "d", Value: 1.5}, {Key: "e", Value: nil}}, false},
		{"顶层 _id 为 ObjectID 不抬级", bson.D{{Key: FieldID, Value: oid}, {Key: "a", Value: "x"}}, false},
		{"嵌套位置的 ObjectID 必须保真", bson.D{{Key: "userId", Value: oid}}, true},
		{"DateTime 必须保真", bson.D{{Key: "createdAt", Value: bson.NewDateTimeFromTime(time.Now())}}, true},
		{"Decimal128 必须保真", bson.D{{Key: "amount", Value: dec128(t, "129.00")}}, true},
		{"Binary 必须保真", bson.D{{Key: "blob", Value: bson.Binary{Subtype: 0, Data: []byte("x")}}}, true},
		{"UUID(Binary subtype 4) 必须保真", bson.D{{Key: "uuid", Value: bson.Binary{Subtype: 4, Data: []byte("1234567890123456")}}}, true},
		{"Timestamp 必须保真", bson.D{{Key: "ts", Value: bson.Timestamp{T: 1, I: 2}}}, true},
		{"Regex 必须保真", bson.D{{Key: "re", Value: bson.Regex{Pattern: "^a", Options: "i"}}}, true},
		{"JavaScript 必须保真", bson.D{{Key: "code", Value: bson.JavaScript("function(){}")}}, true},
		{"MinKey/MaxKey 必须保真", bson.D{{Key: "k", Value: bson.MinKey{}}}, true},
		{"Undefined 必须保真", bson.D{{Key: "u", Value: bson.Undefined{}}}, true},
		{"未登记类型 fail-closed", bson.D{{Key: "x", Value: bson.Raw(nil)}}, true},
		{"int64 一律保真：relaxed 里无小数点的数字词元会解回 int32", bson.D{{Key: "n", Value: int64(1 << 20)}}, true},
		{"超出安全整数的 int64 必须保真", bson.D{{Key: "n", Value: int64(1 << 53)}}, true},
		{"负向 int64 必须保真", bson.D{{Key: "n", Value: int64(-(1 << 53) - 1)}}, true},
		{"NaN 必须保真", bson.D{{Key: "f", Value: math.NaN()}}, true},
		{"Inf 必须保真", bson.D{{Key: "f", Value: math.Inf(1)}}, true},
		{"整值浮点必须保真（1e10 会被解回整型）", bson.D{{Key: "f", Value: 1e10}}, true},
		{"大整值浮点必须保真", bson.D{{Key: "f", Value: 1e20}}, true},
		{"带小数部分的浮点走 plain", bson.D{{Key: "f", Value: 1.5}}, false},
		{"数组内嵌套特殊类型必须保真", bson.D{{Key: "list", Value: bson.A{int32(1), "x", bson.A{dec128(t, "0.1")}}}}, true},
		{"子文档内特殊类型必须保真", bson.D{{Key: "sub", Value: bson.D{{Key: "at", Value: bson.NewDateTimeFromTime(time.Now())}}}}, true},
		{"纯标量与 int32 数组走 plain", bson.D{{Key: "list", Value: bson.A{int32(1), "x", true, nil}}, {Key: "f", Value: 0.5}}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NeedsExtJSON(c.doc); got != c.want {
				t.Fatalf("NeedsExtJSON(%v) = %v, want %v", c.doc, got, c.want)
			}
		})
	}
}

// TestEncodeModeSelection 校验 Encode 的 mode 与 NeedsExtJSON 判定一致，且输出确为合法 JSON。
func TestEncodeModeSelection(t *testing.T) {
	plainDoc := bson.D{{Key: FieldID, Value: bson.NewObjectID()}, {Key: "name", Value: "mayfly"}}
	extDoc := bson.D{{Key: "createdAt", Value: bson.NewDateTimeFromTime(time.Unix(1700000000, 0))}}

	encoded, err := Encode(plainDoc)
	if err != nil {
		t.Fatalf("encode plain doc: %v", err)
	}
	if encoded.Mode != ModePlain {
		t.Fatalf("mode = %s, want %s", encoded.Mode, ModePlain)
	}
	// 顶层 _id 以十六进制字符串展示，不出现 $oid 包装
	if strings.Contains(string(encoded.Doc), "$oid") {
		t.Fatalf("plain mode should not contain type wrapper, got %s", encoded.Doc)
	}
	if !json.Valid(encoded.Doc) {
		t.Fatalf("plain doc is not valid json: %s", encoded.Doc)
	}

	encoded, err = Encode(extDoc)
	if err != nil {
		t.Fatalf("encode extjson doc: %v", err)
	}
	if encoded.Mode != ModeExtJSON || !strings.Contains(string(encoded.Doc), "$date") {
		t.Fatalf("extjson mode/wrapper mismatch: %s / %s", encoded.Mode, encoded.Doc)
	}
}

// TestPlainPreservesFieldOrder 普通 JSON 必须保持文档字段顺序：
// 展示稳定性与后续 diff 比较都依赖它，encoding/json 对 map 编码会随机化，因此不能用 map。
func TestPlainPreservesFieldOrder(t *testing.T) {
	doc := bson.D{{Key: "z", Value: 1}, {Key: "a", Value: 2}, {Key: "m", Value: 3}, {Key: "_id", Value: bson.NewObjectID()}}

	encoded, err := Encode(doc)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	got := string(encoded.Doc)
	if idxZ, idxA, idxM := strings.Index(got, `"z"`), strings.Index(got, `"a"`), strings.Index(got, `"m"`); !(idxZ < idxA && idxA < idxM) {
		t.Fatalf("field order lost: %s", got)
	}
	if !strings.HasPrefix(got, `{"z"`) {
		t.Fatalf("field order lost: %s", got)
	}
}

// TestEncodeNotPlainEncodable 判定与编码器必须共用同一判据：
// 若绕过判定直接把特殊类型交给 plain 编码器，必须报错而不是静默降级成普通 JSON。
func TestEncodeNotPlainEncodable(t *testing.T) {
	if _, err := encodePlain(bson.D{{Key: "a", Value: bson.NewObjectID()}}); err == nil {
		t.Fatal("expect ErrNotPlainEncodable for ObjectID at non-_id position, got nil")
	} else if !strings.Contains(err.Error(), ErrNotPlainEncodable.Error()) {
		t.Fatalf("unexpected err: %v", err)
	}
	if _, err := encodePlain(bson.D{{Key: "a", Value: bson.DateTime(1)}}); err == nil {
		t.Fatal("expect ErrNotPlainEncodable for DateTime, got nil")
	}
}

// TestRoundTripKeepsBSONTypes 是本次重构的核心回归：
// 「读出来 → 原样写回」不得改变任何字段的 BSON 类型，含 NaN/Inf 这类无法用 JSON 词法表达的值。
func TestRoundTripKeepsBSONTypes(t *testing.T) {
	oid := bson.NewObjectID()
	now := time.Unix(1700000000, 0).UTC()

	doc := bson.D{
		{Key: FieldID, Value: oid},
		{Key: "str", Value: "hello"},
		{Key: "bool", Value: true},
		{Key: "i32", Value: int32(-7)},
		{Key: "i64", Value: int64(1 << 53)},
		{Key: "dbl", Value: 1.5},
		{Key: "nan", Value: math.NaN()},
		{Key: "inf", Value: math.Inf(-1)},
		{Key: "big", Value: 1e20},
		{Key: "date", Value: bson.NewDateTimeFromTime(now)},
		{Key: "dec", Value: dec128(t, "129.005")},
		{Key: "bin", Value: bson.Binary{Subtype: 0, Data: []byte("\x00\x01\xff")}},
		{Key: "uuid", Value: bson.Binary{Subtype: 4, Data: []byte("0011223344556677")}},
		{Key: "ts", Value: bson.Timestamp{T: 1700000000, I: 3}},
		{Key: "re", Value: bson.Regex{Pattern: "^a[0-9]+$", Options: "im"}},
		{Key: "min", Value: bson.MinKey{}},
		{Key: "max", Value: bson.MaxKey{}},
		{Key: "undef", Value: bson.Undefined{}},
		{Key: "null", Value: nil},
		{Key: "arr", Value: bson.A{int32(1), "x", bson.D{{Key: "k", Value: dec128(t, "0.5")}}}},
		{Key: "sub", Value: bson.D{{Key: "ref", Value: oid}, {Key: "at", Value: bson.NewDateTimeFromTime(now)}}},
	}

	encoded, err := Encode(doc)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if encoded.Mode != ModeExtJSON {
		t.Fatalf("expect extjson mode for rich document, got %s", encoded.Mode)
	}

	decoded, err := Decode(encoded.Doc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertSameBSON(t, doc, decoded)

	// plain 文档同样必须无损（_id 除外：它展示为十六进制字符串，保真由 IDToken 承载）
	plain := bson.D{{Key: FieldID, Value: oid}, {Key: "str", Value: "x"}, {Key: "i32", Value: int32(1)}, {Key: "dbl", Value: 0.25}, {Key: "arr", Value: bson.A{true, nil}}}
	plainEncoded, err := Encode(plain)
	if err != nil {
		t.Fatalf("encode plain: %v", err)
	}
	if plainEncoded.Mode != ModePlain {
		t.Fatalf("expect plain mode, got %s", plainEncoded.Mode)
	}
	plainDecoded, err := Decode(plainEncoded.Doc)
	if err != nil {
		t.Fatalf("decode plain: %v", err)
	}
	assertSameBSON(t, StripID(plain), StripID(plainDecoded))
	// 顶层 _id 以十六进制展示，便于阅读与复制
	if got := plainDecoded[0].Value; got != oid.Hex() {
		t.Fatalf("_id displayed as %v, want hex %s", got, oid.Hex())
	}
}

// TestDecodeKeepsFieldOrder 请求侧解码必须保留字段顺序：
// 复合排序键与复合索引键的顺序是语义的一部分，此前经 map 传递会被 Go 的随机迭代序打乱。
func TestDecodeKeepsFieldOrder(t *testing.T) {
	doc, err := Decode(json.RawMessage(`{"b":1,"a":-1,"z":1}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	var keys []string
	for _, e := range doc {
		keys = append(keys, e.Key)
	}
	if want := []string{"b", "a", "z"}; !reflect.DeepEqual(keys, want) {
		t.Fatalf("field order = %v, want %v", keys, want)
	}

	// 反复解码同一文本，顺序必须稳定（回归旧实现 map 迭代序随机的缺陷）
	for i := 0; i < 50; i++ {
		again, err := Decode(json.RawMessage(`{"b":1,"a":-1,"z":1}`))
		if err != nil {
			t.Fatalf("decode again: %v", err)
		}
		if again[0].Key != "b" || again[1].Key != "a" || again[2].Key != "z" {
			t.Fatalf("field order unstable at round %d: %v", i, again)
		}
	}
}

// TestDecodeExtJSONTypes 请求侧的 $ 类型包装必须还原为精确 BSON 类型（写回路径的唯一入口）。
func TestDecodeExtJSONTypes(t *testing.T) {
	doc, err := Decode(json.RawMessage(`{"_id":{"$oid":"507f1f77bcf86cd799439011"},"at":{"$date":"2026-01-02T03:04:05.000Z"},"n":{"$numberLong":"9007199254740993"},"d":{"$numberDecimal":"1.5"}}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	expectTypes := map[string]any{
		"_id": bson.ObjectID{},
		"at":  bson.DateTime(0),
		"n":   int64(0),
		"d":   bson.Decimal128{},
	}
	for _, e := range doc {
		want, ok := expectTypes[e.Key]
		if !ok {
			t.Fatalf("unexpected field %q", e.Key)
		}
		if gotType, wantType := reflect.TypeOf(e.Value), reflect.TypeOf(want); gotType != wantType {
			t.Fatalf("field %q decoded as %T, want %T", e.Key, e.Value, want)
		}
	}
}

func TestDecodeEmptyAndInvalid(t *testing.T) {
	if doc, err := Decode(nil); err != nil || len(doc) != 0 {
		t.Fatalf("empty input should decode to empty doc, got %v / %v", doc, err)
	}
	if doc, err := Decode(json.RawMessage("null")); err != nil || len(doc) != 0 {
		t.Fatalf("null should decode to empty doc, got %v / %v", doc, err)
	}
	if _, err := Decode(json.RawMessage(`{"a":`)); err == nil {
		t.Fatal("expect error for malformed json")
	}
	// 非文档 JSON（数字）不能当 filter 用，必须报错而不是解出一个空文档
	if _, err := Decode(json.RawMessage(`5`)); err == nil {
		t.Fatal("expect error for non-document json")
	}
}

// TestEncodeQueryDocWithoutID 投影排除 _id 时仍然返回文档但不签发令牌：
// 查看是合法需求，但写回必须被拦下——否则「无主键」会被降级成「按 _id:null 定位」而误伤另一条真实文档。
func TestEncodeQueryDocWithoutID(t *testing.T) {
	qdoc, err := EncodeQueryDoc(bson.D{{Key: "name", Value: "x"}})
	if err != nil {
		t.Fatalf("querying a projection without _id must succeed: %v", err)
	}
	if qdoc.IDToken != "" {
		t.Fatalf("no id token expected when _id absent, got %q", qdoc.IDToken)
	}
	if _, err = ParseID(qdoc.IDToken); err == nil {
		t.Fatal("empty token must be rejected by the write path")
	}

	// _id 显式为 null 是合法主键，必须能签发与还原
	withNull, err := EncodeQueryDoc(bson.D{{Key: FieldID, Value: nil}, {Key: "name", Value: "x"}})
	if err != nil {
		t.Fatalf("sign null id: %v", err)
	}
	if withNull.IDToken == "" {
		t.Fatal("null _id should still get a token")
	}
	val, err := ParseID(withNull.IDToken)
	if err != nil {
		t.Fatalf("parse null id: %v", err)
	}
	if val != nil {
		t.Fatalf("parsed null id = %v, want nil", val)
	}
}

// TestQueryDocHashDetectsChange 内容指纹要能区分「真的改了」与「什么都没改」，
// 并且同一内容反复计算结果一致，否则乐观锁会漏报或误报。
func TestQueryDocHashDetectsChange(t *testing.T) {
	id := bson.NewObjectID()
	doc := bson.D{{Key: FieldID, Value: id}, {Key: "n", Value: int32(1)}}

	first, err := EncodeQueryDoc(doc)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	again, err := EncodeQueryDoc(doc)
	if err != nil {
		t.Fatalf("encode again: %v", err)
	}
	if first.Hash == "" || first.Hash != again.Hash {
		t.Fatalf("hash must be stable for identical content: %q vs %q", first.Hash, again.Hash)
	}

	changed, err := EncodeQueryDoc(bson.D{{Key: FieldID, Value: id}, {Key: "n", Value: int64(1)}})
	if err != nil {
		t.Fatalf("encode changed: %v", err)
	}
	if changed.Hash == first.Hash {
		t.Fatal("int32 -> int64 with the same value must change the hash")
	}
}

func TestStripID(t *testing.T) {
	stripped := StripID(bson.D{{Key: FieldID, Value: bson.NewObjectID()}, {Key: "a", Value: 1}, {Key: "b", Value: 2}})
	if len(stripped) != 2 || stripped[0].Key != "a" {
		t.Fatalf("strip _id failed: %v", stripped)
	}
}

// TestIDKindDistinguishesLookalikeIDs 形状相同的主键必须靠类型区分开：
// ObjectID 与「24 位十六进制的字符串」在展示上完全一致，不标类型就无法判断正在动哪一条。
func TestIDKindDistinguishesLookalikeIDs(t *testing.T) {
	objectID, err := bson.ObjectIDFromHex("507f1f77bcf86cd799439011")
	if err != nil {
		t.Fatalf("build oid: %v", err)
	}

	cases := []struct {
		id   any
		want string
	}{
		{objectID, "objectId"},
		{"507f1f77bcf86cd799439011", "string"},
		{int32(7), "number"},
		{int64(7), "number"},
		{true, "boolean"},
		{nil, "null"},
		{bson.DateTime(1), "date"},
		{bson.Binary{Subtype: 4}, "binary"},
		{bson.D{{Key: "a", Value: int32(1)}}, "document"},
		{bson.A{int32(1)}, "array"},
		{bson.Regex{Pattern: "a"}, "regex"},
		{bson.JavaScript("x"), "other"}, // 未归类的具体类型走 other，不致于显示空标签
	}
	for _, c := range cases {
		if got := IDKind(c.id); got != c.want {
			t.Fatalf("IDKind(%T %v) = %s, want %s", c.id, c.id, got, c.want)
		}
	}
}

// TestQueryDocCarriesIDKind 编码结果必须带出主键类型，否则前端无从区分同形主键。
func TestQueryDocCarriesIDKind(t *testing.T) {
	const hexLike = "507f1f77bcf86cd799439011"
	objectID, err := bson.ObjectIDFromHex(hexLike)
	if err != nil {
		t.Fatalf("build oid: %v", err)
	}

	objectIDDoc, err := EncodeQueryDoc(bson.D{{Key: FieldID, Value: objectID}, {Key: "a", Value: "x"}})
	if err != nil {
		t.Fatalf("encode objectid doc: %v", err)
	}
	if objectIDDoc.IDKind != "objectId" {
		t.Fatalf("objectId doc kind = %s", objectIDDoc.IDKind)
	}

	stringDoc, err := EncodeQueryDoc(bson.D{{Key: FieldID, Value: hexLike}, {Key: "a", Value: "x"}})
	if err != nil {
		t.Fatalf("encode string doc: %v", err)
	}
	if stringDoc.IDKind != "string" {
		t.Fatalf("string doc kind = %s", stringDoc.IDKind)
	}
	// 主键顶层享有展示豁免，两种类型的编码文本完全一致：
	// 这正是 idKind 必须随文档下发的理由（集成测试里两条文档看起来一模一样）
	if string(objectIDDoc.Doc) != string(stringDoc.Doc) {
		t.Fatalf("look-alike ids should render the same text, got %s vs %s", objectIDDoc.Doc, stringDoc.Doc)
	}
	if objectIDDoc.IDToken == stringDoc.IDToken {
		t.Fatal("same-looking ids must still carry different tokens, otherwise writes would hit the wrong document")
	}
}

// TestDocNormalizesNilDocument 归一化判据：driver 无法编码 nil 文档，调用侧的「没写条件」就是 nil。
func TestDocNormalizesNilDocument(t *testing.T) {
	if got := Doc(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil doc must normalize to an empty non-nil doc, got %#v", got)
	}
	doc := bson.D{{Key: "a", Value: int32(1)}}
	if got := Doc(doc); !reflect.DeepEqual(got, doc) {
		t.Fatalf("non-nil doc must pass through untouched, got %+v", got)
	}
}

// dec128 Decimal128 构造助手，失败即终止测试。
func dec128(t *testing.T, s string) bson.Decimal128 {
	t.Helper()
	d, err := bson.ParseDecimal128(s)
	if err != nil {
		t.Fatalf("decimal %q: %v", s, err)
	}
	return d
}

// assertSameBSON 以 BSON 规范字节比较两个文档。
//
// 不用 reflect.DeepEqual 的原因是 NaN != NaN，会把「保真成功」误判成失败；
// 序列化后的字节相同即类型与取值都一致（字段顺序已在编码侧保证）。
func assertSameBSON(t *testing.T, want, got bson.D) {
	t.Helper()

	wantBytes, err := bson.Marshal(want)
	if err != nil {
		t.Fatalf("marshal want: %v", err)
	}
	gotBytes, err := bson.Marshal(got)
	if err != nil {
		t.Fatalf("marshal got: %v", err)
	}
	if string(wantBytes) != string(gotBytes) {
		t.Fatalf("bson round-trip lost fidelity\nwant: %+v\n got: %+v", want, got)
	}
}
