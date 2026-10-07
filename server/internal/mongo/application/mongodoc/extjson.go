// Package mongodoc 是 Mongo 文档面的 BSON 类型保真边界：
// 请求侧把 JSON 文本解码为「保留字段顺序」的 BSON 文档，响应侧按「普通 JSON 能否无损表达该文档」
// 决定编码模式，并为每个文档签发承载 _id 精确类型的主键令牌。
//
// 包内不持有连接、不做任何 IO，因此全部逻辑可就地单测。
//
// 之所以要有这个包：Mongo 的 BSON 类型集比 JSON 大（Date/ObjectID/Decimal128/Binary/Timestamp/
// Regex 等）。直接把解码结果交给 encoding/json 再让前端原样回传，会让这些类型在「读出来看一眼、
// 点一下保存」之间被静默换成 String 或嵌入文档，属于不可逆的数据损坏。
package mongodoc

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// 文档编码模式。Mode 只决定「怎么展示」，不影响写入语义：写回一律走 Decode，
// 因此两种模式共用同一条解码路径，不会因为模式分叉产生新的出错面。
const (
	// ModePlain 普通 JSON：文档所有字段都能用 JSON 无损表达，展示最清爽。
	ModePlain = "plain"
	// ModeExtJSON MongoDB Extended JSON（relaxed）：含 Date/ObjectID/Binary 等类型时用 $ 包装保真。
	ModeExtJSON = "extjson"
)

// FieldID 文档主键字段名。仅「顶层」主键享有展示豁免（其保真由 QueryDoc.IDToken 承载）。
const FieldID = "_id"

var (
	// ErrDecodeDocument JSON 文本无法解析为 BSON 文档（filter/sort/doc/command 等入参共用）。
	ErrDecodeDocument = errors.New("mongodoc: content is not a valid json document")
	// ErrNotPlainEncodable 值无法用普通 JSON 无损表达。
	//
	// 正常流程不会触发：NeedsExtJSON 与编码器共用 jsonSafeValue 这同一判据。
	// 一旦触发说明两者出现分叉，此时必须报错而不是静默降级——降级即重新引入类型损坏。
	ErrNotPlainEncodable = errors.New("mongodoc: value is not representable as plain json")
)

// EncodedDoc 文档的编码结果。前端展示时按 Mode 决定是否解析 $ 类型包装。
type EncodedDoc struct {
	Mode string          `json:"mode"`
	Doc  json.RawMessage `json:"doc"`
}

// QueryDoc 查询结果里的单个文档。
//
// IDToken 由服务端签发、写回时原样带回：普通 JSON 无法区分「24 位十六进制的字符串 _id」与
// 「ObjectID _id」，按十六进制形状猜测会静默把更新/删除打到错误的文档上（或打空）。
// 投影排除了 _id 的文档 IDToken 为空串：允许查看，但写路径不会受理它的定位请求。
//
// Hash 是存储内容的 BSON 指纹，更新时作为 BaseHash 回传即可实现乐观锁；
// 它不要求额外查库，因为指纹就是在读回的这份文档上算的。
type QueryDoc struct {
	IDToken string `json:"idToken"`
	// IDKind 主键类型名（objectId/string/number/...），用于把形状相同的主键区分开
	IDKind string          `json:"idKind"`
	Hash   string          `json:"hash"`
	Mode   string          `json:"mode"`
	Doc    json.RawMessage `json:"doc"`
}

// Decode 把请求侧 JSON 文本解码为有序 BSON 文档。
//
// 用 UnmarshalExtJSON 而非 encoding/json 有两个必要：
//  1. 它把 JSON 对象解码为 bson.D（切片），字段顺序与请求文本一致，driver 编码时不再随机化，
//     复合排序键与复合索引键因此才确定；
//  2. 普通 JSON 是 Extended JSON 的真子集，所以 plain 与 extjson 两种展示模式的文档都能用同一个
//     入口解回，写回路径无需按模式分叉。
//
// raw 为空或为 null 视为空文档（filter/projection 的「不限制」语义）。
func Decode(raw json.RawMessage) (bson.D, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return bson.D{}, nil
	}

	var doc bson.D
	if err := bson.UnmarshalExtJSON(trimmed, false, &doc); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDecodeDocument, err.Error())
	}
	return doc, nil
}

// DecodeValue 把请求侧 JSON 文本解码为任意 BSON 值（用于非文档位置的入参）。
func DecodeValue(raw json.RawMessage) (any, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}

	var val any
	if err := bson.UnmarshalExtJSON(trimmed, false, &val); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDecodeDocument, err.Error())
	}
	return val, nil
}

