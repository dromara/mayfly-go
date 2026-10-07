package mongodoc

import (
	"encoding/base64"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// TestIDTokenKeepsExactTypeID 主键令牌的类型还原必须零猜测。
//
// 这里锁死本次重构的关键回归：旧实现用 ObjectIDFromHex 能否解析成功来判定 _id 类型，
// 于是「24 位十六进制的字符串主键」会被误转成 ObjectID，更新/删除直接打空或打到别的文档上。
func TestIDTokenKeepsExactTypeID(t *testing.T) {
	oid, err := bson.ObjectIDFromHex("507f1f77bcf86cd799439011")
	if err != nil {
		t.Fatalf("build oid: %v", err)
	}

	cases := []struct {
		name string
		id   any
		want any
	}{
		{"ObjectID 主键", oid, oid},
		{"形状像 ObjectID 的字符串主键必须保持字符串", "507f1f77bcf86cd799439011", "507f1f77bcf86cd799439011"},
		{"普通字符串主键", "order-1", "order-1"},
		{"int32 主键", int32(7), int32(7)},
		{"int64 主键（超出安全整数）", int64(9007199254740993), int64(9007199254740993)},
		{"零值主键 0 不被当作缺失", int32(0), int32(0)},
		{"空字符串主键", "", ""},
		{"false 主键", false, false},
		{"double 主键", 1.5, 1.5},
		{"UUID(Binary subtype 4) 主键", bson.Binary{Subtype: 4, Data: []byte("0011223344556677")}, bson.Binary{Subtype: 4, Data: []byte("0011223344556677")}},
		{"复合主键（子文档）", bson.D{{Key: "a", Value: int32(1)}, {Key: "b", Value: "x"}}, bson.D{{Key: "a", Value: int32(1)}, {Key: "b", Value: "x"}}},
		{"Date 主键", bson.NewDateTimeFromTime(time1700000000()), bson.NewDateTimeFromTime(time1700000000())},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			token, err := SignID(bson.D{{Key: FieldID, Value: c.id}})
			if err != nil {
				t.Fatalf("sign: %v", err)
			}
			if token == "" {
				t.Fatal("token should not be empty")
			}

			got, err := ParseID(token)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if gotType, wantType := reflect.TypeOf(got), reflect.TypeOf(c.want); gotType != wantType {
				t.Fatalf("type changed: got %T, want %T", got, c.want)
			}

			// 以 BSON 字节比较，避免 NaN 之类不可比较值干扰，同时也能验出取值差异
			gotBytes, err := bson.Marshal(bson.D{{Key: FieldID, Value: got}})
			if err != nil {
				t.Fatalf("marshal got: %v", err)
			}
			wantBytes, err := bson.Marshal(bson.D{{Key: FieldID, Value: c.want}})
			if err != nil {
				t.Fatalf("marshal want: %v", err)
			}
			if string(gotBytes) != string(wantBytes) {
				t.Fatalf("id value changed: %+v vs %+v", got, c.want)
			}
		})
	}
}

func TestSignIDMissingID(t *testing.T) {
	if _, err := SignID(bson.D{{Key: "name", Value: "x"}}); err != ErrIDTokenMissing {
		t.Fatalf("expect ErrIDTokenMissing, got %v", err)
	}
}

func TestParseIDRejectsBadTokens(t *testing.T) {
	if _, err := ParseID(""); err == nil {
		t.Fatal("empty token must be rejected")
	}
	if _, err := ParseID("!!!not-base64!!!"); err == nil {
		t.Fatal("non base64 token must be rejected")
	}
	if _, err := ParseID(base64.RawURLEncoding.EncodeToString([]byte(`{"a":`))); err == nil {
		t.Fatal("malformed json token must be rejected")
	}
	// relaxed 形态（如 {"$date":"iso8601"}）不是 SignID 的输出形态，
	// canonicalOnly 校验必须拒掉它，防止调用方自备一个形状正确但类型已丢的令牌
	if _, err := ParseID(base64.RawURLEncoding.EncodeToString([]byte(`{"v":{"$date":"2026-01-01T00:00:00.000Z"}}`))); err == nil {
		t.Fatal("relaxed date payload must be rejected")
	}
	if _, err := ParseID(base64.RawURLEncoding.EncodeToString([]byte(`{"v":{"$oid":"507f1f77bcf86cd799439011"},"x":1}`))); err == nil {
		t.Fatal("multi-field payload must be rejected")
	}
}

