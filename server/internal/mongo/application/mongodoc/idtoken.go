package mongodoc

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	// ErrIDTokenInvalid 主键令牌缺失、被篡改或内容不合法。
	ErrIDTokenInvalid = errors.New("mongodoc: invalid id token")
	// ErrIDTokenMissing 文档没有主键字段（例如投影显式排除了 _id），因此无法签发定位用令牌。
	// 与 ErrIDTokenInvalid 分开：前者要提示「调整投影」，后者要提示「内容不合法」。
	ErrIDTokenMissing = errors.New("mongodoc: document has no _id field")
)

// idTokenKey 令牌内层的固定字段名。
//
// bson.MarshalExtJSON 不接受顶层裸标量（driver 报 "can only write while positioned on a Element or
// Value but is positioned on a TopLevel"），所以主键值必须包在一个单字段文档里再编解码；
// 该字段名同时充当令牌的格式校验位。
const idTokenKey = "v"

// SignID 把文档主键值签发为不透明令牌（canonical Extended JSON 的 base64url 编码）。
//
// 为什么不让前端直接回传 _id 的 JSON 文本：普通 JSON 无法区分「24 位十六进制的字符串 _id」与
// 「ObjectID _id」，也无法区分 int64 与 double。按形状猜测会让更新/删除打到错误的文档上（或打空），
// 而调用方无法察觉。令牌由服务端签发、以 canonical 形态承载类型，还原过程零猜测。
//
// 文档不含 _id 字段时返回 ErrIDTokenMissing，写路径据此拒绝定位，避免把「无主键」降级成
// 「按 _id:null 定位」而误伤另一条真实文档。
func SignID(doc bson.D) (string, error) {
	id, ok := Lookup(doc, FieldID)
	if !ok {
		return "", ErrIDTokenMissing
	}
	return SignIDValue(id)
}

// SignIDValue 直接为一个 _id 值签发令牌，供插入后拿到驱动生成的主键时立即签发使用
// （此时文档尚未回读，不能走 SignID(doc)）。
func SignIDValue(id any) (string, error) {
	// canonical=true：类型必须以 $oid/$date/$numberLong 等包装显式表达，
	// 这样 ParseID 用 canonicalOnly 校验时才能力拒被改写成 relaxed 形态的伪造令牌
	b, err := bson.MarshalExtJSON(bson.D{{Key: idTokenKey, Value: id}}, true, false)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ParseID 还原主键令牌为精确类型的 BSON 值。
func ParseID(token string) (any, error) {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: token is empty", ErrIDTokenInvalid)
	}

	raw, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrIDTokenInvalid, err.Error())
	}

	// canonicalOnly=true：令牌由 SignID 以 canonical 形态签发，收到非 canonical 内容即说明被篡改
	var wrapper bson.D
	if err = bson.UnmarshalExtJSON(raw, true, &wrapper); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrIDTokenInvalid, err.Error())
	}
	if len(wrapper) != 1 || wrapper[0].Key != idTokenKey {
		return nil, fmt.Errorf("%w: unexpected payload shape", ErrIDTokenInvalid)
	}

	val := wrapper[0].Value
	// 主键值会以 {_id: <val>} 形式进入查询条件，若允许 $ 前缀键，
	// {"$ne":null} 这类操作符文档会被 Mongo 解释成「匹配任意文档」，等于按令牌注入条件。
	// 类型包装（$oid/$date 等）在上一步已被还原为 BSON 值，不会以 $ 键残留。
	if err = assertNoOperatorKeys(val); err != nil {
		return nil, err
	}
	return val, nil
}

// assertNoOperatorKeys 递归拒绝任何 $ 前缀键。
func assertNoOperatorKeys(v any) error {
	switch val := v.(type) {
	case bson.D:
		for _, e := range val {
			if strings.HasPrefix(e.Key, "$") {
				return fmt.Errorf("%w: operator-like key %q is not allowed", ErrIDTokenInvalid, e.Key)
			}
			if err := assertNoOperatorKeys(e.Value); err != nil {
				return err
			}
		}
	case bson.M:
		for key, item := range val {
			if strings.HasPrefix(key, "$") {
				return fmt.Errorf("%w: operator-like key %q is not allowed", ErrIDTokenInvalid, key)
			}
			if err := assertNoOperatorKeys(item); err != nil {
				return err
			}
		}
	case bson.A:
		for _, item := range val {
			if err := assertNoOperatorKeys(item); err != nil {
				return err
			}
		}
	}
	return nil
}

// IDFilter 构造按主键精确定位的条件文档。
// 入参一般是 ParseID 还原出的主键值（类型与存储一致），所以这里不再做任何形状推断。
func IDFilter(idValue any) bson.D {
	return bson.D{{Key: FieldID, Value: idValue}}
}

// IDKind 返回主键值的类型名，随文档一起下发。
//
// 为什么需要它：顶层 _id 不参与「是否需要 Extended JSON」判定（它的保真由 IDToken 承载），
// 于是 ObjectID 主键与「24 位十六进制的字符串主键」在展示上长得一模一样。
// 不告知类型，人就无法判断自己正在动的是哪一条（真实集合同一时存在两种主键并不罕见）。
func IDKind(id any) string {
	switch id.(type) {
	case nil:
		// nil 既可能是「_id 不存在」也可能是「主键值就是 null」；前者不会走到这里（SignID 先报错），
		// 所以这里就是后者
		return "null"
	case string:
		return "string"
	case bool:
		return "boolean"
	case int32, int64, float64, uint32, uint64:
		return "number"
	case bson.ObjectID:
		return "objectId"
	case bson.DateTime:
		return "date"
	case bson.Decimal128:
		return "decimal"
	case bson.Binary:
		return "binary"
	case bson.Regex:
		return "regex"
	case bson.Timestamp:
		return "timestamp"
	case bson.D, bson.M:
		return "document"
	case bson.A:
		return "array"
	default:
		return "other"
	}
}

// Lookup 取文档顶层字段值并报告该字段是否存在（用于区分「字段不存在」与「字段值为 null」）。
func Lookup(doc bson.D, key string) (any, bool) {
	for _, e := range doc {
		if e.Key == key {
			return e.Value, true
		}
	}
	return nil, false
}
