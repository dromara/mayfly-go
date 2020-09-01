package machine

import (
	"github.com/siddontang/go/log"
	"io/ioutil"
	"mayfly-go/base"
	"mayfly-go/base/utils"
	"mayfly-go/models"
	"time"
)

const BasePath = "./machine/shell/"

const MonitorTemp = "cpuRate:{cpuRate}%,memRate:{memRate}%,sysLoad:{sysLoad}\n"

// shell文件内容缓存，避免每次读取文件
var shellCache = make(map[string]string)

func GetProcessByName(cli *Cli, name string) string {
	return cli.Run(getShellContent("sys_info"))
}

func GetSystemInfo(cli *Cli) string {
	return cli.Run(getShellContent("system_info"))
}

func GetMonitorInfo(cli *Cli) *models.MachineMonitor {
	mm := new(models.MachineMonitor)
	res := cli.Run(getShellContent("monitor"))
	resMap := make(map[string]interface{})
	utils.ReverStrTemplate(MonitorTemp, res, resMap)

	err := utils.Map2Struct(resMap, mm)
	if err != nil {
		log.Error("解析machine monitor: %s", err.Error())
		return nil
	}
	mm.MachineId = cli.machine.Id
	mm.CreateTime = time.Now()
	return mm
}

// 获取shell内容
func getShellContent(name string) string {
	cacheShell := shellCache[name]
	if cacheShell != "" {
		return cacheShell
	}
	bytes, err := ioutil.ReadFile(BasePath + name + ".sh")
	base.ErrIsNil(err, "获取shell文件失败")
	shellStr := string(bytes)
	shellCache[name] = shellStr
	return shellStr
}
