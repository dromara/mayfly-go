package machine

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/utils"
	"strconv"
	"strings"
)

type SystemVersion struct {
	Version string
}

func (c *Cli) GetSystemVersion() *SystemVersion {
	res, _ := c.Run("cat /etc/redhat-release")
	return &SystemVersion{
		Version: *res,
	}
}

//top - 17:14:07 up 5 days,  6:30,  2 users,  load average: 0.03, 0.04, 0.05
//Tasks: 101 total,   1 running, 100 sleeping,   0 stopped,   0 zombie
//%Cpu(s):  6.2 us,  0.0 sy,  0.0 ni, 93.8 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st
//KiB Mem :  1882012 total,    73892 free,   770360 used,  1037760 buff/cache
//KiB Swap:        0 total,        0 free,        0 used.   933492 avail Mem
type Top struct {
	Time string `json:"time"`
	// 从本次开机到现在经过的时间
	Up string `json:"up"`
	// 当前有几个用户登录到该机器
	NowUsers int `json:"nowUsers"`
	// load average: 0.03, 0.04, 0.05 (系统1分钟、5分钟、15分钟内的平均负载值)
	OneMinLoadavg     float32 `json:"oneMinLoadavg"`
	FiveMinLoadavg    float32 `json:"fiveMinLoadavg"`
	FifteenMinLoadavg float32 `json:"fifteenMinLoadavg"`
	// 进程总数
	TotalTask int `json:"totalTask"`
	// 正在运行的进程数，对应状态TASK_RUNNING
	RunningTask  int `json:"runningTask"`
	SleepingTask int `json:"sleepingTask"`
	StoppedTask  int `json:"stoppedTask"`
	ZombieTask   int `json:"zombieTask"`
	// 进程在用户空间（user）消耗的CPU时间占比，不包含调整过优先级的进程
	CpuUs float32 `json:"cpuUs"`
	// 进程在内核空间（system）消耗的CPU时间占比
	CpuSy float32 `json:"cpuSy"`
	// 调整过用户态优先级的（niced）进程的CPU时间占比
	CpuNi float32 `json:"cpuNi"`
	// 空闲的（idle）CPU时间占比
	CpuId float32 `json:"cpuId"`
	// 等待（wait）I/O完成的CPU时间占比
	CpuWa float32 `json:"cpuWa"`
	// 处理硬中断（hardware interrupt）的CPU时间占比
	CpuHi float32 `json:"cpuHi"`
	// 处理硬中断（hardware interrupt）的CPU时间占比
	CpuSi float32 `json:"cpuSi"`
	// 当Linux系统是在虚拟机中运行时，等待CPU资源的时间（steal time）占比
	CpuSt float32 `json:"cpuSt"`

	TotalMem int `json:"totalMem"`
	FreeMem  int `json:"freeMem"`
	UsedMem  int `json:"usedMem"`
	CacheMem int `json:"cacheMem"`

	TotalSwap int `json:"totalSwap"`
	FreeSwap  int `json:"freeSwap"`
	UsedSwap  int `json:"usedSwap"`
	AvailMem  int `json:"availMem"`
}

func (c *Cli) GetTop() *Top {
	res, _ := c.Run("top -b -n 1 | head -5")
	topTemp := "top - {upAndUsers},  load average: {loadavg}\n" +
		"Tasks:{totalTask} total,{runningTask} running,{sleepingTask} sleeping,{stoppedTask} stopped,{zombieTask} zombie\n" +
		"%Cpu(s):{cpuUs} us,{cpuSy} sy,{cpuNi} ni,{cpuId} id,{cpuWa} wa,{cpuHi} hi,{cpuSi} si,{cpuSt} st\n" +
		"KiB Mem :{totalMem} total,{freeMem} free,{usedMem} used,{cacheMem} buff/cache\n" +
		"KiB Swap:{totalSwap} total,{freeSwap} free,{usedSwap} used. {availMem} avail Mem \n"
	resMap := make(map[string]interface{})
	utils.ReverStrTemplate(topTemp, *res, resMap)

	//17:14:07 up 5 days,  6:30,  2
	timeUpAndUserStr := resMap["upAndUsers"].(string)
	timeUpAndUser := strings.Split(timeUpAndUserStr, "up")
	time := utils.StrTrim(timeUpAndUser[0])
	upAndUsers := strings.Split(timeUpAndUser[1], ",")
	up := utils.StrTrim(upAndUsers[0]) + upAndUsers[1]
	users, _ := strconv.Atoi(utils.StrTrim(strings.Split(utils.StrTrim(upAndUsers[2]), " ")[0]))
	// 0.03, 0.04, 0.05
	loadavgs := strings.Split(resMap["loadavg"].(string), ",")
	oneMinLa, _ := strconv.ParseFloat(loadavgs[0], 32)
	fiveMinLa, _ := strconv.ParseFloat(utils.StrTrim(loadavgs[1]), 32)
	fifMinLa, _ := strconv.ParseFloat(utils.StrTrim(loadavgs[2]), 32)

	top := &Top{Time: time, Up: up, NowUsers: users, OneMinLoadavg: float32(oneMinLa), FiveMinLoadavg: float32(fiveMinLa), FifteenMinLoadavg: float32(fifMinLa)}
	err := utils.Map2Struct(resMap, top)
	biz.ErrIsNil(err, "解析top出错")
	return top
}

type Status struct {
	// 系统版本
	SysVersion SystemVersion
	// top信息
	Top Top
}
