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
