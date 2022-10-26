package vo

import "time"

type MachineVO struct {
	Id                 *uint64    `json:"id"`
	Name               *string    `json:"name"`
	Username           *string    `json:"username"`
	Ip                 *string    `json:"ip"`
	Port               *int       `json:"port"`
	AuthMethod         *int8      `json:"authMethod"`
	Status             *int8      `json:"status"`
	EnableSshTunnel    *int8      `json:"enableSshTunnel"`    // 是否启用ssh隧道
	SshTunnelMachineId *uint64    `json:"sshTunnelMachineId"` // ssh隧道机器id
	CreateTime         *time.Time `json:"createTime"`
	Creator            *string    `json:"creator"`
	CreatorId          *int64     `json:"creatorId"`
	UpdateTime         *time.Time `json:"updateTime"`
	Modifier           *string    `json:"modifier"`
	ModifierId         *int64     `json:"modifierId"`
	HasCli             bool       `json:"hasCli" gorm:"-"`
	Remark             *string    `json:"remark"`
	EnableRecorder     int8       `json:"enableRecorder"`
	TagId              uint64     `json:"tagId"`
	TagPath            string     `json:"tagPath"`
}

type MachineScriptVO struct {
	Id          *int64  `json:"id"`
	Name        *string `json:"name"`
	Script      *string `json:"script"`
	Type        *int    `json:"type"`
	Description *string `json:"description"`
	Params      *string `json:"params"`
	MachineId   *uint64 `json:"machineId"`
}

type MachineFileVO struct {
	Id        *int64  `json:"id"`
	Name      *string `json:"name"`
	Path      *string `json:"path"`
	Type      *int    `json:"type"`
	MachineId *uint64 `json:"machineId"`
}

type MachineFileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Type    string `json:"type"`
	Mode    string `json:"mode"`
	ModTime string `json:"modTime"`
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
