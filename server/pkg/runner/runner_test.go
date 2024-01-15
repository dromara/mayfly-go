package runner

import (
	"context"
	"github.com/stretchr/testify/require"
	"mayfly-go/pkg/utils/timex"
	"sync"
	"testing"
	"time"
)

var _ Job = &testJob{}

func newTestJob(key string, runTime time.Duration) *testJob {
	return &testJob{
		deadline: time.Now(),
		Key:      key,
		run: func(ctx context.Context) {
			timex.SleepWithContext(ctx, runTime)
		},
	}
}

type testJob struct {
	run      RunFunc
	Key      JobKey
	status   JobStatus
	ran      bool
	deadline time.Time
}

func (t *testJob) Update(_ Job) {
}

func (t *testJob) GetDeadline() time.Time {
	return t.deadline
}

func (t *testJob) Schedule() bool {
	return !t.ran
}

func (t *testJob) Run(ctx context.Context) {
	if t.run == nil {
		return
	}
	t.run(ctx)
	t.ran = true
}

func (t *testJob) Runnable(_ NextFunc) bool {
	return true
}

func (t *testJob) GetKey() JobKey {
	return t.Key
}

func (t *testJob) GetStatus() JobStatus {
	return t.status
}

func (t *testJob) SetStatus(status JobStatus) {
	t.status = status
}

func TestRunner_Close(t *testing.T) {
	runner := NewRunner[*testJob](1)
	signal := make(chan struct{}, 1)
	waiting := sync.WaitGroup{}
	waiting.Add(1)
	go func() {
		job := &testJob{
			Key: "close",
			run: func(ctx context.Context) {
				waiting.Done()
				timex.SleepWithContext(ctx, time.Hour)
				signal <- struct{}{}
			},
		}
		_ = runner.Add(context.Background(), job)
	}()
	waiting.Wait()
	timer := time.NewTimer(time.Microsecond * 10)
	defer timer.Stop()
	runner.Close()
	select {
	case <-timer.C:
		require.FailNow(t, "runner 未能及时退出")
	case <-signal:
	}
}

func TestRunner_AddJob(t *testing.T) {
	type testCase struct {
		name string
		job  *testJob
		want bool
	}
	testCases := []testCase{
		{
			name: "first job",
			job:  newTestJob("single", time.Hour),
			want: true,
		},
		{
			name: "second job",
			job:  newTestJob("dual", time.Hour),
			want: true,
		},
		{
			name: "non repetitive job",
			job:  newTestJob("single", time.Hour),
			want: true,
		},
		{
			name: "repetitive job",
			job:  newTestJob("dual", time.Hour),
			want: true,
		},
	}
	runner := NewRunner[*testJob](1)
	defer runner.Close()
	for _, tc := range testCases {
		err := runner.Add(context.Background(), tc.job)
		require.NoError(t, err)
	}
}

func TestJob_UpdateStatus(t *testing.T) {
	const d = time.Millisecond * 20
	runner := NewRunner[*testJob](1)
	running := newTestJob("running", d*2)
	waiting := newTestJob("waiting", d*2)
	_ = runner.Add(context.Background(), running)
	_ = runner.Add(context.Background(), waiting)

	time.Sleep(d)
	require.Equal(t, JobRunning, running.status)
	require.Equal(t, JobWaiting, waiting.status)

	time.Sleep(d * 2)
	require.Equal(t, JobRemoved, running.status)
	require.Equal(t, JobRunning, waiting.status)

	time.Sleep(d * 2)
	require.Equal(t, JobRemoved, running.status)
	require.Equal(t, JobRemoved, waiting.status)
}