// DecodeDocuments 解码「一个文档或文档数组」为文档列表。
//
// 插入入口同时支持单文档与批量：UI 里新增一条不应包成数组，两种写法都是合法的 JSON 意图，
// 在服务端收口成一个函数而不是让调用方各自判断。
func DecodeDocuments(raw json.RawMessage) ([]bson.D, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil
	}

	if trimmed[0] == '[' {
		var docs []bson.D
		if err := bson.UnmarshalExtJSON(trimmed, false, &docs); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrDecodeDocument, err.Error())
		}
		return docs, nil
	}

	doc, err := Decode(trimmed)
	if err != nil {
		return nil, err
	}
	return []bson.D{doc}, nil
}

// Encode 按「最小必要保真」原则编码一个文档：
// 全部字段都能用普通 JSON 表达时输出 plain（字段顺序保持），否则输出 Extended JSON。
func Encode(doc bson.D) (*EncodedDoc, error) {
	if NeedsExtJSON(doc) {
		// canonical=true：类型必须完整包装。实测 relaxed 下 int64 会输出成裸数字
		// （如 9007199254740993），前端 json-bigint(storeAsString) 拿到即成字符串，写回从 int64 损坏为 String；
		// 宁可多一层 $numberLong/$numberDouble，也不能让保真在「驱动输出」这一步静默失效
		b, err := bson.MarshalExtJSON(doc, true, false)
		if err != nil {
			return nil, err
		}
		return &EncodedDoc{Mode: ModeExtJSON, Doc: b}, nil
	}

	b, err := encodePlain(doc)
	if err != nil {
		return nil, err
	}
	return &EncodedDoc{Mode: ModePlain, Doc: b}, nil
}

// EncodeQueryDoc 组装查询结果文档：编码 + 签发主键令牌 + 内容指纹。
func EncodeQueryDoc(doc bson.D) (*QueryDoc, error) {
	encoded, err := Encode(doc)
	if err != nil {
		return nil, err
	}

	token, err := SignID(doc)
	if err != nil {
		// 投影把 _id 排除了是合法查询，不应该整次查询失败；
		// 此时给出空令牌，由写路径按「无主键不可定位」报错，而不是降级成按 _id:null 匹配
		if !errors.Is(err, ErrIDTokenMissing) {
			return nil, err
		}
		token = ""
	}

	hash, err := DocHash(doc)
	if err != nil {
		return nil, err
	}
	idValue, _ := Lookup(doc, FieldID)
	return &QueryDoc{IDToken: token, IDKind: IDKind(idValue), Hash: hash, Mode: encoded.Mode, Doc: encoded.Doc}, nil
}

// EncodeQueryDocs 批量编码查询结果。
func EncodeQueryDocs(docs []bson.D) ([]*QueryDoc, error) {
	res := make([]*QueryDoc, 0, len(docs))
	for _, doc := range docs {
		qdoc, err := EncodeQueryDoc(doc)
		if err != nil {
			return nil, err
		}
		res = append(res, qdoc)
	}
	return res, nil
}

// NeedsExtJSON 报告文档是否需要 Extended JSON 才能无损表达。
//
// 顶层 _id 是唯一豁免：它的类型保真交给 IDToken，展示层按十六进制字符串渲染即可。
// 嵌套位置的 _id 不豁免（那里它只是个普通字段名，没有令牌承载）。
func NeedsExtJSON(doc bson.D) bool {
	for _, e := range doc {
		if e.Key == FieldID {
			if _, ok := e.Value.(bson.ObjectID); ok {
				continue
			}
		}
		if !jsonSafeValue(e.Value) {
			return true
		}
	}
	return false
}

