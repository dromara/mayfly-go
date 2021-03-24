package models

import (
	"encoding/json"
	"mayfly-go/base/biz"
	"mayfly-go/base/rediscli"
)

const machineKey = "ccbscf:machines"

type Machine struct {
	Name     string `json:"name"`
	Ip       string `json:"ip"`       // IP地址
	Username string `json:"username"` // 用户名
	Password string `json:"-"`
	Port     int    `json:"port"` // 端口号
}

func (c *Machine) CreateMachine() {
	biz.IsTrue(!rediscli.HExist(machineKey, c.Ip), "该机器已存在")
	val, _ := json.Marshal(c)
	rediscli.HSet(machineKey, c.Ip, val)
}

func DeleteMachine(ip string) {
	rediscli.HDel(machineKey, ip)
}

func GetMachineByIp(ip string) *Machine {
	machine := &Machine{}
	json.Unmarshal([]byte(rediscli.HGet(machineKey, ip)), machine)
	return machine
}

func GetMachineList() map[string]string {
	return rediscli.HGetAll(machineKey)
}
