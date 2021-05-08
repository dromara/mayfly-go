package machine

import (
	"io/ioutil"
	"mayfly-go/base/biz"
	"mayfly-go/base/global"
	"mayfly-go/base/utils"
	"mayfly-go/devops/models"
	"time"
)

const BasePath = "./machine/shell/"

const MonitorTemp = "cpuRate:{cpuRate}%,memRate:{memRate}%,sysLoad:{sysLoad}\n"

// shell文件内容缓存，避免每次读取文件
var shellCache = make(map[string]string)

func (c *Cli) GetProcessByName(name string) (*string, error) {
	return c.Run(getShellContent("sys_info"))
}

func (c *Cli) GetSystemInfo() (*string, error) {
	return c.Run(getShellContent("system_info"))
}

func (c *Cli) GetMonitorInfo() *models.MachineMonitor {
	mm := new(models.MachineMonitor)
	res, _ := c.Run(getShellContent("monitor"))
	if res == nil {
		return nil
	}
	resMap := make(map[string]interface{})
	utils.ReverStrTemplate(MonitorTemp, *res, resMap)

	err := utils.Map2Struct(resMap, mm)
	if err != nil {
		global.Log.Error("解析machine monitor: %s", err.Error())
		return nil
	}
	mm.MachineId = c.machine.Id
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
	biz.ErrIsNil(err, "获取shell文件失败")
	shellStr := string(bytes)
	shellCache[name] = shellStr
	return shellStr
}
