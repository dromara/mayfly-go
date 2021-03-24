package machine

import (
	"io/ioutil"
	"mayfly-go/base/biz"
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
