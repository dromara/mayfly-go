package api

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mayfly-go/pkg/model"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// FileDel此前make(len(ids))预填零值后append，多传文件id时前面多出len(ids)个0，
// 导致误删id为0的记录。parseFileIds必须只返回真实有效的id
func TestParseFileIds(t *testing.T) {
	// 正常多id解析
	assert.Equal(t, []uint64{1, 2, 3}, parseFileIds("1,2,3"))

	// 单id
	assert.Equal(t, []uint64{42}, parseFileIds("42"))

	// 非法片段过滤，不影响有效id（原实现会让非法值转为0混入结果）
	assert.Equal(t, []uint64{1, 2}, parseFileIds("1,abc,2"))

	// 零值id过滤，杜绝误删id为0的记录
	assert.Equal(t, []uint64{1}, parseFileIds("0,1"))
	assert.Empty(t, parseFileIds("0,0"))

	// 空串与纯分隔符
	assert.Empty(t, parseFileIds(""))
	assert.Empty(t, parseFileIds(","))

	// 片段前后空格可容忍（cast转换前TrimSpace）
	assert.Equal(t, []uint64{5, 6}, parseFileIds(" 5 , 6 "))

	// 负数无效
	assert.Empty(t, parseFileIds("-1,-2"))
}

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
