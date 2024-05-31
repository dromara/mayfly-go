package runner

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mayfly-go/pkg/utils/timex"
	"sync"
	"testing"
	"time"
)

var _ Job = &testJob{}

func newTestJob(key string) *testJob {
	return &testJob{
		Key: key,
	}
}

type testJob struct {
	Key    JobKey
	status int
}

func (t *testJob) Update(_ Job) {}

func (t *testJob) GetKey() JobKey {
	return t.Key
}

func (t *testJob) SetStatus(status JobStatus, err error) {}

func (t *testJob) SetEnabled(enabled bool, desc string) {}

func TestRunner_Close(t *testing.T) {
	signal := make(chan struct{}, 1)
	waiting := sync.WaitGroup{}
	waiting.Add(1)
	runner := NewRunner[*testJob](1, func(ctx context.Context, job *testJob) error {
		waiting.Done()
		timex.SleepWithContext(ctx, time.Hour)
		signal <- struct{}{}
		return nil
	})
	go func() {
		job := &testJob{
			Key: "close",
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
		want error
	}
	testCases := []testCase{
		{
			name: "first job",
			job:  newTestJob("single"),
			want: nil,
		},
		{
			name: "second job",
			job:  newTestJob("dual"),
			want: nil,
		},
		{
			name: "repetitive job",
			job:  newTestJob("dual"),
			want: ErrJobExist,
		},
	}
	runner := NewRunner[*testJob](1, func(ctx context.Context, job *testJob) error {
		timex.SleepWithContext(ctx, time.Hour)
		return nil
	})
	defer runner.Close()
	for _, tc := range testCases {
		err := runner.Add(context.Background(), tc.job)
		if tc.want != nil {
			require.ErrorIs(t, err, tc.want)
			continue
		}
		require.NoError(t, err)
	}
}

func TestJob_UpdateStatus(t *testing.T) {
	const d = time.Millisecond * 20
	const (
		unknown = iota
		running
		finished
	)
	runner := NewRunner[*testJob](1, func(ctx context.Context, job *testJob) error {
		job.status = running
		timex.SleepWithContext(ctx, d*2)
		job.status = finished
		return nil
	})
	first := newTestJob("first")
	second := newTestJob("second")
	_ = runner.Add(context.Background(), first)
	_ = runner.Add(context.Background(), second)

	time.Sleep(d)
	assert.Equal(t, running, first.status)
	assert.Equal(t, unknown, second.status)

	time.Sleep(d * 2)
	assert.Equal(t, finished, first.status)
	assert.Equal(t, running, second.status)

	time.Sleep(d * 2)
	assert.Equal(t, finished, first.status)
	assert.Equal(t, finished, second.status)
}
