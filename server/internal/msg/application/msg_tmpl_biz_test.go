package application

import (
	"context"
	"errors"
	"testing"

	"mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/internal/msg/imsg"
	"mayfly-go/internal/msg/infra/persistence"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/ioc"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// flowTaskNotifyBizType 流程定义待办通知的业务类型。
// 真机上出问题的那条绑定就是它：t_msg_tmpl_biz 里 (flow:task:notify, biz_id=1) 绑的模板
// 在 t_msg_tmpl_channel 里没有任何关联行，待办通知从来发不出去，而界面上流程定义「保存成功」
const flowTaskNotifyBizType = "flow:task:notify"

// tmplSendableReject 模拟模板服务返回的「发不出去」业务错误（真实文案由 imsg 按请求语言生成）
var tmplSendableReject = errors.New("消息模板[brokenTmpl]未关联任何消息渠道，通知发不出去")

// fakeMsgTmplSendable 只实现 CheckSendableById 的模板服务替身。
//
// 内嵌 nil 接口即可满足 MsgTmpl：本组用例只应触达 CheckSendableById，其余方法一旦被调到就
// 当场 panic——那说明绑定保存走偏了路径，比返回零值静默通过更有信号
type fakeMsgTmplSendable struct {
	MsgTmpl

	checked []uint64
	err     error
}

func (f *fakeMsgTmplSendable) CheckSendableById(_ context.Context, tmplId uint64) error {
	f.checked = append(f.checked, tmplId)
	return f.err
}

// setupMsgTmplBizApps 打开 sqlite 内存库（模板 / 模板渠道关联 / 模板业务关联三表），
// 装配一对绑定真实仓储的 app：模板服务是真的，故「无渠道的模板会被拒」这条断言走的是
// 生产同一份判据，而不是替身里手写的返回值
func setupMsgTmplBizApps(t *testing.T) (*msgTmplBizAppImpl, *msgTmplAppImpl) {
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
	require.NoError(t, db.AutoMigrate(&entity.MsgTmpl{}, &entity.MsgTmplChannel{}, &entity.MsgTmplBiz{}))

	prev := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = prev })

	registerMsgRepoOnce.Do(persistence.InitIoc)

	tmplApp := &msgTmplAppImpl{}
	tmplApp.Repo = ioc.Get[repository.MsgTmpl]()
	tmplApp.msgTmplChannelRepo = ioc.Get[repository.MsgTmplChannel]()

	bizApp := &msgTmplBizAppImpl{}
	bizApp.Repo = ioc.Get[repository.MsgTmplBiz]()
	bizApp.msgTmplApp = tmplApp
	return bizApp, tmplApp
}

// boundTmplId 读回某业务当前绑定的模板 id；未绑定返回 0。
// 断言「被拒的保存不留半截写入」全靠它
func boundTmplId(t *testing.T, app *msgTmplBizAppImpl, bizType string, bizId uint64) uint64 {
	t.Helper()

	row := &entity.MsgTmplBiz{BizId: bizId, BizType: bizType}
	if err := app.GetByCond(row); err != nil {
		return 0
	}
	return row.TmplId
}

// seedBinding 直接写一条绑定行，绕过保存校验。
// 这正是生产上的既有形态：坏绑定是在校验上线之前存进去的，任何「改无关字段」的保存都不该被它拦下
func seedBinding(t *testing.T, app *msgTmplBizAppImpl, bizType string, bizId, tmplId uint64) {
	t.Helper()

	require.NoError(t, app.Insert(context.Background(), &entity.MsgTmplBiz{
		BizId:   bizId,
		BizType: bizType,
		TmplId:  tmplId,
	}))
}

