package application

import (
	"context"
	"sync"
	"testing"

	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/internal/msg/imsg"
	"mayfly-go/internal/msg/infra/persistence"
	"mayfly-go/internal/msg/msgx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/ioc"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var registerMsgRepoOnce sync.Once

// setupMsgTmplApp 打开 sqlite 内存库（模板 + 模板渠道关联两表），装配一个绑定真实仓储的
// msgTmplAppImpl。
//
// msgChannelApp 刻意不装配：本组用例只覆盖 loadSendableTmpl 的三条拒绝分支与放行分支，
// 全都在「取渠道实体」之前返回。哪天真有分支走到了 GetByIds，nil 接口会当场 panic，
// 那正是需要补装配的信号，比静默跳过更有用
func setupMsgTmplApp(t *testing.T) *msgTmplAppImpl {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:               logger.Default.LogMode(logger.Silent),
		NamingStrategy:       gormx.MetaNamingStrategy(),
		DisableAutomaticPing: true,
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	// sqlite 内存库随最后一个连接关闭而消失，钉住单连接让全测试共用同一份数据
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	require.NoError(t, db.AutoMigrate(&entity.MsgTmpl{}, &entity.MsgTmplChannel{}))

	// 仓储经 gormx.DB(ctx) 取连接，无事务上下文时回落到 global.Db
	prev := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = prev })

	registerMsgRepoOnce.Do(persistence.InitIoc)

	app := &msgTmplAppImpl{}
	app.Repo = ioc.Get[repository.MsgTmpl]()
	app.msgTmplChannelRepo = ioc.Get[repository.MsgTmplChannel]()
	return app
}

// seedTmpl 建一个模板；channelIds 是它关联的渠道 id，不传即「未关联任何渠道」
func seedTmpl(t *testing.T, app *msgTmplAppImpl, code string, status entity.MsgTmplStatus, channelIds ...uint64) *entity.MsgTmpl {
	t.Helper()

	ctx := context.Background()
	tmpl := &entity.MsgTmpl{
		Name:    "tmpl-" + code,
		Code:    code,
		Title:   "title",
		Tmpl:    "hello {{.runNo}}",
		MsgType: msgx.MsgType(1), // text
		Status:  status,
	}
	require.NoError(t, app.Insert(ctx, tmpl))
	require.NotZero(t, tmpl.Id, "模板未落库，后续渠道关联无从挂靠")

	for _, channelId := range channelIds {
		require.NoError(t, app.msgTmplChannelRepo.Insert(ctx, &entity.MsgTmplChannel{
			TmplId:    tmpl.Id,
			ChannelId: channelId,
		}))
	}
	return tmpl
}

// zhCtx 中文请求上下文：错误文案在 NewBizI 构造时就按 ctx 语言译好，故断言必须用同语言的 ctx
func zhCtx() context.Context {
	return i18n.NewCtxWithLang(context.Background(), i18n.Zh_CN)
}

// TestCheckSendableRejectsUnsendableTemplates 三种「存得进去但发不出去」的模板都必须被拒。
//
// 这三条正是真机复现的静默失败面：告警触发器绑了未关联渠道的模板，保存时「操作成功」、
// alert→trigger→Run 全链跑通，结果通知只在服务端日志里留一条 ERROR，配置人毫不知情。
// 断言逐条钉住「哪个分支用哪个 msgId」，避免三处拒绝退化成同一句笼统文案
func TestCheckSendableRejectsUnsendableTemplates(t *testing.T) {
	app := setupMsgTmplApp(t)
	ctx := zhCtx()

	seedTmpl(t, app, "disabledTmpl", entity.TmplStatusDisable, 7)
	seedTmpl(t, app, "noChannelTmpl", entity.TmplStatusEnable)

	tests := []struct {
		name    string
		code    string
		wantMsg string
	}{
		{
			name:    "模板不存在",
			code:    "ghostTmpl",
			wantMsg: i18n.TC(ctx, imsg.ErrMsgTmplNotExist, "code", "ghostTmpl"),
		},
		{
			name:    "模板已停用",
			code:    "disabledTmpl",
			wantMsg: i18n.TC(ctx, imsg.ErrMsgTmplDisabled, "code", "disabledTmpl"),
		},
		{
			name:    "模板未关联任何渠道",
			code:    "noChannelTmpl",
			wantMsg: i18n.TC(ctx, imsg.ErrMsgTmplNoChannel, "code", "noChannelTmpl"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.CheckSendable(ctx, tt.code)
			require.Error(t, err)
			require.Equal(t, tt.wantMsg, err.Error())
			require.NotEmpty(t, tt.wantMsg, "期望文案为空，说明 imsg 漏配，断言会退化成恒真")
		})
	}
}

// TestCheckSendablePassesWhenChannelBound 启用 + 已关联渠道 → 放行
func TestCheckSendablePassesWhenChannelBound(t *testing.T) {
	app := setupMsgTmplApp(t)

	seedTmpl(t, app, "sendableTmpl", entity.TmplStatusEnable, 4)

	require.NoError(t, app.CheckSendable(zhCtx(), "sendableTmpl"))
}

