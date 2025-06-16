package vo

import (
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
	"time"
)

type MachineVO struct {
	model.ExtraData
	tagentity.ResourceTags // 标签信息
	tagentity.AuthCerts    // 授权凭证信息

	Id                 uint64     `json:"id"`
	Code               string     `json:"code"`
	Name               string     `json:"name"`
	Protocol           int        `json:"protocol"`
	Ip                 string     `json:"ip"`
	Port               int        `json:"port"`
	Status             *int8      `json:"status"`
	SshTunnelMachineId int        `json:"sshTunnelMachineId"` // ssh隧道机器id
	CreateTime         *time.Time `json:"createTime"`
	Creator            *string    `json:"creator"`
	CreatorId          *int64     `json:"creatorId"`
	UpdateTime         *time.Time `json:"updateTime"`
	Modifier           *string    `json:"modifier"`
	ModifierId         *int64     `json:"modifierId"`
	Remark             *string    `json:"remark"`
	EnableRecorder     int8       `json:"enableRecorder"`

	Stat map[string]any `json:"stat" gorm:"-"`
}

func (m *MachineVO) GetCode() string {
	return m.Code
}

type SimpleMachineVO struct {
	Id     uint64  `json:"id"`
	Code   string  `json:"code"`
	Name   string  `json:"name"`
	Ip     string  `json:"ip"`
	Port   int     `json:"port"`
	Remark *string `json:"remark"`
}

type MachineScriptVO struct {
	Id          *int64  `json:"id"`
	Name        *string `json:"name"`
	Script      *string `json:"script"`
	Type        *int    `json:"type"`
	Category    string  `json:"category"`
	Description *string `json:"description"`
	Params      *string `json:"params"`
	MachineId   *uint64 `json:"machineId"`
}

// 机器记录任务
type MachineCronJobVO struct {
	tagentity.RelateTags // 标签信息

	Id              uint64 `json:"id"`
	Key             string `json:"key"`
	Name            string `json:"name"`
	Cron            string `json:"cron"` // cron
	Script          string `json:"script"`
	Status          int    `json:"status"`
	SaveExecResType int    `json:"saveExecResType"`
	Remark          string `json:"remark"`
	Running         bool   `json:"running" gorm:"-"` // 是否运行中
}

func (mcj *MachineCronJobVO) GetRelateId() uint64 {
	return mcj.Id
}

type MachineFileVO struct {
	Id        int64  `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Type      int8   `json:"type"`
	MachineId uint64 `json:"machineId"`
}

type MachineFileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Type    string `json:"type"`
	Mode    string `json:"mode"`
	ModTime string `json:"modTime"`

	UID uint32 `json:"uid"`
	GID uint32 `json:"gid"`
}

type MachineFileInfos []MachineFileInfo

func (s MachineFileInfos) Len() int { return len(s) }

func (s MachineFileInfos) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s MachineFileInfos) Less(i, j int) bool {
	if s[i].Type != s[j].Type {
		return s[i].Type > s[j].Type
	}
	return s[i].Name < s[j].Name
}

type MachineCmdConfVO struct {
	tagentity.RelateTags // 标签信息
	model.Model

	Name     string              `json:"name"`
	Cmds     model.Slice[string] `json:"cmds" gorm:"type:varchar"` // 命令配置，要加gorm标签才会正确解析model.Slice
	Status   int8                `json:"execCmds"`                 // 状态
	Stratege string              `json:"stratege"`                 // 策略，空禁用
	Remark   string              `json:"remark"`                   // 备注
}

func (mcc *MachineCmdConfVO) GetRelateId() uint64 {
	return mcc.Id
}