// TestSaveBizTmplRejectsUnsendableTemplate 新建与变更两条路径都必须在保存期拒掉发不出去的模板，
// 且库里不留半截写入。
//
// 为什么必须在保存期：待办通知经 EventBus 异步派发（见 msg/init 对 EventTopicBizMsgTmplSend
// 的注册），失败只回到 eventbus 日志，界面上流程定义早已「保存成功」，待办人永远收不到通知。
// 唯一调用者 flow/application/procdef.go 把 SaveBizTmpl 放在 SaveProcdef 的同一个 Tx 里，
// 故这里返回错误会连带回滚流程定义本身——不会出现「定义存了、模板没绑上」的中间态
func TestSaveBizTmplRejectsUnsendableTemplate(t *testing.T) {
	bizApp, tmplApp := setupMsgTmplBizApps(t)
	ctx := zhCtx()

	noChannel := seedTmpl(t, tmplApp, "noChannelTmpl", entity.TmplStatusEnable)
	sendable := seedTmpl(t, tmplApp, "sendableTmpl", entity.TmplStatusEnable, 4)
	wantErr := i18n.TC(ctx, imsg.ErrMsgTmplNoChannel, "code", "noChannelTmpl")

	tests := []struct {
		name       string
		bizId      uint64
		bindFirst  uint64 // 预置绑定，0 表示「本来没绑」（新建路径）
		wantBefore uint64 // 被拒后库里应该还是这个值
	}{
		{name: "新建绑定", bizId: 11, bindFirst: 0, wantBefore: 0},
		{name: "变更已有绑定", bizId: 12, bindFirst: sendable.Id, wantBefore: sendable.Id},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.bindFirst != 0 {
				seedBinding(t, bizApp, flowTaskNotifyBizType, tt.bizId, tt.bindFirst)
			}

			err := bizApp.SaveBizTmpl(ctx, &dto.MsgTmplBizSave{
				TmplId:  noChannel.Id,
				BizType: flowTaskNotifyBizType,
				BizId:   tt.bizId,
			})

			require.Error(t, err, "绑一个发不出去的模板不该保存成功")
			require.Equal(t, wantErr, err.Error(), "拒绝文案必须是可直接当 toast 用的中文译文")
			assert.Equal(t, tt.wantBefore, boundTmplId(t, bizApp, flowTaskNotifyBizType, tt.bizId),
				"被拒的保存改动了库里的绑定")
		})
	}
}

// TestSaveBizTmplAcceptsSendableTemplate 可发模板照常绑定；已绑定时是覆盖而不是再插一行
func TestSaveBizTmplAcceptsSendableTemplate(t *testing.T) {
	bizApp, tmplApp := setupMsgTmplBizApps(t)
	ctx := zhCtx()

	first := seedTmpl(t, tmplApp, "firstTmpl", entity.TmplStatusEnable, 4)
	second := seedTmpl(t, tmplApp, "secondTmpl", entity.TmplStatusEnable, 5)

	require.NoError(t, bizApp.SaveBizTmpl(ctx, &dto.MsgTmplBizSave{
		TmplId: first.Id, BizType: flowTaskNotifyBizType, BizId: 21,
	}))
	assert.Equal(t, first.Id, boundTmplId(t, bizApp, flowTaskNotifyBizType, 21))

	require.NoError(t, bizApp.SaveBizTmpl(ctx, &dto.MsgTmplBizSave{
		TmplId: second.Id, BizType: flowTaskNotifyBizType, BizId: 21,
	}))
	assert.Equal(t, second.Id, boundTmplId(t, bizApp, flowTaskNotifyBizType, 21))

	rows, err := bizApp.Repo.SelectByCond(&entity.MsgTmplBiz{BizId: 21, BizType: flowTaskNotifyBizType})
	require.NoError(t, err)
	assert.Len(t, rows, 1, "同一业务被写进了多行绑定，后续 GetByCond 会读到不确定的一行")
}

// TestSaveBizTmplUnchangedBindingSkipsSendabilityCheck 绑定没变时不校验、不重写。
//
// 这条守的是「校验不得追诉既往」：生产库里的坏绑定是在校验上线之前存进去的
// （真机实测 flow:task:notify → tmpl 1 就没有渠道关联）。若对未变的绑定也校验，
// 用户改流程定义的名字就会被一条自己根本没碰的旧绑定拦下，而且界面上那个下拉框显示的
// 还是原值，他无从判断错在哪
func TestSaveBizTmplUnchangedBindingSkipsSendabilityCheck(t *testing.T) {
	bizApp, _ := setupMsgTmplBizApps(t)
	fake := &fakeMsgTmplSendable{err: tmplSendableReject}
	bizApp.msgTmplApp = fake

	seedBinding(t, bizApp, flowTaskNotifyBizType, 31, 77)

	require.NoError(t, bizApp.SaveBizTmpl(zhCtx(), &dto.MsgTmplBizSave{
		TmplId: 77, BizType: flowTaskNotifyBizType, BizId: 31,
	}), "绑定未变时被一条既有坏绑定拦下，会让流程定义的无关修改全部无法保存")
	assert.Empty(t, fake.checked, "绑定未变不该去查模板可发性")
	assert.Equal(t, uint64(77), boundTmplId(t, bizApp, flowTaskNotifyBizType, 31))
}

