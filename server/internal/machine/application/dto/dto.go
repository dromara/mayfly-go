package dto

import (
	"mayfly-go/internal/machine/domain/entity"
	tagentity "mayfly-go/internal/tag/domain/entity"
)

type SaveMachine struct {
	Machine      *entity.Machine
	TagCodePaths []string
	AuthCerts    []*tagentity.ResourceAuthCert
}

type MachineFileOp struct {
	MachineId    uint64 `json:"machineId" binding:"required" form:"machineId"`
	Protocol     int    `json:"protocol" binding:"required" form:"protocol"`
	AuthCertName string `json:"authCertName"  binding:"required" form:"authCertName"` // 授权凭证
	Path         string `json:"path" form:"path"`                                     // 文件路径
}

type SaveMachineCmdConf struct {
	CmdConf   *entity.MachineCmdConf
	CodePaths []string
}

type SaveMachineCronJob struct {
	CronJob   *entity.MachineCronJob
	CodePaths []string
}

// BatchCmdResult 单台机器的批量命令执行结果。
// 单台失败/超时不影响他台，逐台聚合后整体返回
type BatchCmdResult struct {
	MachineId    uint64 `json:"machineId"`
	Name         string `json:"name"`
	Ip           string `json:"ip"`
	Port         int    `json:"port"`
	AuthCertName string `json:"authCertName"`
	Username     string `json:"username"`

	Success      bool   `json:"success"`
	Output       string `json:"output"`
	Error        string `json:"error"`
	PolicyNotice string `json:"policyNotice"` // 命令策略「仅提醒」命中提示（不是失败）
	CostMs       int64  `json:"costMs"`
	Timeout      bool   `json:"timeout"`
}
