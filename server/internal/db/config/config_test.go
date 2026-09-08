package config

import (
	"reflect"
	"testing"
)

func TestParseMaskExemptRoleIds(t *testing.T) {
	cases := []struct {
		name string
		val  any
		want []uint64
	}{
		{name: "nil", val: nil, want: nil},
		{name: "empty string", val: "", want: nil},
		{name: "blank string", val: "  ", want: nil},
		// 逗号分隔字符串（系统配置动态表单存字符串）
		{name: "comma separated", val: "1,2,3", want: []uint64{1, 2, 3}},
		{name: "comma separated with spaces", val: "1, 2 ,3", want: []uint64{1, 2, 3}},
		{name: "single id", val: "5", want: []uint64{5}},
		// 数组格式（直接改库为json数组）
		{name: "json array", val: []any{"1", "2"}, want: []uint64{1, 2}},
		{name: "empty array", val: []any{}, want: []uint64{}},
		// 非法值
		{name: "invalid number", val: "abc", want: []uint64{0}},
		{name: "unsupported type", val: 123, want: nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseMaskExemptRoleIds(c.val)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("parseMaskExemptRoleIds(%v) = %v, want %v", c.val, got, c.want)
			}
		})
	}
}
