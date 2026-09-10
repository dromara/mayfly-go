package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/pkg/model"
)

type ctxKeyType struct{}

// TestAsyncTaskCtxMustDetachFromRequest 异步SQL文件导入的上下文必须脱离请求取消信号
//
// 回归的缺陷：/dbTransfer/files/run 用 gox.GoCtx(rc.MetaCtx, ...) 异步执行导入，
// 而 MetaCtx 派生自 *http.Request 的 ctx——net/http 在 handler 返回时即取消它，
// 导致导入任务在第一条语句处就被 ExecReader 判定为「已取消」并回滚，功能整体失效。
// 本测试固化该前提：请求级 ctx 必然被取消，WithoutCancel 后的 ctx 不再被取消且仍保留 ctx 值
// （traceId/登录账号/语言等下游日志与国际化文案依赖这些值）。
func TestAsyncTaskCtxMustDetachFromRequest(t *testing.T) {
	handlerCtxCh := make(chan context.Context, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟 api 层：MetaCtx 包裹请求 ctx，并携带登录账号等值
		handlerCtxCh <- context.WithValue(r.Context(), ctxKeyType{}, &model.LoginAccount{Id: 1, Username: "tester"})
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	reqCtx := <-handlerCtxCh
	// 等待 net/http 在 handler 返回后取消请求 ctx
	require.Eventually(t, func() bool { return reqCtx.Err() != nil }, time.Second, 10*time.Millisecond,
		"请求上下文未被取消：异步导入透传请求ctx的风险判断已变化，请重新评估")

	taskCtx := context.WithoutCancel(reqCtx)
	select {
	case <-taskCtx.Done():
		t.Fatal("异步任务上下文被请求结束取消，导入将中断并回滚")
	default:
	}
	// ctx 值必须仍可读取，否则日志链路（traceId）与国际化提示会退化
	assert.Equal(t, &model.LoginAccount{Id: 1, Username: "tester"}, taskCtx.Value(ctxKeyType{}), "WithoutCancel 丢失了ctx值")
}
