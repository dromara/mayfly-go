package transfer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"mayfly-go/internal/db/dbm/dbi"
)

// ---------------- normalizeConcurrency ----------------

func TestNormalizeConcurrency(t *testing.T) {
	// 0/负数 → 默认值
	assert.Equal(t, DefaultTransferConcurrency, normalizeConcurrency(0))
	assert.Equal(t, DefaultTransferConcurrency, normalizeConcurrency(-1))
	assert.Equal(t, DefaultTransferConcurrency, normalizeConcurrency(-100))

	// 正常范围原样保留
	assert.Equal(t, 1, normalizeConcurrency(1))
	assert.Equal(t, 4, normalizeConcurrency(4))
	assert.Equal(t, 16, normalizeConcurrency(16))

	// 越界钳制
	assert.Equal(t, MaxTransferConcurrency, normalizeConcurrency(17))
	assert.Equal(t, MaxTransferConcurrency, normalizeConcurrency(1000))
}

// ---------------- dbValToInt64 ----------------

func TestDbValToInt64(t *testing.T) {
	cases := []struct {
		v    any
		want int64
		ok   bool
	}{
		{nil, 0, false},          // 空表MIN/MAX为NULL
		{int64(42), 42, true},    // mysql/pg驱动常见形态
		{int(42), 42, true},      //
		{int32(42), 42, true},    //
		{uint64(42), 42, true},   //
		{float64(42), 42, true},  // sqlite等驱动
		{[]byte("42"), 42, true}, // 部分驱动[]byte形态
		{"42", 42, true},         // 字符串形态
		{"42.0", 42, true},       // float字符串
		{"abc", 0, false},        // 非数值
		{struct{}{}, 0, false},   // 未知形态
		{int64(-9223372036854775808), -9223372036854775808, true}, // 极小值
	}
	for _, c := range cases {
		got, ok := dbi.ValToInt64(c.v)
		assert.Equal(t, c.ok, ok, "输入%#v的ok", c.v)
		if c.ok {
			assert.Equal(t, c.want, got, "输入%#v的值", c.v)
		}
	}
}