// TestSendSharesSendabilityPredicateWithCheckSendable 发送期与保存期必须同一套判据。
//
// 这条是防漂移的主锁：CheckSendable 的全部价值在于「保存时放行的，发送时一定也放行」。
// 两处各写一份判据（历史上的 Send 就是自带三条 if）时，任何一侧新增/放宽条件都会让
// 保存期校验变成一句空话。这里逐个不可发模板比对两者返回的错误文本，一旦分叉立刻红。
// 只覆盖不可发分支：可发分支的 Send 会继续走到 msgChannelApp.GetByIds，而本组用例未装配它
func TestSendSharesSendabilityPredicateWithCheckSendable(t *testing.T) {
	app := setupMsgTmplApp(t)
	ctx := zhCtx()

	seedTmpl(t, app, "disabledTmpl", entity.TmplStatusDisable, 7)
	seedTmpl(t, app, "noChannelTmpl", entity.TmplStatusEnable)

	for _, code := range []string{"ghostTmpl", "disabledTmpl", "noChannelTmpl"} {
		checkErr := app.CheckSendable(ctx, code)
		sendErr := app.Send(ctx, code, map[string]any{"runNo": "r-1"})

		require.Errorf(t, checkErr, "模板 %s 应被 CheckSendable 拒绝", code)
		require.Errorf(t, sendErr, "模板 %s 应被 Send 拒绝", code)
		require.Equalf(t, checkErr.Error(), sendErr.Error(),
			"模板 %s 的保存期判据与发送期判据已分叉，保存时放行的配置会在运行时静默发不出去", code)
	}
}

// TestCheckSendableByIdRejectsUnsendableTemplates 按 ID 校验的可发性：模板不存在 / 已停用 /
// 未关联渠道都必须拒，可发的放行。
//
// 这个入口存在的理由：模板-业务关联（流程定义的待办通知）按 ID 绑定模板，拿不到编码。
// 真机实测的形态是 t_msg_tmpl_biz 里 flow:task:notify 绑的模板在 t_msg_tmpl_channel 里
// 没有任何关联行——待办通知走 EventBus 异步派发，失败只回到日志，界面上早已「保存成功」
func TestCheckSendableByIdRejectsUnsendableTemplates(t *testing.T) {
	app := setupMsgTmplApp(t)
	ctx := zhCtx()

	disabled := seedTmpl(t, app, "disabledTmpl", entity.TmplStatusDisable, 7)
	noChannel := seedTmpl(t, app, "noChannelTmpl", entity.TmplStatusEnable)
	sendable := seedTmpl(t, app, "sendableTmpl", entity.TmplStatusEnable, 4)

	tests := []struct {
		name    string
		id      uint64
		wantErr string
	}{
		{
			name:    "模板不存在",
			id:      404,
			wantErr: i18n.TC(ctx, imsg.ErrMsgTmplIdNotExist, "id", uint64(404)),
		},
		{
			name:    "模板已停用",
			id:      disabled.Id,
			wantErr: i18n.TC(ctx, imsg.ErrMsgTmplDisabled, "code", "disabledTmpl"),
		},
		{
			name:    "模板未关联任何渠道",
			id:      noChannel.Id,
			wantErr: i18n.TC(ctx, imsg.ErrMsgTmplNoChannel, "code", "noChannelTmpl"),
		},
		{
			name: "可发模板放行",
			id:   sendable.Id,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.CheckSendableById(ctx, tt.id)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Equal(t, tt.wantErr, err.Error())
			require.NotEmpty(t, tt.wantErr, "期望文案为空，说明 imsg 漏配，断言会退化成恒真")
		})
	}
}

// TestCheckSendableByIdSharesPredicateWithCheckSendable 按 ID 与按编码两个入口必须同一判据。
//
// 这是防绕过的第二道主锁：两个入口若各写一份判据，就会出现「按编码查会被拒、按 ID 绑却能
// 绑上」的旁路——而流程定义恰恰走的是按 ID 那条路。这里对同一模板比对两者的错误文本，
// 包括「不存在」分支（一个按编码报不存在、一个按 ID 报不存在，文案本就不同，故只比对
// 停用与无渠道这两种「模板存在但发不出去」的情形）
func TestCheckSendableByIdSharesPredicateWithCheckSendable(t *testing.T) {
	app := setupMsgTmplApp(t)
	ctx := zhCtx()

	tmpls := []*entity.MsgTmpl{
		seedTmpl(t, app, "disabledTmpl", entity.TmplStatusDisable, 7),
		seedTmpl(t, app, "noChannelTmpl", entity.TmplStatusEnable),
	}
	for _, tmpl := range tmpls {
		byCode := app.CheckSendable(ctx, tmpl.Code)
		byId := app.CheckSendableById(ctx, tmpl.Id)

		require.Errorf(t, byCode, "模板 %s 应被 CheckSendable 拒绝", tmpl.Code)
		require.Errorf(t, byId, "模板 %s 应被 CheckSendableById 拒绝", tmpl.Code)
		require.Equalf(t, byCode.Error(), byId.Error(),
			"模板 %s 的两个校验入口已分叉，按 ID 绑定这条路会绕过可发性校验", tmpl.Code)
	}
}
