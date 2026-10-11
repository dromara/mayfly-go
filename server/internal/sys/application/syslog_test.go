package application

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
)

// captureSyslogRepo 只记下「保存时用的是哪个 ctx」与落库实体，不真连库
type captureSyslogRepo struct {
	*base.RepoImpl[*entity.SysLog]

	savedCtx context.Context
	saved    *entity.SysLog
}

func (r *captureSyslogRepo) Insert(ctx context.Context, e *entity.SysLog) error {
	r.savedCtx = ctx
	r.saved = e
	return nil
}

func (r *captureSyslogRepo) GetPageList(*entity.SysLogQuery, ...string) (*model.PageResult[*entity.SysLog], error) {
	return nil, nil
}

// newLogReqCtx 造一次「已批准待执行工单被改期」的请求上下文：MetaCtx 处于已取消状态，
// 正是真实故障现场（响应已返回、日志保存才在协程里跑起来）
func newLogReqCtx(metaCtx context.Context) *req.Ctx {
	gin.SetMode(gin.TestMode)
	gc, _ := gin.CreateTestContext(httptest.NewRecorder())
	gc.Request = httptest.NewRequest("POST", "/flow/procinsts/55/reschedule-exec", nil)

	rc := req.NewCtxWithGin(gc)
	rc.MetaCtx = metaCtx
	rc.WithLog(req.NewLogSave("流程-改期执行"))
	rc.ReqParam = map[string]any{"id": 55, "plannedExecTime": "2026-10-12 08:00:00"}
	return rc
}

// TestSaveFromReqWritesAfterRequestContextCancelled 操作日志必须能在「响应已返回」之后仍写进库。
//
// 回归的真实故障：日志保存被放在响应返回之后的协程里，却沿用请求 ctx —— net/http 在请求结束时
// 已把它 cancel，于是每一条审计记录都在驱动层失败（实测 INSERT INTO t_sys_log → rows:0 +
// "context canceled"），而原实现连返回的错误都没检查，审计页面从此永远停在某一刻且界面上不报任何错。
// 对运维平台来说，丢的是「谁在什么时候做了什么」这条唯一凭证
func TestSaveFromReqWritesAfterRequestContextCancelled(t *testing.T) {
	repo := &captureSyslogRepo{RepoImpl: &base.RepoImpl[*entity.SysLog]{}}
	app := &syslogAppImpl{syslogRepo: repo}

	ctx, cancel := context.WithCancel(context.Background())
	ctx = contextx.WithTraceId(ctx)
	ctx = contextx.WithLoginAccount(ctx, &model.LoginAccount{Id: 1, Username: "admin"})
	cancel()

	app.SaveFromReq(newLogReqCtx(ctx))

	if repo.saved == nil {
		t.Fatal("审计记录未被写入：保存链路根本没走到仓储")
	}
	if repo.savedCtx == nil {
		t.Fatal("传给仓储的 ctx 为 nil")
	}
	if repo.savedCtx.Err() != nil {
		t.Fatalf("落库用的 ctx 已失效，驱动层会直接丢弃这条审计记录: %v", repo.savedCtx.Err())
	}
	// 剥掉的只能是取消信号：traceId 丢了就无法与请求日志串联，登录账号丢了审计归属人会退化成 '-'
	if contextx.GetTraceId(repo.savedCtx) == "" {
		t.Fatal("traceId 必须保留，否则这批写入无法与请求日志串联")
	}
	if la := contextx.GetLoginAccount(repo.savedCtx); la == nil || la.Username != "admin" {
		t.Fatalf("登录账号必须一并交给落库，否则审计记录的归属人会被写成 '-' 或退化成系统账号，实际: %+v", la)
	}
}
