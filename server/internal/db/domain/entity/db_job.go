package entity

import (
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
	DbJobRunning = runner.JobRunning
	DbJobSuccess = runner.JobSuccess
	DbJobFailed  = runner.JobFailed
)

type DbJobType string

func (typ DbJobType) String() string {
	return string(typ)
}

const (
	DbJobUnknown     DbJobType = "db-unknown"
	DbJobTypeBackup  DbJobType = "db-backup"
	DbJobTypeRestore DbJobType = "db-restore"
	DbJobTypeBinlog  DbJobType = "db-binlog"
)

const (
	DbJobNameUnknown = "未知任务"
	DbJobNameBackup  = "数据库备份"
	DbJobNameRestore = "数据库恢复"
	DbJobNameBinlog  = "BINLOG同步"
)

var _ runner.Job = (DbJob)(nil)

type DbJobBase interface {
	model.ModelI
}

type DbJob interface {
	runner.Job
	DbJobBase

	GetInstanceId() uint64
	GetKey() string
	GetJobType() DbJobType
	GetDbName() string
	Schedule() (time.Time, error)
	IsEnabled() bool
	IsExpired() bool
	SetEnabled(enabled bool, desc string)
	Update(job runner.Job)
	GetInterval() time.Duration
}

var _ DbJobBase = (*DbJobBaseImpl)(nil)

type DbJobBaseImpl struct {
	model.Model

	LastStatus DbJobStatus    // 最近一次执行状态
	LastResult string         // 最近一次执行结果
	LastTime   timex.NullTime // 最近一次执行时间
	jobKey     runner.JobKey
}

func (d *DbJobBaseImpl) getJobType() DbJobType {
	job, ok := any(d).(DbJob)
	if !ok {
		return DbJobUnknown
	}
	return job.GetJobType()
}

func (d *DbJobBaseImpl) setLastStatus(jobType DbJobType, status DbJobStatus, err error) {
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

	switch jobType {
	case DbJobTypeBackup:
		jobName = DbJobNameBackup
	case DbJobTypeRestore:
		jobName = DbJobNameRestore
	case DbJobTypeBinlog:
		jobName = DbJobNameBinlog
	default:
		jobName = jobType.String()
	}
	d.LastStatus = status
	var result = jobName + statusName
	if err != nil {
		result = fmt.Sprintf("%s: %v", result, err)
	}
	d.LastResult = stringx.TruncateStr(result, LastResultSize)
	d.LastTime = timex.NewNullTime(time.Now())
}

func FormatJobKey(typ DbJobType, jobId uint64) DbJobKey {
	return fmt.Sprintf("%v-%d", typ, jobId)
}

func (d *DbJobBaseImpl) getKey(jobType DbJobType) DbJobKey {
	if len(d.jobKey) == 0 {
		d.jobKey = FormatJobKey(jobType, d.Id)
	}
	return d.jobKey
}
