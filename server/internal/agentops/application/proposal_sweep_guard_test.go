package application

import (
	"context"
	"testing"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"

	"github.com/stretchr/testify/require"
)

// stalledRepoStub 只实现停滞清扫真正用到的方法（其余由嵌入接口兜底，接口漂移时编译期报错）
type stalledRepoStub struct {
	repository.Proposal
	n int
}

// SelectByCond 签名与 base.Repo 一致：`cond any` 而不是 *model.QueryCond
func (s *stalledRepoStub) SelectByCond(_ any, _ ...string) ([]*entity.Proposal, error) {
	out := make([]*entity.Proposal, 0, s.n)
	for i := 0; i < s.n; i++ {
		out = append(out, &entity.Proposal{Id: uint64(i + 1)})
	}
	return out, nil
}

// TestSweepStalledProposalsNeverBlocks 停滞清扫在队列满时必须停下，绝不能阻塞。
//
// 真机风险形态：审批路径的投递是**阻塞式** select（队列满时等执行体空出名额，见 submitForExecution），
// 那是给调用方的背压；但清扫路径没有调用方可背压，且 Init 里这轮清扫排在 Run 工作池启动与
// cron 绑定**之前**。一旦清扫用了阻塞式投递，积压超过队列容量就会把 Init 挂住——而阻塞不报错，
// 日志里只有一行 Warn，表现是「进程已起、HTTP 可用，但队列不消费、定时任务不调度」。
//
// 用看门狗判：若把清扫改回阻塞式投递（或让 trySubmit 恒成功），本用例会超时/计数不符而不是静默通过
func TestSweepStalledProposalsNeverBlocks(t *testing.T) {
	const queueCap = 10 // 故意远小于停滞条数，模拟积压把队列塞满
	gw := &proposalGatewayImpl{
		execStarted: true,
		execQueue:   make(chan uint64, queueCap),
		execStop:    make(chan struct{}),
	}
	gw.Repo = &stalledRepoStub{n: proposalStallSweepBatch + 30}

	done := make(chan int, 1)
	go func() {
		requeued, err := gw.SweepStalledProposals(context.Background())
		if err != nil {
			done <- -1
			return
		}
		done <- requeued
	}()

	select {
	case requeued := <-done:
		require.Equal(t, queueCap, requeued,
			"队列满后必须停止投递：继续塞会把阻塞式投递的等待叠加起来拖住 Init")
	case <-time.After(3 * time.Second):
		t.Fatal("停滞清扫在队列满时阻塞了：单轮上限或失败即停缺失，会把 Init（工作池启动/cron 绑定）一起卡住")
	}
}
