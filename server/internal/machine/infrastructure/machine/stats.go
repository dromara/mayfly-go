package machine

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type FSInfo struct {
	MountPoint string
	Used       uint64
	Free       uint64
}

type NetIntfInfo struct {
	IPv4 string
	IPv6 string
	Rx   uint64
	Tx   uint64
}

type CPUInfo struct {
	User    float32
	Nice    float32
	System  float32
	Idle    float32
	Iowait  float32
	Irq     float32
	SoftIrq float32
	Steal   float32
	Guest   float32
}

type Stats struct {
	Uptime       string
	Hostname     string
	Load1        string
	Load5        string
	Load10       string
	RunningProcs string
	TotalProcs   string
	MemTotal     uint64
	MemFree      uint64
	MemBuffers   uint64
	MemAvailable uint64
	MemCached    uint64
	SwapTotal    uint64
	SwapFree     uint64
	FSInfos      []FSInfo
	NetIntf      map[string]NetIntfInfo
	CPU          CPUInfo // or []CPUInfo to get all the cpu-core's stats?
}

const StatsShell = `
cat /proc/uptime
echo '-----'
/bin/hostname -f
echo '-----'
cat /proc/loadavg
echo '-----'
cat /proc/meminfo
echo '-----'
df -B1
echo '-----'
/sbin/ip -o addr
echo '-----'
/bin/cat /proc/net/dev
echo '-----'
top -b -n 1 | grep Cpu
`

func (c *Cli) GetAllStats() *Stats {
	res, _ := c.Run(StatsShell)
	infos := strings.Split(*res, "-----")
	stats := new(Stats)
	getUptime(infos[0], stats)
	getHostname(infos[1], stats)
	getLoad(infos[2], stats)
	getMemInfo(infos[3], stats)
	getFSInfo(infos[4], stats)
	getInterfaces(infos[5], stats)
	getInterfaceInfo(infos[6], stats)
	getCPU(infos[7], stats)
	return stats
}

func getUptime(uptime string, stats *Stats) (err error) {
	parts := strings.Fields(uptime)
	if len(parts) == 2 {
		var upsecs float64
		upsecs, err = strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return
		}
		stats.Uptime = fmtUptime(time.Duration(upsecs * 1e9))
	}
	return
}

func fmtUptime(dur time.Duration) string {
	dur = dur - (dur % time.Second)
	var days int
	for dur.Hours() > 24.0 {
		days++
		dur -= 24 * time.Hour
	}
	s1 := dur.String()
	s2 := ""
	if days > 0 {
		s2 = fmt.Sprintf("%dd ", days)
	}
	for _, ch := range s1 {
		s2 += string(ch)
		if ch == 'h' || ch == 'm' {
			s2 += " "
		}
	}
	return s2
}

func getHostname(hostname string, stats *Stats) (err error) {
	stats.Hostname = strings.TrimSpace(hostname)
	return
}

func getLoad(loadInfo string, stats *Stats) (err error) {
	parts := strings.Fields(loadInfo)
	if len(parts) == 5 {
		stats.Load1 = parts[0]
		stats.Load5 = parts[1]
		stats.Load10 = parts[2]
		if i := strings.Index(parts[3], "/"); i != -1 {
			stats.RunningProcs = parts[3][0:i]
			if i+1 < len(parts[3]) {
				stats.TotalProcs = parts[3][i+1:]
			}
		}
	}

	return
}

func getMemInfo(memInfo string, stats *Stats) (err error) {
	// "/bin/cat /proc/meminfo"
	scanner := bufio.NewScanner(strings.NewReader(memInfo))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 3 {
			val, err := strconv.ParseUint(parts[1], 10, 64)
			if err != nil {
				continue
			}
			val *= 1024
			switch parts[0] {
			case "MemTotal:":
				stats.MemTotal = val
			case "MemFree:":
				stats.MemFree = val
			case "Buffers:":
				stats.MemBuffers = val
			case "Cached:":
				stats.MemCached = val
			case "SwapTotal:":
				stats.SwapTotal = val
			case "SwapFree:":
				stats.SwapFree = val
			case "MemAvailable:":
				stats.MemAvailable = val
			}
		}
	}

	return
}

