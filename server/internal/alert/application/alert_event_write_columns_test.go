package application

import (
	"context"
	"testing"

	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"

	"github.com/stretchr/testify/require"
)

// captureEventRepo 只截获写入口的列清单（其余方法由嵌入接口兜底，接口漂移时编译期报错）
type captureEventRepo struct {
	repository.AlertEvent
	event *entity.AlertEvent
	cols  []string
}

func (r *captureEventRepo) GetById(_ uint64, _ ...string) (*entity.AlertEvent, error) {
	return r.event, nil
}

func (r *captureEventRepo) UpdateById(_ context.Context, e *entity.AlertEvent, cols ...string) error {
	r.cols = cols
	r.event = e
	return nil
}

// TestEventWritePathsUseColumnLimitedUpdates 告警事件的写入口必须**按列**更新，不得整行写回。
//
// 缺陷类：`gormx.UpdateById` 在不传列时是 `Select(columns).Updates(model)`，即「写所有非零字段」。
// 评估循环与通知刷新循环是两个独立 cron，双方都持有自己那份事件快照：评估侧的一次触摸写回会把
// 通知侧刚写的 notify_count/last_notify_time 覆盖成旧值（反之亦然），后果是重复通知节奏偏移、
// 已通知次数倒退——且没有任何报错，只能从时间线反推。`UpdateNotifyInfo`/`UpdateEscalationLvl`
// 早已按列写（注释写明就是为此），但评估侧的 TouchActiveEvent / Recover 与用户侧的 Ack / Close
// 当时仍是整行写回，本用例把它们一并钉住。
func TestEventWritePathsUseColumnLimitedUpdates(t *testing.T) {
	bg := context.Background()
	firing := func() *entity.AlertEvent {
		return &entity.AlertEvent{Id: 1, Status: entity.AlertEventStatusFiring, NotifyCount: 7}
	}

	assertCols := func(t *testing.T, name string, got []string, want []string) {
		t.Helper()
		require.NotEmptyf(t, got, "%s 改成整行写回了：会用旧快照覆盖并发写入的通知/升级列", name)
		for _, col := range want {
			require.Containsf(t, got, col, "%s 缺少本应写入的列 %s", name, col)
		}
		// 这三个列属于别的写入口，任何越权写回都会造成并发覆盖
		for _, forbidden := range []string{"notify_count", "last_notify_time", "escalation_lvl"} {
			require.NotContainsf(t, got, forbidden, "%s 不应写 %s（那是通知/升级链路的列）", name, forbidden)
		}
	}

	t.Run("TouchActiveEvent", func(t *testing.T) {
		repo := &captureEventRepo{event: firing()}
		app := &alertEventAppImpl{}
		app.Repo = repo
		require.NoError(t, app.TouchActiveEvent(bg, repo.event, "mem_rate", 91.5))
		// resource_name 必须在列清单里：评估链路在「历史活跃事件资源名为空」时先回填再触摸，
		// 少了这一列回填会永久丢失
		assertCols(t, "TouchActiveEvent", repo.cols,
			[]string{"last_trigger_time", "trigger_count", "metric", "current_value", "resource_name"})
		require.Equal(t, 1, repo.event.TriggerCount, "触摸必须把累计触发次数 +1 带进写入")
	})

	t.Run("Recover", func(t *testing.T) {
		repo := &captureEventRepo{event: firing()}
		app := &alertEventAppImpl{}
		app.Repo = repo
		require.NoError(t, app.Recover(bg, repo.event))
		assertCols(t, "Recover", repo.cols, []string{"status", "recover_time"})
		require.Equal(t, entity.AlertEventStatusRecovered, repo.event.Status)
	})

	t.Run("Ack", func(t *testing.T) {
		repo := &captureEventRepo{event: firing()}
		app := &alertEventAppImpl{}
		app.Repo = repo
		require.NoError(t, app.Ack(bg, 1, 42))
		assertCols(t, "Ack", repo.cols, []string{"status", "ack_user_id", "ack_time"})
	})

	t.Run("Close", func(t *testing.T) {
		repo := &captureEventRepo{event: firing()}
		app := &alertEventAppImpl{}
		app.Repo = repo
		require.NoError(t, app.Close(bg, 1))
		assertCols(t, "Close", repo.cols, []string{"status"})
	})
}
