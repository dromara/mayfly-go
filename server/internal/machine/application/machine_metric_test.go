package application

import (
	"math"
	"testing"

	"mayfly-go/internal/machine/mcm"
)

func almostEqual(t *testing.T, got, want float64, what string) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("%s = %v, want %v", what, got, want)
	}
}

func TestCpuUsage(t *testing.T) {
	// 使用率 = 100 - 空闲%，与 top 的读数口径一致
	almostEqual(t, cpuUsage(&mcm.Stats{CPU: mcm.CPUInfo{Idle: 90.5}}), 9.5, "cpuUsage(100-90.5)")
	almostEqual(t, cpuUsage(&mcm.Stats{CPU: mcm.CPUInfo{Idle: 100}}), 0, "cpuUsage(全空闲)")
}

func TestMemUsage(t *testing.T) {
	stats := &mcm.Stats{MemInfo: mcm.MemInfo{Total: 1691451392, Available: 420687872}}
	// 用 available（而非 free）才算「真正可用」，否则 page cache 会被当成已用
	almostEqual(t, memUsage(stats), 100*float64(1691451392-420687872)/float64(1691451392), "memUsage")
	almostEqual(t, memUsage(&mcm.Stats{MemInfo: mcm.MemInfo{Total: 0}}), 0, "memUsage(总量为0不除零)")
}

func TestDiskUsage(t *testing.T) {
	rootFirst := &mcm.Stats{FSInfos: []mcm.FSInfo{
		{MountPoint: "/", Used: 20, Free: 80},
		{MountPoint: "/data", Used: 90, Free: 10},
	}}
	// 有根分区就取根分区：磁盘使用率代表系统盘，不被数据盘绑架
	almostEqual(t, diskUsage(rootFirst), 20, "diskUsage(有根分区取根)")

	rootLast := &mcm.Stats{FSInfos: []mcm.FSInfo{
		{MountPoint: "/data", Used: 90, Free: 10},
		{MountPoint: "/", Used: 30, Free: 70},
	}}
	almostEqual(t, diskUsage(rootLast), 30, "diskUsage(根分区在后仍取根)")

	noRoot := &mcm.Stats{FSInfos: []mcm.FSInfo{
		{MountPoint: "/data", Used: 90, Free: 10},
		{MountPoint: "/bak", Used: 10, Free: 90},
	}}
	almostEqual(t, diskUsage(noRoot), 90, "diskUsage(无根分区取最满)")

	almostEqual(t, diskUsage(&mcm.Stats{FSInfos: []mcm.FSInfo{{MountPoint: "/", Used: 0, Free: 0}}}), 0, "diskUsage(容量0不除零)")
	almostEqual(t, diskUsage(&mcm.Stats{}), 0, "diskUsage(无分区信息)")
}

// 网络聚合口径：回环与容器侧虚拟接口不得计入，否则总量虚高且把本地流量当流量尖峰
func TestTotalNetExcludesLoopbackAndVirtual(t *testing.T) {
	stats := &mcm.Stats{NetIntf: map[string]mcm.NetIntfInfo{
		"eth0":        {IPv4: "172.28.33.195/20", Rx: 1000, Tx: 2000},
		"lo":          {IPv4: "127.0.0.1/8", Rx: 9000, Tx: 9000},
		"docker0":     {IPv4: "172.17.0.1/16", Rx: 500, Tx: 600},
		"veth861d1b0": {IPv4: "", Rx: 700, Tx: 800},
		"br-9f2c1":    {IPv4: "10.0.0.1/16", Rx: 100, Tx: 100},
	}}
	if got := totalNetRx(stats); got != 1000 {
		t.Fatalf("totalNetRx = %d, want 1000（仅物理网卡）", got)
	}
	if got := totalNetTx(stats); got != 2000 {
		t.Fatalf("totalNetTx = %d, want 2000（仅物理网卡）", got)
	}

	// 无名回环（容器/自定义网卡把回环地址挂在别的名字下）也要排除
	if got := totalNetRx(&mcm.Stats{NetIntf: map[string]mcm.NetIntfInfo{"bond0": {IPv4: "127.0.0.5/8", Rx: 50}}}); got != 0 {
		t.Fatalf("totalNetRx(127.* 地址) = %d, want 0", got)
	}
}

func TestParseLoad(t *testing.T) {
	// 采集侧 load 是字符串（/proc/loadavg 原文），非法值必须归零而不是中断落库
	almostEqual(t, parseLoad("0.09"), 0.09, "parseLoad(0.09)")
	almostEqual(t, parseLoad(""), 0, "parseLoad(空)")
	almostEqual(t, parseLoad("n/a"), 0, "parseLoad(非法)")
}
