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

type DbJobStatus int

const (
	DbJobRunning DbJobStatus = iota
	DbJobSuccess
	DbJobFailed
)

type DbJobType = string

const (
	DbJobTypeBackup  DbJobType = "db-backup"
	DbJobTypeRestore DbJobType = "db-restore"
	DbJobTypeBinlog  DbJobType = "db-binlog"
)

const (
	DbJobNameBackup  = "数据库备份"
	DbJobNameRestore = "数据库恢复"
	DbJobNameBinlog  = "BINLOG同步"
)

var _ runner.Job = (DbJob)(nil)

type DbJobBase interface {
	model.ModelI

	GetKey() string
	GetJobType() DbJobType
	SetJobType(typ DbJobType)
	GetJobBase() *DbJobBaseImpl
	SetLastStatus(status DbJobStatus, err error)
}

type DbJob interface {
	runner.Job
	DbJobBase

	GetDbName() string
	Schedule() (time.Time, error)
	IsEnabled() bool
	SetEnabled(enabled bool)
	Update(job runner.Job)
	GetInterval() time.Duration
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
	LastStatus   DbJobStatus    // 最近一次执行状态
	LastResult   string         // 最近一次执行结果
	LastTime     timex.NullTime // 最近一次执行时间
	jobType      DbJobType
	jobKey       runner.JobKey
}

func NewDbBJobBase(instanceId uint64, jobType DbJobType) *DbJobBaseImpl {
	return &DbJobBaseImpl{
		DbInstanceId: instanceId,
		jobType:      jobType,
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
	case DbJobNameBinlog:
		jobName = DbJobNameBinlog
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

func (d *DbJobBaseImpl) GetJobBase() *DbJobBaseImpl {
	return d
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
