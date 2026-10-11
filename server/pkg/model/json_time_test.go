package model

import "testing"

// TestJsonTimeUnmarshalJSON 校验入参统一时间结构体对空串/正常串/畸形串的处理：
// 空串归零值（= 未设置），正常串按本地时区解析，畸形串必须报错而非静默吞成零值。
// 注：JSON null 由 encoding/json 对指针字段直接置 nil，不会进入本方法，故此处不测 null。
func TestJsonTimeUnmarshalJSON(t *testing.T) {
	var j JsonTime

	if err := j.UnmarshalJSON([]byte(`""`)); err != nil {
		t.Fatalf(`empty string must be zero, got error: %v`, err)
	}
	if !j.IsZero() {
		t.Fatal(`empty string should decode to zero time`)
	}

	if err := j.UnmarshalJSON([]byte(`"2026-10-09 14:30:00"`)); err != nil {
		t.Fatalf("valid datetime must parse, got error: %v", err)
	}
	if j.Year() != 2026 || j.Month() != 10 || j.Day() != 9 || j.Hour() != 14 || j.Minute() != 30 {
		t.Fatalf("parsed wrong datetime: %v", j.Time)
	}

	// 非法时间（月/日/时分秒越界）必须报错，不能静默吞成零值（那会把拼错的入参伪装成「立即执行」）
	if err := j.UnmarshalJSON([]byte(`"2026-13-45 99:99:99"`)); err == nil {
		t.Fatal("malformed datetime must return an error")
	}
}