// TestParseIDRejectsOperatorDocuments 令牌不得成为条件注入面：
// {_id: {"$ne":null}} 会被 Mongo 解释成「匹配任意文档」，等于按一个不可控的条件删改数据。
func TestParseIDRejectsOperatorDocuments(t *testing.T) {
	for _, payload := range []string{
		// 外层形状合法、内层主键值是操作符文档：必须被 assertNoOperatorKeys 拦下
		`{"v":{"$ne":null}}`,
		`{"v":{"$gt":""}}`,
		`{"v":{"nested":{"$where":"sleep(5000)"}}}`,
		`{"v":[{"$in":[1,2]}]}`,
		// 缺 wrapper 或多余字段：不是 SignID 的输出形态
		`{"$ne":null}`,
		`{"v":1,"extra":2}`,
		`{"other":1}`,
	} {
		if _, err := ParseID(base64.RawURLEncoding.EncodeToString([]byte(payload))); err == nil {
			t.Fatalf("payload %q must be rejected", payload)
		}
	}

	// 复合主键里的普通子文档必须放行（它是数据，不是操作符）
	composite := bson.D{{Key: "a", Value: int32(1)}, {Key: "b", Value: "x"}}
	got, err := ParseID(mustSign(t, composite))
	if err != nil {
		t.Fatalf("plain subdocument id must be accepted, got %v", err)
	}
	if _, ok := got.(bson.D); !ok {
		t.Fatalf("composite id decoded as %T, want bson.D", got)
	}
}

func TestIDFilterShape(t *testing.T) {
	oid := bson.NewObjectID()
	value, err := ParseID(mustSign(t, oid))
	if err != nil {
		t.Fatalf("parse id: %v", err)
	}

	cond := IDFilter(value)
	if len(cond) != 1 || cond[0].Key != FieldID {
		t.Fatalf("unexpected cond shape: %+v", cond)
	}
	if got, ok := cond[0].Value.(bson.ObjectID); !ok || got != oid {
		t.Fatalf("cond value = %+v, want oid %v", cond[0].Value, oid)
	}
}

// TestPlainAndExtJSONBothWritable 两种展示模式的文档都能用同一入口解回：
// 写路径不分叉是「按类型启用 ExtJSON」这个设计不产生新出错面的前提。
func TestPlainAndExtJSONBothWritable(t *testing.T) {
	oid := bson.NewObjectID()

	plainDoc := bson.D{{Key: FieldID, Value: oid}, {Key: "name", Value: "x"}, {Key: "n", Value: int32(1)}}
	plainEncoded, err := Encode(plainDoc)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if plainEncoded.Mode != ModePlain {
		t.Fatalf("expect plain, got %s", plainEncoded.Mode)
	}
	if _, err = Decode(plainEncoded.Doc); err != nil {
		t.Fatalf("plain doc must be decodable: %v", err)
	}

	extDoc := bson.D{{Key: FieldID, Value: oid}, {Key: "at", Value: bson.NewDateTimeFromTime(time1700000000())}}
	extEncoded, err := Encode(extDoc)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if extEncoded.Mode != ModeExtJSON {
		t.Fatalf("expect extjson, got %s", extEncoded.Mode)
	}
	decoded, err := Decode(extEncoded.Doc)
	if err != nil {
		t.Fatalf("extjson doc must be decodable: %v", err)
	}
	if _, ok := decoded[1].Value.(bson.DateTime); !ok {
		t.Fatalf("date lost in write path: %T", decoded[1].Value)
	}
}

func mustSign(t *testing.T, id any) string {
	t.Helper()
	token, err := SignID(bson.D{{Key: FieldID, Value: id}})
	if err != nil {
		t.Fatalf("sign id: %v", err)
	}
	return token
}

func time1700000000() time.Time {
	return time.Unix(1700000000, 0).UTC()
}
