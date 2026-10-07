package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

// MachineMetric 机器指标历史采样点。
//
// 时序语义：追加写 + 定期物理清理，不做软删除（用 CreateModelNLD，LogicDelete=false）。
// 字段口径与 mcm.Stats 及告警评估器 MachineEvaluator 对齐：
//   - CpuUsage = 100 - CPU.Idle
//   - MemUsage = (Total-Available)/Total*100
//   - DiskUsage = 根分区使用率，无根分区则取最满分区
//   - NetRx/NetTx 存累计字节，速率由查询侧相邻点差分得到（采集端不存 rate）
//   - Status 1 在线、0 离线；采集失败也落一个离线点，保证趋势不断线
type MachineMetric struct {
	model.CreateModelNLD

	MachineId   uint64    `json:"machineId" gorm:"not null;index:idx_machine_collect,priority:1;comment:机器id"`
	CollectTime time.Time `json:"collectTime" gorm:"not null;index:idx_machine_collect,priority:2;comment:采集时间"`
	CpuUsage    float64   `json:"cpuUsage" gorm:"comment:CPU使用率%"`
	MemUsage    float64   `json:"memUsage" gorm:"comment:内存使用率%"`
	DiskUsage   float64   `json:"diskUsage" gorm:"comment:磁盘使用率%"`
	Load1       float64   `json:"load1" gorm:"comment:1分钟负载"`
	Load5       float64   `json:"load5" gorm:"comment:5分钟负载"`
	Load10      float64   `json:"load10" gorm:"comment:10分钟负载"`
	NetRx       uint64    `json:"netRx" gorm:"comment:网络累计接收字节"`
	NetTx       uint64    `json:"netTx" gorm:"comment:网络累计发送字节"`
	Status      int8      `json:"status" gorm:"not null;comment:1在线 0离线"`
}

func (m *MachineMetric) TableName() string {
	return "t_machine_metric"
}
