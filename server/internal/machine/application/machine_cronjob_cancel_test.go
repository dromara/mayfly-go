package application

import (
	"context"
	"testing"
)

// TestCronJobCancelRuns 删除任务时取消该任务全部在跑执行（同一任务可能已在多台机器上并发执行）
func TestCronJobCancelRuns(t *testing.T) {
	app := &machineCronJobAppImpl{}

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	entry1 := app.registerRun("job-a", cancel1)
	entry2 := app.registerRun("job-a", cancel2)

	app.cancelRuns("job-a")

	if ctx1.Err() == nil || ctx2.Err() == nil {
		t.Fatalf("cancelRuns 应取消全部在跑执行: ctx1=%v ctx2=%v", ctx1.Err(), ctx2.Err())
	}

	app.unregisterRun("job-a", entry1)
	app.unregisterRun("job-a", entry2)
	app.runCancelsMu.Lock()
	_, exists := app.runCancels["job-a"]
	app.runCancelsMu.Unlock()
	if exists {
		t.Fatal("全部执行结束并摘除后不应残留取消表条目")
	}
}

// TestCronJobUnregisterOnlyOwnEntry 收尾只能摘除自己那条登记，不能误删同 key 的其他在跑执行
func TestCronJobUnregisterOnlyOwnEntry(t *testing.T) {
	app := &machineCronJobAppImpl{}

	ctx1, cancel1 := context.WithCancel(context.Background())
	entry1 := app.registerRun("job-b", cancel1)
	ctx2, cancel2 := context.WithCancel(context.Background())
	app.registerRun("job-b", cancel2)
	defer cancel2()

	app.unregisterRun("job-b", entry1)
	app.cancelRuns("job-b")

	if ctx1.Err() != nil {
		t.Fatal("已摘除的登记不应再被取消")
	}
	if ctx2.Err() == nil {
		t.Fatal("仍在登记的执应被取消")
	}
}
