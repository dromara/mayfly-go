package vo

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/model"
	"time"
)

type DbTransferTaskListVO struct {
	model.ExtraData

	Id         uint64     `json:"id"`
	CreateTime *time.Time `json:"createTime"`
	Creator    string     `json:"creator"`
	UpdateTime *time.Time `json:"updateTime"`
	Modifier   string     `json:"modifier"`

	RunningState     entity.TransferRunState `json:"runningState"`
	LogId            uint64                  `json:"logId"`
	TaskName         string                  `json:"taskName"`         // 任务名称
	Status           int                     `json:"status"`           // 任务状态 1启用 -1禁用
	CronEnabled      int                     `json:"cronEnabled"`      // 是否定时  1是 -1否
	Cron             string                  `json:"cron"`             // 定时任务cron表达式
	Mode             int                     `json:"mode"`             // 数据迁移方式，1、迁移到数据库  2、迁移到文件
	TargetFileDbType string                  `json:"targetFileDbType"` // 目标文件数据库类型
	FileSaveDays     int                     `json:"fileSaveDays"`     // 文件保存天数

	CheckedKeys string `json:"checkedKeys"` // 选中需要迁移的表
	DeleteTable int    `json:"deleteTable"` // 创建表前是否删除表
	NameCase    int    `json:"nameCase"`    // 表名、字段大小写转换  1无  2大写  3小写
	Strategy    int    `json:"strategy"`    // 迁移策略  1全量  2增量
	// 迁移并发度：列表也需返回，编辑抽屉直接以列表行作为回填数据源，缺字段会把已保存的并发度重置回默认值
	Concurrency int `json:"concurrency"`

	SrcDbId     int64  `json:"srcDbId"`     // 源库id
	SrcDbName   string `json:"srcDbName"`   // 源库名
	SrcTagPath  string `json:"srcTagPath"`  // 源库tagPath
	SrcDbType   string `json:"srcDbType"`   // 源库类型
	SrcInstName string `json:"srcInstName"` // 源库实例名

	TargetDbId     int    `json:"targetDbId"`     // 目标库id
	TargetDbName   string `json:"targetDbName"`   // 目标库名
	TargetDbType   string `json:"targetDbType"`   // 目标库类型
	TargetInstName string `json:"targetInstName"` // 目标库实例名
	TargetTagPath  string `json:"targetTagPath"`  // 目标库tagPath
}

// DbTransferLogListVO 迁移执行日志列表行。
// 不含 runLog：运行日志为追加式大文本，列表一次返回15条会使响应体膨胀至数百 KB，由按日志 id 的接口单条获取
type DbTransferLogListVO struct {
	Id         uint64                      `json:"id"`
	CreateTime *time.Time                  `json:"createTime"`
	TaskId     uint64                      `json:"taskId"`
	Mode       entity.TransferMode         `json:"mode"`
	Purpose    entity.DbTransferLogPurpose `json:"purpose"`
	TargetFile string                      `json:"targetFile"`
	ErrText    string                      `json:"errText"`
	Status     int8                        `json:"status"`

	// 监控指标
	DurationMs int64 `json:"durationMs"`
	TotalRows  int64 `json:"totalRows"`
	TableCount int   `json:"tableCount"`
}

// DbTransferLogRunVO 单条迁移执行日志的运行日志内容
type DbTransferLogRunVO struct {
	Id     uint64 `json:"id"`
	Status int8   `json:"status"`
	RunLog string `json:"runLog"`
}