// jsonSafeValue 报告 v 能否用普通 JSON 无损表达。
//
// 数值只放行 int32 与「带小数部分的双精度浮点」：relaxed Extended JSON 里无小数点的数字词元
// 会被解回 int32/int64，所以 int64 与整值 double（1.0、1e10、1e20）一旦走 plain，写回就发生了
// 类型漂移。宁可让这些字段多一层 $numberLong/$numberDouble 包装，也不能静默改掉它们的类型。
//
// fail-closed：default 分支覆盖 ObjectID/DateTime/Decimal128/Binary/Timestamp/Regex/Code/
// CodeWithScope/DBPointer/Undefined/MinKey/MaxKey/Symbol/Null/Raw/int64 以及任何 driver 未来新增的
// 类型，一律判为「需要 ExtJSON」。新增类型无需改这里就能继续保真，这是判据只维护一处的意义。
func jsonSafeValue(v any) bool {
	switch val := v.(type) {
	case nil, string, bool, int32:
		return true
	case float64:
		return jsonSafeFloat(val)
	case bson.D:
		for _, e := range val {
			if !jsonSafeValue(e.Value) {
				return false
			}
		}
		return true
	case bson.M:
		for _, item := range val {
			if !jsonSafeValue(item) {
				return false
			}
		}
		return true
	case bson.A:
		for _, item := range val {
			if !jsonSafeValue(item) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// jsonSafeFloat 仅「有限且带小数部分」的双精度浮点可以用普通 JSON 无损往返。
//
// NaN/Inf 无法由 encoding/json 表达（会直接报 UnsupportedValueError，旧实现因此一个 NaN 就让整次
// 查询失败）；整值浮点则受词法影响被解回整型，两者都必须走 $numberDouble 字符串表达。
func jsonSafeFloat(f float64) bool {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return false
	}
	return f != math.Trunc(f)
}

// encodePlain 输出普通 JSON，并保持文档原有字段顺序（展示稳定与 diff 可比依赖该顺序）。
func encodePlain(doc bson.D) ([]byte, error) {
	return encodeDocument(doc, true)
}

// encodeDocument 编码对象。top 标识当前是否处于文档顶层，用于放行顶层 _id 的 ObjectID。
func encodeDocument(doc bson.D, top bool) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 128))
	buf.WriteByte('{')
	for i, e := range doc {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(e.Key)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')

		val, err := encodeValue(e.Value, top && e.Key == FieldID)
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// encodeValue 编码任意值。allowObjectID 仅在主键展示位置为 true：
// ObjectID 输出为十六进制字符串，其余位置出现 ObjectID 说明 plain 判定已被绕过，直接报错。
func encodeValue(v any, allowObjectID bool) ([]byte, error) {
	switch val := v.(type) {
	case bson.D:
		return encodeDocument(val, false)
	case bson.A:
		return encodeArray(val)
	case bson.M:
		// driver 解码结果是 bson.D，bson.M 仅出现在调用方手工构造的场景。
		// 按 key 排序保证输出确定，否则同一文档两次渲染的字段序可能不同。
		keys := make([]string, 0, len(val))
		for key := range val {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		docs := make(bson.D, 0, len(keys))
		for _, key := range keys {
			docs = append(docs, bson.E{Key: key, Value: val[key]})
		}
		return encodeDocument(docs, false)
	default:
		_, isID := v.(bson.ObjectID)
		if isID {
			if !allowObjectID {
				return nil, fmt.Errorf("%w: %T", ErrNotPlainEncodable, v)
			}
		} else if !jsonSafeValue(v) {
			return nil, fmt.Errorf("%w: %T", ErrNotPlainEncodable, v)
		}
		return json.Marshal(v)
	}
}

func encodeArray(arr bson.A) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 64))
	buf.WriteByte('[')
	for i, item := range arr {
		if i > 0 {
			buf.WriteByte(',')
		}
		b, err := encodeValue(item, false)
		if err != nil {
			return nil, err
		}
		buf.Write(b)
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

// Doc 把可能为 nil 的文档归一为空文档。
//
// driver 不能把 nil bson.D 当文档编码（报 "cannot marshal type bson.D ... WriteNull can only write
// while positioned on a Element or Value but is positioned on a TopLevel"，已实测），
// 而「没有条件」在调用侧天然就是 nil。归一放在进驱动前的最后一站，调用方不必记住这个区别。
func Doc(doc bson.D) bson.D {
	if doc == nil {
		return bson.D{}
	}
	return doc
}

// StripID 返回去掉顶层 _id 的文档副本，用于「以 _id 定位 + 更新其余字段」的写路径，
// 从而在服务端（而非依赖调用方自觉）保证主键不可被修改。
func StripID(doc bson.D) bson.D {
	res := make(bson.D, 0, len(doc))
	for _, e := range doc {
		if e.Key == FieldID {
			continue
		}
		res = append(res, e)
	}
	return res
}

// DocHash 计算文档内容指纹，用于更新时的乐观锁比对。
//
// 指纹取自 BSON 规范字节而不是 JSON 文本：BSON 保留字段序与类型，而 JSON 文本会因 map 随机化
// 或数值词法差异出现「内容未变但指纹变了」的误报。
func DocHash(doc bson.D) (string, error) {
	b, err := bson.Marshal(doc)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
