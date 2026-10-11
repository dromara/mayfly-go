package req

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// newLogTestCtx 造一个带请求对象的 Ctx：LogHandler 会读请求方法与路径，缺了会直接 panic
func newLogTestCtx(t *testing.T) *Ctx {
	t.Helper()
	gin.SetMode(gin.TestMode)
	gc, _ := gin.CreateTestContext(httptest.NewRecorder())
	gc.Request = httptest.NewRequest("POST", "/flow/procinsts/1/reschedule-exec", nil)
	return NewCtxWithGin(gc)
}

// TestLogHandlerSavesOnlyMarkedActions 只有声明了「要保存」的动作才触发日志保存函数。
//
// 保存函数是不是该把日志落库、用哪个 ctx 落库，由实现方（sys/application.SaveFromReq）决定；
// 这里是统一派发点，只负责「派发与否」，不认识数据库与事务
func TestLogHandlerSavesOnlyMarkedActions(t *testing.T) {
	saved := make(chan string, 2)
	SetSaveLogFunc(func(rc *Ctx) { saved <- rc.GetLogInfo().Description })
	t.Cleanup(func() { SetSaveLogFunc(nil) })

	saveRc := newLogTestCtx(t)
	saveRc.Conf = &Conf{logInfo: NewLogSave("流程-改期执行")}
	noSaveRc := newLogTestCtx(t)
	noSaveRc.Conf = &Conf{logInfo: NewLog("只记录不保存")}

	for _, rc := range []*Ctx{saveRc, noSaveRc} {
		if err := LogHandler(rc); err != nil {
			t.Fatalf("LogHandler returned an error: %v", err)
		}
	}

	select {
	case description := <-saved:
		if description != "流程-改期执行" {
			t.Fatalf("只有声明要保存的动作应被派发，实际派发的是: %q", description)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("声明要保存的动作没有被派发")
	}

	select {
	case description := <-saved:
		t.Fatalf("未声明要保存的动作也被派发了: %q", description)
	case <-time.After(100 * time.Millisecond):
	}
}
