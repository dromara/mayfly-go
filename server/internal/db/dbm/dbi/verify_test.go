package dbi

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalValue(t *testing.T) {
	tm := time.Date(2024, 6, 15, 8, 30, 5, 123456789, time.FixedZone("UTC+8", 8*3600))
	tmUtc := tm.UTC().Format(time.RFC3339Nano)

	cases := []struct {
		v    any
		want string
	}{
		{nil, "<nil>"},
		{"hello", "hello"},
		{"", ""},
		{true, "true"},
		{false, "false"},
		// 整型族统一十进制
		{int(42), "42"},
		{int8(-1), "-1"},
		{int16(300), "300"},
		{int32(-70000), "-70000"},
		{int64(9223372036854775807), "9223372036854775807"},
		{uint8(255), "255"},
		{uint64(18446744073709551615), "18446744073709551615"},
		// 浮点最短精确表示
		{float64(1.5), "1.5"},
		{float64(0.1), "0.1"},
		{float32(2.5), "2.5"},
		// 二进制 → 0x前缀hex（区别于普通字符串）
		{[]byte{0x01, 0xab}, "0x01ab"},
		{[]byte{}, "0x"},
		// 时间 → UTC RFC3339Nano（消除时区差异）
		{tm, tmUtc},
		{time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), "2024-01-01T00:00:00Z"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, CanonicalValue(c.v), "输入%#v", c.v)
	}
}

func TestCanonicalValue_TimezoneNormalize(t *testing.T) {
	// 同一时刻不同时区 → 相同规范化值
	t1 := time.Date(2024, 6, 15, 0, 30, 0, 0, time.UTC)
	t2 := t1.In(time.FixedZone("UTC+8", 8*3600))
	assert.Equal(t, CanonicalValue(t1), CanonicalValue(t2))
}

func TestCanonicalValue_UnknownTypeFallback(t *testing.T) {
	assert.Equal(t, "{a b}", CanonicalValue(struct{ A, B string }{"a", "b"}))
}

func TestCanonicalEqual(t *testing.T) {
	// 跨形态等值：int64(42) vs uint8(42)
	assert.True(t, CanonicalEqual(int64(42), uint8(42)))
	assert.True(t, CanonicalEqual(int64(42), int32(42)))
	// 数值与十进制字符串等值：pg numeric等驱动将数值以字符串返回，跨方言比对需视为等值
	assert.True(t, CanonicalEqual(int64(42), "42"))
	assert.True(t, CanonicalEqual(nil, nil))
	// 时间与其Unix时间戳整型不跨形态等值
	assert.False(t, CanonicalEqual(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), int64(1735689600)))
}

// NULL与真实字符串必须严格区分：字符串值恰好等于NULL规范化表示时不得误判相等（否则差异漏报）
func TestCanonicalEqual_NilNotCollideWithString(t *testing.T) {
	assert.False(t, CanonicalEqual(nil, CanonicalNilValue))
	assert.False(t, CanonicalEqual(CanonicalNilValue, nil))
	assert.True(t, CanonicalEqual(CanonicalNilValue, CanonicalNilValue))
	// 二进制与其0xhex文本形态视为等值：兼容以文本返回hex的驱动，属设计内的跨形态等价
	assert.True(t, CanonicalEqual([]byte{0x01}, "0x01"))
}

func TestCanonicalRowKeyAndSignature(t *testing.T) {
	row := map[string]any{"id": int64(1), "name": "n1", "val": []byte{0xff}}
	assert.Equal(t, "1", CanonicalRowKey(row, "id"))
	assert.Equal(t, CanonicalNilValue, CanonicalRowKey(row, "not_exist"))

	sig := CanonicalRowSignature(row, []string{"id", "name"})
	assert.Equal(t, "id\x1fV:1\x1ename\x1fV:n1", sig)
	// 列顺序影响签名（调用方须保证顺序一致）
	assert.NotEqual(t, sig, CanonicalRowSignature(row, []string{"name", "id"}))
}

// 签名不得因值中含分隔符或与NULL规范化表示同形而产生歧义
func TestCanonicalRowSignature_AmbiguityAndNilSemantics(t *testing.T) {
	columns := []string{"a", "b"}
	// NULL vs 字符串"<nil>"：旧形态均为"<nil>"会碰撞，新形态以 N:/V: 前缀区分
	nilRow := map[string]any{"a": nil, "b": "x"}
	strRow := map[string]any{"a": CanonicalNilValue, "b": "x"}
	assert.NotEqual(t, CanonicalRowSignature(nilRow, columns), CanonicalRowSignature(strRow, columns))

	// 值中含 , = 与列名同名的文本：不得拼出与另一行相同的签名
	r1 := map[string]any{"a": "1", "b": "a\x1f1,b\x1f2"}
	r2 := map[string]any{"a": "1,b=a\x1f1", "b": "2"}
	assert.NotEqual(t, CanonicalRowSignature(r1, columns), CanonicalRowSignature(r2, columns))

	// 完全相同的行签名一致（含多行、JSON、emoji等复杂值）
	complex := map[string]any{"a": "{\"k\":\"v1\n\"}", "b": "中文😀'';"}
	assert.Equal(t, CanonicalRowSignature(complex, columns), CanonicalRowSignature(map[string]any{"a": "{\"k\":\"v1\n\"}", "b": "中文😀'';"}, columns))
}

// 复杂字符串规范化：驱动以string返回时原样保留（换行/引号/反斜杠/emoji/JSON均不得被改写），
// 以[]byte返回时统一为0xhex形态
func TestCanonicalValue_ComplexStrings(t *testing.T) {
	kases := []struct {
		name string
		v    any
		want string
	}{
		{"含单引号", "it's", "it's"},
		{"含双引号", `say "hi"`, `say "hi"`},
		{"含反斜杠", `a\b`, `a\b`},
		{"含LF换行", "l1\nl2", "l1\nl2"},
		{"含CRLF", "l1\r\nl2", "l1\r\nl2"},
		{"含制表符", "a\tb", "a\tb"},
		{"JSON文本", `{"k":"v;a"}`, `{"k":"v;a"}`},
		{"中文与emoji", "中文，😀", "中文，😀"},
		{"字节序列与同文本字符串", []byte("abc"), "0x" + hex.EncodeToString([]byte("abc"))},
	}
	for _, k := range kases {
		t.Run(k.name, func(t *testing.T) {
			assert.Equal(t, k.want, CanonicalValue(k.v))
		})
	}
}
