package dbi

import (
	"context"
	"errors"
	"fmt"
)

// TLSStatus 一次连接**实际协商出**的加密事实。
//
// 与「实例配置了什么」是两件事：配置写着 verify-full 也可能因为库侧只支持明文而回落，
// 只有从服务端视角读到协商结果才算证据。因此本结构刻意区分三种状态：
//   - Available=false：该方言读不到（实现不了），不是「没加密」；
//   - Available=true + Encrypted=false：确实以明文在跑；
//   - Available=true + Encrypted=true：拿到了版本与套件。
//
// 把「读不到」渲染成「未加密」会制造假警报，渲染成「已加密」则是假安心，二者都必须避免。
type TLSStatus struct {
	Available bool   `json:"available"`
	Encrypted bool   `json:"encrypted"`
	Version   string `json:"version,omitempty"`
	Cipher    string `json:"cipher,omitempty"`
}

// ErrTLSStatusUnavailable 方言能查、但本次查不到本连接的状态（如系统视图缺行）。
//
// 与 ErrUnsupportedKind 分开：前者是「观测缺口」（下次可能读到），后者是「该方言没这能力」。
// 上层对二者可以给不同的文案，但都必须标成不可判定。
var ErrTLSStatusUnavailable = errors.New("tls status of the current connection is unavailable")

// TLSStatusInspector 可选能力契约：读取当前连接实际的 TLS 协商结果。
//
// 与 SessionInspector 同属「可选能力 + 运行期类型断言」一族：未实现的方言由
// MetadataReader 代理返回 ErrUnsupportedKind，能力协商端点也不吐该 feature，
// 因此不存在「显示了入口却报不支持」的谎报路径。
type TLSStatusInspector interface {
	// TLSStatus 查询**本连接**的加密状态。
	//
	// 实现必须用只作用于自身会话的查询（如 mysql 的 SHOW STATUS、pg 的
	// pg_stat_ssl WHERE pid = pg_backend_pid()），不得依赖需要高权限的全局视图——
	// 取不到时应返回错误，由上层如实标注「不可判定」。
	TLSStatus(ctx context.Context) (*TLSStatus, error)
}

// TLSStatus 代理方言的 TLSStatusInspector 能力；方言未实现则返回 ErrUnsupportedKind。
//
// 返回错误时上层**不得**把零值 TLSStatus 当成「已加密」或「未加密」使用——
// Available=false 才是「读不到」的唯一合法表达。
func (m *MetadataReader) TLSStatus(ctx context.Context) (*TLSStatus, error) {
	inspector, ok := m.provider.(TLSStatusInspector)
	if !ok {
		return nil, fmt.Errorf("%w: tls_status", ErrUnsupportedKind)
	}
	return inspector.TLSStatus(ctx)
}
