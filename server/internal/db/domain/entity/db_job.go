package entity

import (
	"context"
	"fmt"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/runner"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/timex"
	"time"
)

const LastResultSize = 256

type DbJobKey = runner.JobKey

type DbJobStatus = runner.JobStatus

const (
	DbJobUnknown = runner.JobUnknown
	DbJobDelay   = runner.JobDelay
	DbJobReady   = runner.JobWaiting
	DbJobRunning = runner.JobRunning
	DbJobRemoved = runner.JobRemoved
)

const (
	DbJobSuccess DbJobStatus = 0x20 + iota
	DbJobFailed
)

type DbJobType = string

const (
	DbJobTypeBackup  DbJobType = "db-backup"
	DbJobTypeRestore DbJobType = "db-restore"
)

const (
	DbJobNameBackup  = "数据库备份"
	DbJobNameRestore = "数据库恢复"
)

var _ runner.Job = (DbJob)(nil)

type DbJobBase interface {
	model.ModelI
	runner.Job

	GetId() uint64
	GetJobType() DbJobType
	SetJobType(typ DbJobType)
	GetJobBase() *DbJobBaseImpl
	SetLastStatus(status DbJobStatus, err error)
	IsEnabled() bool
}

type DbJob interface {
	DbJobBase

	SetRun(fn func(ctx context.Context, job DbJob))
	SetRunnable(fn func(job DbJob, next runner.NextFunc) bool)
}

func NewDbJob(typ DbJobType) DbJob {
	switch typ {
	case DbJobTypeBackup:
		return &DbBackup{
			DbJobBaseImpl: &DbJobBaseImpl{
				jobType: DbJobTypeBackup},
		}
	case DbJobTypeRestore:
		return &DbRestore{
			DbJobBaseImpl: &DbJobBaseImpl{
				jobType: DbJobTypeRestore},
		}
	default:
		panic(fmt.Sprintf("invalid DbJobType: %v", typ))
	}
}

var _ DbJobBase = (*DbJobBaseImpl)(nil)

type DbJobBaseImpl struct {
	model.Model

	DbInstanceId uint64         // 数据库实例ID
	DbName       string         // 数据库名称
	Enabled      bool           // 是否启用
	StartTime    time.Time      // 开始时间
	Interval     time.Duration  // 间隔时间
	Repeated     bool           // 是否重复执行
	LastStatus   DbJobStatus    // 最近一次执行状态
	LastResult   string         // 最近一次执行结果
	LastTime     timex.NullTime // 最近一次执行时间
	Deadline     time.Time      `gorm:"-" json:"-"` // 计划执行时间
	run          runner.RunFunc
	runnable     runner.RunnableFunc
	jobType      DbJobType
	jobKey       runner.JobKey
	jobStatus    runner.JobStatus
}

func NewDbBJobBase(instanceId uint64, dbName string, jobType DbJobType, enabled bool, repeated bool, startTime time.Time, interval time.Duration) *DbJobBaseImpl {
	return &DbJobBaseImpl{
		DbInstanceId: instanceId,
		DbName:       dbName,
		jobType:      jobType,
		Enabled:      enabled,
		Repeated:     repeated,
		StartTime:    startTime,
		Interval:     interval,
	}
}

func (d *DbJobBaseImpl) GetJobType() DbJobType {
	return d.jobType
}

func (d *DbJobBaseImpl) SetJobType(typ DbJobType) {
	d.jobType = typ
}

func (d *DbJobBaseImpl) SetLastStatus(status DbJobStatus, err error) {
	var statusName, jobName string
	switch status {
	case DbJobRunning:
		statusName = "运行中"
	case DbJobSuccess:
		statusName = "成功"
	case DbJobFailed:
		statusName = "失败"
	default:
		return
	}
	switch d.jobType {
	case DbJobTypeBackup:
		jobName = DbJobNameBackup
	case DbJobTypeRestore:
		jobName = DbJobNameRestore
	default:
		jobName = d.jobType
	}
	d.LastStatus = status
	var result = jobName + statusName
	if err != nil {
		result = fmt.Sprintf("%s: %v", result, err)
	}
	d.LastResult = stringx.TruncateStr(result, LastResultSize)
	d.LastTime = timex.NewNullTime(time.Now())
}

func (d *DbJobBaseImpl) GetId() uint64 {
	if d == nil {
		return 0
	}
	return d.Id
}

func (d *DbJobBaseImpl) GetDeadline() time.Time {
	return d.Deadline
}

func (d *DbJobBaseImpl) Schedule() bool {
	if d.IsFinished() || !d.Enabled {
		return false
	}
	switch d.LastStatus {
	case DbJobSuccess:
		if d.Interval == 0 {
			return false
		}
		lastTime := d.LastTime.Time
		if lastTime.Sub(d.StartTime) < 0 {
			lastTime = d.StartTime.Add(-d.Interval)
		}
		d.Deadline = lastTime.Add(d.Interval - lastTime.Sub(d.StartTime)%d.Interval)
	case DbJobFailed:
		d.Deadline = time.Now().Add(time.Minute)
	default:
		d.Deadline = d.StartTime
	}
	return true
}

func (d *DbJobBaseImpl) IsFinished() bool {
	return !d.Repeated && d.LastStatus == DbJobSuccess
}

func (d *DbJobBaseImpl) Update(job runner.Job) {
	jobBase := job.(DbJob).GetJobBase()
	d.StartTime = jobBase.StartTime
	d.Interval = jobBase.Interval
}

func (d *DbJobBaseImpl) GetJobBase() *DbJobBaseImpl {
	return d
}

func (d *DbJobBaseImpl) IsEnabled() bool {
	return d.Enabled
}

func (d *DbJobBaseImpl) Run(ctx context.Context) {
	if d.run == nil {
		return
	}
	d.run(ctx)
}

func (d *DbJobBaseImpl) Runnable(next runner.NextFunc) bool {
	if d.runnable == nil {
		return true
	}
	return d.runnable(next)
}

func FormatJobKey(typ DbJobType, jobId uint64) DbJobKey {
	return fmt.Sprintf("%v-%d", typ, jobId)
}

func (d *DbJobBaseImpl) GetKey() DbJobKey {
	if len(d.jobKey) == 0 {
		d.jobKey = FormatJobKey(d.jobType, d.Id)
	}
	return d.jobKey
}

func (d *DbJobBaseImpl) GetStatus() DbJobStatus {
	return d.jobStatus
}

func (d *DbJobBaseImpl) SetStatus(status DbJobStatus) {
	d.jobStatus = status
}
