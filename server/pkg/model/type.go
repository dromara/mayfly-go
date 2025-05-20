package model

import (
	"database/sql/driver"
	"encoding/json"
	"mayfly-go/pkg/utils/collx"
)

type Map[K comparable, V any] map[K]V

func (m *Map[K, V]) Scan(value any) error {
	if v, ok := value.([]byte); ok && len(v) > 0 {
		return json.Unmarshal(v, m)
	}
	return nil
}

func (m Map[K, V]) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

type Slice[T int | uint64 | string | Map[string, any]] []T

func (s *Slice[T]) Scan(value any) error {
	if v, ok := value.([]byte); ok && len(v) > 0 {
		return json.Unmarshal(v, s)
	}
	return nil
}

func (s Slice[T]) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// ExtraData 带有额外其他信息字段的结构体
type ExtraData struct {
	Extra collx.M `json:"extra" gorm:"type:varchar(2000)"`
}

// SetExtraValue 设置额外信息字段值
func (m *ExtraData) SetExtraValue(key string, val any) {
	if m.Extra != nil {
		m.Extra[key] = val
	} else {
		m.Extra = collx.M{key: val}
	}
}

// GetExtraVal 获取额外信息字段值
func (e ExtraData) GetExtraVal(key string) any {
	if e.Extra == nil {
		return nil
	}
	return e.Extra[key]
}

// GetExtraString 获取额外信息中的string类型字段值
func (e ExtraData) GetExtraString(key string) string {
	return e.Extra.GetStr(key)
}

func (e ExtraData) GetExtraStringSlice(key string) []string {
	return e.Extra.GetStrSlice(key)
}

// GetExtraInt 获取额外信息中的int类型字段值
func (e ExtraData) GetExtraInt(key string) int {
	return e.Extra.GetInt(key)
}

// GetExtraInt64 获取额外信息中的int64类型字段值
func (e ExtraData) GetExtraInt64(key string) int64 {
	return e.Extra.GetInt64(key)
}

func (e ExtraData) GetExtraFloat32(key string) float32 {
	return e.Extra.GetFloat32(key)
}

// GetExtraBool 获取额外信息中的bool类型字段值
func (e ExtraData) GetExtraBool(key string) bool {
	return e.Extra.GetBool(key)
}
