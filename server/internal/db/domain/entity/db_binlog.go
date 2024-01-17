package entity

import (
	"mayfly-go/pkg/runner"
	"time"
)

const (
	BinlogDownloadInterval = time.Minute * 15
)

// BinlogFile is the metadata of the MySQL binlog file.
type BinlogFile struct {
	Name string
	Size int64

	// Sequence is parsed from Name and is for the sorting purpose.
	Sequence       int64
	FirstEventTime time.Time
	LastEventTime  time.Time
	Downloaded     bool
}

var _ DbJob = (*DbBinlog)(nil)

// DbBinlog 数据库备份任务
type DbBinlog struct {
	DbJobBaseImpl
}

func NewDbBinlog(instanceId uint64) *DbBinlog {
	job := &DbBinlog{}
	job.Id = instanceId
	job.DbInstanceId = instanceId
	return job
}

func (b *DbBinlog) GetDbName() string {
	// binlog 是全库级别的
	return ""
}

func (b *DbBinlog) Schedule() (time.Time, error) {
	switch b.GetJobBase().LastStatus {
	case DbJobSuccess:
		return time.Time{}, runner.ErrFinished
	case DbJobFailed:

		return time.Now().Add(BinlogDownloadInterval), nil
	default:
		return time.Now(), nil
	}
}

func (b *DbBinlog) Update(_ runner.Job) {}

func (b *DbBinlog) IsEnabled() bool {
	return true
}

func (b *DbBinlog) SetEnabled(_ bool) {}

func (b *DbBinlog) GetInterval() time.Duration {
	return 0
}
