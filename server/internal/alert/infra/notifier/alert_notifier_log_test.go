package notifier

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/internal/alert/domain/service"
	msgentity "mayfly-go/internal/msg/domain/entity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeNotifyLogRepo 只实现 Create 的通知日志仓库替身。
//
// 内嵌 nil 接口即可满足 repository.AlertNotifyLog：本组用例只应触达 Create，
// 其余统计方法一旦被调到就当场 panic——那说明写日志的路径走偏了，
// 比返回零值让断言静默通过更有信号
type fakeNotifyLogRepo struct {
	repository.AlertNotifyLog

	rows []*entity.AlertNotifyLog
	err  error
}

func (r *fakeNotifyLogRepo) Create(_ context.Context, log *entity.AlertNotifyLog) error {
	if r.err != nil {
		return r.err
	}
	r.rows = append(r.rows, log)
	return nil
}

// notifyChannels 两条渠道，取自真机复现的形态：4 是本地验证用的钉钉接收器，1 是真实企微群
func notifyChannels() []*msgentity.MsgChannel {
	local := &msgentity.MsgChannel{Name: "本地钉钉验证"}
	local.Id = 4
	wecom := &msgentity.MsgChannel{Name: "test-企微"}
	wecom.Id = 1
	return []*msgentity.MsgChannel{local, wecom}
}

// attributedTarget 渠道 4 由通知策略 2 路由而来，渠道 1 由升级策略 11 推送而来
func attributedTarget() service.NotifyTarget {
	return service.NotifyTarget{ChannelIds: []uint64{4, 1}}.
		WithNotifyPolicyBinding(2, []uint64{4}).
		WithEscalationBinding(11, []uint64{1})
}

// TestRecordNotifyLogsWritesPolicySourceType 通知日志必须把「来源类型 + 策略 id」一起落库。
//
// 真机复现的缺陷面：policy_id 一列同时写过通知策略与升级策略的 id，两表主键重叠后，
// 日志里的 policy_id=11 在通知策略表查无此行，「这条通知为什么发给了这个渠道」无从溯源
func TestRecordNotifyLogsWritesPolicySourceType(t *testing.T) {
	repo := &fakeNotifyLogRepo{}
	n := &alertNotifierImpl{notifyLogRepo: repo}

	n.recordNotifyLogs(context.Background(), 100, notifyChannels(), attributedTarget(), nil)

	require.Len(t, repo.rows, 2, "每条渠道各一行日志")
	assert.Equal(t, uint64(2), repo.rows[0].PolicyId)
	assert.Equal(t, entity.NotifyPolicyTypeNotify, repo.rows[0].PolicyType)
	assert.Equal(t, uint64(11), repo.rows[1].PolicyId)
	assert.Equal(t, entity.NotifyPolicyTypeEscalation, repo.rows[1].PolicyType,
		"升级策略推送的渠道若不带类型，与通知策略 11 在日志里完全无法区分")
	for _, row := range repo.rows {
		assert.Equal(t, uint64(100), row.EventId)
		assert.EqualValues(t, 1, row.Status, "SendMsg 未报错即视为投递成功")
		assert.Empty(t, row.ErrorMsg)
	}
	assert.Equal(t, "本地钉钉验证", repo.rows[0].ChannelName, "渠道名冗余落库，渠道改名后历史日志仍可读")
}

// TestRecordNotifyLogsTruncatesErrMsgByRune 失败原因按字符截断，不得切出半个汉字。
//
// 按字节切（errMsg[:500]）在中文文案上有 2/3 概率产生非法 UTF-8，MySQL utf8mb4 严格模式下
// 整行 INSERT 报 Error 1366，而写入失败只留一条 Warn——恰恰是「投递失败」这种最需要留痕的行
// 反而静默丢掉
func TestRecordNotifyLogsTruncatesErrMsgByRune(t *testing.T) {
	repo := &fakeNotifyLogRepo{}
	n := &alertNotifierImpl{notifyLogRepo: repo}
	long := strings.Repeat("渠道不可达-", 200) // 1200 字符 / 3200 字节
	require.Greater(t, len(long), notifyLogErrMsgMax, "用例前提：必须长到需要截断")

	n.recordNotifyLogs(context.Background(), 102, notifyChannels()[:1], service.NotifyTarget{}, errors.New(long))

	require.Len(t, repo.rows, 1)
	msg := repo.rows[0].ErrorMsg
	assert.True(t, utf8.ValidString(msg), "截断后必须是合法 UTF-8，否则 MySQL 会拒掉整行 INSERT")
	assert.Len(t, []rune(msg), notifyLogErrMsgMax, "按字符截断到列宽")
	assert.EqualValues(t, 2, repo.rows[0].Status, "有 sendErr 即投递失败")
}

// TestRecordNotifyLogsKeepsShortErrMsg 未超长的错误文案原样保留（截断不得误伤正常长度）
func TestRecordNotifyLogsKeepsShortErrMsg(t *testing.T) {
	repo := &fakeNotifyLogRepo{}
	n := &alertNotifierImpl{notifyLogRepo: repo}

	n.recordNotifyLogs(context.Background(), 103, notifyChannels()[:1], service.NotifyTarget{},
		errors.New("消息模板未关联任何消息渠道"))

	require.Len(t, repo.rows, 1)
	assert.Equal(t, "消息模板未关联任何消息渠道", repo.rows[0].ErrorMsg)
}

// TestRecordNotifyLogsSwallowsRepoError 写日志失败不得影响通知主链路：
// 通知已经发出去了，日志留不下只是少一条取证记录，不该把它变成一次通知失败
func TestRecordNotifyLogsSwallowsRepoError(t *testing.T) {
	repo := &fakeNotifyLogRepo{err: errors.New("db down")}
	n := &alertNotifierImpl{notifyLogRepo: repo}

	require.NotPanics(t, func() {
		n.recordNotifyLogs(context.Background(), 104, notifyChannels(), attributedTarget(), nil)
	})
}

// TestRecordNotifyLogsSkipsWithoutRepoOrChannels 仓库未注入或无渠道时直接返回，
// 不产生任何写入（也不 panic——单测与降级装配下 notifyLogRepo 可能为 nil）
func TestRecordNotifyLogsSkipsWithoutRepoOrChannels(t *testing.T) {
	require.NotPanics(t, func() {
		(&alertNotifierImpl{}).recordNotifyLogs(context.Background(), 105, notifyChannels(), service.NotifyTarget{}, nil)
	}, "notifyLogRepo 为 nil 时必须跳过")

	repo := &fakeNotifyLogRepo{}
	(&alertNotifierImpl{notifyLogRepo: repo}).
		recordNotifyLogs(context.Background(), 106, nil, service.NotifyTarget{}, nil)
	assert.Empty(t, repo.rows, "无渠道时不该写任何日志行")
}
