package mcm

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type FSInfo struct {
	MountPoint string `json:"mountPoint"`
	Used       uint64 `json:"used"`
	Free       uint64 `json:"free"`
}

type NetIntfInfo struct {
	IPv4 string `json:"ipv4"`
	IPv6 string `json:"ipv6"`
	Rx   uint64 `json:"rx"`
	Tx   uint64 `json:"tx"`
}

type MemInfo struct {
	Total     uint64 `json:"total"`
	Free      uint64 `json:"free"`
	Buffers   uint64 `json:"buffers"`
	Available uint64 `json:"available"`
	Cached    uint64 `json:"cached"`
	SwapTotal uint64 `json:"swapTotal"`
	SwapFree  uint64 `json:"swapFree"`
}

type CPUInfo struct {
	User    float32 `json:"user"`
	Nice    float32 `json:"nice"`
	System  float32 `json:"system"`
	Idle    float32 `json:"idle"`
	Iowait  float32 `json:"iowait"`
	Irq     float32 `json:"irq"`
	SoftIrq float32 `json:"softIrq"`
	Steal   float32 `json:"steal"`
	Guest   float32 `json:"guest"`
}

type Stats struct {
	Uptime       string                 `json:"uptime"`
	Hostname     string                 `json:"hostname"`
	Load1        string                 `json:"load1"`
	Load5        string                 `json:"load5"`
	Load10       string                 `json:"load10"`
	RunningProcs string                 `json:"runningProcs"`
	TotalProcs   string                 `json:"totalProcs"`
	MemInfo      MemInfo                `json:"memInfo"`
	FSInfos      []FSInfo               `json:"fSInfos"`
	NetIntf      map[string]NetIntfInfo `json:"netIntf"`
	CPU          CPUInfo                `json:"cpu"` // or []CPUInfo to get all the cpu-core's stats?
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
				stats.MemInfo.Total = val
			case "MemFree:":
				stats.MemInfo.Free = val
			case "Buffers:":
				stats.MemInfo.Buffers = val
			case "Cached:":
				stats.MemInfo.Cached = val
			case "SwapTotal:":
				stats.MemInfo.SwapTotal = val
			case "SwapFree:":
				stats.MemInfo.SwapFree = val
			case "MemAvailable:":
				stats.MemInfo.Available = val
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
