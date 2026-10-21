package model

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
)

/*************** Map  ***************/

type Map[K comparable, V any] map[K]V

// Scan 实现 sql.Scanner 接口
func (m *Map[K, V]) Scan(value any) error {
	if v, ok := value.([]byte); ok && len(v) > 0 {
		return json.Unmarshal(v, m)
	}
	return nil
}

// Value 实现 driver.Valuer 接口
func (m Map[K, V]) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

var _ driver.Valuer = &Map[string, any]{}
var _ sql.Scanner = &Map[string, any]{}

/*************** Slice  ***************/

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

/*************** ExtraData  ***************/

// ExtraData 带有额外其他信息字段的结构体
type ExtraData struct {
	Extra collx.M `json:"extra" gorm:"type:varchar(2000)"`
}

// SetExtraValue 设置额外信息字段值：val 为零值（空串/false/0）时**删除该键**。
//
// 为什么零值要删键而不是跳过：基座此前对空值直接 return，所有一切「把值改回空/关成 false/
// 配置改回 0」的编辑都静默不生效（保存成功、旧值赖在 JSON 里，无报错）。典型受害形态：
//   - ResourceAuthCert.CiphertextClear 传 "" 想抹掉私钥口令，实际口令仍在 extra 里随列表接口回给前端；
//   - 会话元信息 contextTokens 归 0 时读到的是上一次的旧值；
//   - file 从对象存储换回本地存储时，旧 bucket 键残留，拼接存储路径时会拿到已废弃的 bucket。
//
// 读取侧天然自洽：GetExtraString/GetExtraInt/GetExtraBool 对缺键返回对应零值，所以「没键」与
// 「键值为零」对外等价。容器为 nil 且值为零时直接返回，不会为空删除而凭空建出一个 `{}` 落库。
//
// 需要「没传就保留旧值」的调用方（如不回显的密文/口令）必须自己先判空再显式传旧值，
// 不能依赖这里的跳过语义——参考 tag/application/resouce_auth_cert.go 里 passphrase 的合并写法。
func (m *ExtraData) SetExtraValue(key string, val any) *ExtraData {
	if anyx.IsBlank(val) {
		if m.Extra != nil {
			delete(m.Extra, key)
		}
		return m
	}
	if m.Extra != nil {
		m.Extra[key] = val
	} else {
		m.Extra = collx.M{key: val}
	}
	return m
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