// TestSaveBizTmplUnbindSkipsSendabilityCheck 模板 ID 为 0 是解绑（前端 MsgTmplSelect 带
// clearable，清空后提交 0），必须放行且不去校验——校验一个「正要解除的绑定」毫无意义，
// 反而会让用户无法摘掉一条坏绑定
func TestSaveBizTmplUnbindSkipsSendabilityCheck(t *testing.T) {
	bizApp, _ := setupMsgTmplBizApps(t)
	fake := &fakeMsgTmplSendable{err: tmplSendableReject}
	bizApp.msgTmplApp = fake
	ctx := zhCtx()

	// 已绑定 → 删除绑定行
	seedBinding(t, bizApp, flowTaskNotifyBizType, 41, 88)
	require.NoError(t, bizApp.SaveBizTmpl(ctx, &dto.MsgTmplBizSave{
		TmplId: 0, BizType: flowTaskNotifyBizType, BizId: 41,
	}))
	assert.Zero(t, boundTmplId(t, bizApp, flowTaskNotifyBizType, 41), "解绑后绑定行应被删除")

	// 本来就没绑 → 空操作，不报错也不建行
	require.NoError(t, bizApp.SaveBizTmpl(ctx, &dto.MsgTmplBizSave{
		TmplId: 0, BizType: flowTaskNotifyBizType, BizId: 42,
	}))
	assert.Zero(t, boundTmplId(t, bizApp, flowTaskNotifyBizType, 42))

	assert.Empty(t, fake.checked, "解绑路径不该去查模板可发性")
}

// TestSaveBizTmplRequiresBizIdAndBizType 业务标识缺失时立刻拒，且不落库。
//
// 少了 bizId 的绑定行会变成「谁的待办通知都不知道」的孤儿数据：Send 按 (bizType, bizId) 反查，
// 永远匹配不上它，删也不好删（DeleteByBiz 按同样两个字段定位）
func TestSaveBizTmplRequiresBizIdAndBizType(t *testing.T) {
	bizApp, _ := setupMsgTmplBizApps(t)
	fake := &fakeMsgTmplSendable{}
	bizApp.msgTmplApp = fake
	ctx := zhCtx()

	tests := []struct {
		name    string
		param   *dto.MsgTmplBizSave
		wantMsg string
	}{
		{
			name:    "缺业务 ID",
			param:   &dto.MsgTmplBizSave{TmplId: 5, BizType: flowTaskNotifyBizType},
			wantMsg: i18n.TC(ctx, imsg.ErrMsgTmplBizIdRequired),
		},
		{
			name:    "缺业务类型",
			param:   &dto.MsgTmplBizSave{TmplId: 5, BizId: 51},
			wantMsg: i18n.TC(ctx, imsg.ErrMsgTmplBizTypeRequired),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bizApp.SaveBizTmpl(ctx, tt.param)
			require.Error(t, err)
			require.Equal(t, tt.wantMsg, err.Error())
			require.NotEmpty(t, tt.wantMsg, "期望文案为空，说明 imsg 漏配，断言会退化成恒真")
		})
	}
	assert.Empty(t, fake.checked, "业务标识都没校验过就不该去查模板")
	assert.Zero(t, boundTmplId(t, bizApp, flowTaskNotifyBizType, 51), "被拒的保存不该留下绑定行")
}

// TestSaveBizTmplSkipsCheckWhenTmplAppMissing 模板服务未注入时跳过校验而不是 panic。
//
// 与 agentops 侧 checkNotifyTmplSendable 同一取向：一个「只是没法校验」的依赖不该把
// 整条业务保存主链拖死。真机上 IOC 装配齐全，这里守的是单测装配与将来重构时的退化
func TestSaveBizTmplSkipsCheckWhenTmplAppMissing(t *testing.T) {
	bizApp, _ := setupMsgTmplBizApps(t)
	bizApp.msgTmplApp = nil

	require.NotPanics(t, func() {
		require.NoError(t, bizApp.SaveBizTmpl(zhCtx(), &dto.MsgTmplBizSave{
			TmplId: 99, BizType: flowTaskNotifyBizType, BizId: 61,
		}))
	})
	assert.Equal(t, uint64(99), boundTmplId(t, bizApp, flowTaskNotifyBizType, 61))
}
