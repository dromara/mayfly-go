package entity

import (
	"mayfly-go/pkg/model"
)

type Db struct {
	model.Model

	Name      string `orm:"column(name)" json:"name"`
	Type      string `orm:"column(type)" json:"type"` // 类型，mysql oracle等
	Host      string `orm:"column(host)" json:"host"`
	Port      int    `orm:"column(port)" json:"port"`
	Network   string `orm:"column(network)" json:"network"`
	Username  string `orm:"column(username)" json:"username"`
	Password  string `orm:"column(password)" json:"-"`
	Database  string `orm:"column(database)" json:"database"`
	Params    string `json:"params"`
	ProjectId uint64
	Project   string
	EnvId     uint64
	Env       string

	EnableSSH int    `orm:"column(enable_ssh)" json:"enable_ssh"`
	SSHHost   string `orm:"column(ssh_host)" json:"ssh_host"`
	SSHPort   int    `orm:"column(ssh_port)" json:"ssh_port"`
	SSHUser   string `orm:"column(ssh_user)" json:"ssh_user"`
	SSHPass   string `orm:"column(ssh_pass)" json:"-"`
}