func getFSInfo(fsInfo string, stats *Stats) (err error) {
	// "/bin/df -B1"
	scanner := bufio.NewScanner(strings.NewReader(fsInfo))
	flag := 0
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		n := len(parts)
		dev := n > 0 && strings.Index(parts[0], "/dev/") == 0
		if n == 1 && dev {
			flag = 1
		} else if (n == 5 && flag == 1) || (n == 6 && dev) {
			i := flag
			flag = 0
			used, err := strconv.ParseUint(parts[2-i], 10, 64)
			if err != nil {
				continue
			}
			free, err := strconv.ParseUint(parts[3-i], 10, 64)
			if err != nil {
				continue
			}
			stats.FSInfos = append(stats.FSInfos, FSInfo{
				parts[5-i], used, free,
			})
		}
	}

	return
}

func getInterfaces(iInfo string, stats *Stats) (err error) {
	// "/sbin/ip -o addr"
	if stats.NetIntf == nil {
		stats.NetIntf = make(map[string]NetIntfInfo)
	}

	scanner := bufio.NewScanner(strings.NewReader(iInfo))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 4 && (parts[2] == "inet" || parts[2] == "inet6") {
			ipv4 := parts[2] == "inet"
			intfname := parts[1]
			if info, ok := stats.NetIntf[intfname]; ok {
				if ipv4 {
					info.IPv4 = parts[3]
				} else {
					info.IPv6 = parts[3]
				}
				stats.NetIntf[intfname] = info
			} else {
				info := NetIntfInfo{}
				if ipv4 {
					info.IPv4 = parts[3]
				} else {
					info.IPv6 = parts[3]
				}
				stats.NetIntf[intfname] = info
			}
		}
	}

	return
}

func getInterfaceInfo(iInfo string, stats *Stats) (err error) {
	// /bin/cat /proc/net/dev
	if stats.NetIntf == nil {
		return
	} // should have been here already

	scanner := bufio.NewScanner(strings.NewReader(iInfo))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 17 {
			intf := strings.TrimSpace(parts[0])
			intf = strings.TrimSuffix(intf, ":")
			if info, ok := stats.NetIntf[intf]; ok {
				rx, err := strconv.ParseUint(parts[1], 10, 64)
				if err != nil {
					continue
				}
				tx, err := strconv.ParseUint(parts[9], 10, 64)
				if err != nil {
					continue
				}
				info.Rx = rx
				info.Tx = tx
				stats.NetIntf[intf] = info
			}
		}
	}

	return
}

func getCPU(cpuInfo string, stats *Stats) (err error) {
	// %Cpu(s):  6.1 us,  3.0 sy,  0.0 ni, 90.9 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st
	value := strings.Split(cpuInfo, ":")[1]
	values := strings.Split(value, ",")

	separator := " "
	// 兼容旧版本使用%的情况
	if strings.Contains(values[0], "%") {
		separator = "%"
	}
	us, _ := strconv.ParseFloat(strings.Split(strings.TrimSpace(values[0]), separator)[0], 32)
	stats.CPU.User = float32(us)

	sy, _ := strconv.ParseFloat(strings.Split(strings.TrimSpace(values[1]), separator)[0], 32)
	stats.CPU.System = float32(sy)

	id, _ := strconv.ParseFloat(strings.Split(strings.TrimSpace(values[3]), separator)[0], 32)
	stats.CPU.Idle = float32(id)

	wa, _ := strconv.ParseFloat(strings.Split(strings.TrimSpace(values[4]), separator)[0], 32)
	stats.CPU.Iowait = float32(wa)

	return nil
}
