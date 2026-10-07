package application

import (
	"context"
	"mayfly-go/internal/machine/config"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/internal/machine/mcm"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/taskx"
	"strconv"
	"strings"
	"time"
)

// NoHealthHit 分诊优先级哨兵值：没有任何规则命中
const NoHealthHit int8 = -1

// HealthHit 单条阈值命中明细。
//
// 由告警侧按「已启用的机器告警规则」判定后回填：阈值真源始终在告警规则，健康总览不自带一套
// 判定，否则两处结论必然漂移
type HealthHit struct {
	RuleId    uint64  `json:"ruleId"`
	RuleName  string  `json:"ruleName"`
	Metric    string  `json:"metric"`    // 指标名（cpu_rate / mem_rate / disk_usage / status）
	Compare   string  `json:"compare"`   // 比较方式 gt / gte / lt / lte / eq / neq
	Threshold float64 `json:"threshold"` // 规则阈值
	Current   float64 `json:"current"`   // 当前值
	Priority  int8    `json:"priority"`  // 命中规则优先级 0:P0..3:P3
}

// MachineHealth 健康总览单机器视图：机器基础信息 + 最新指标（无指标点视为离线）+ 告警规则分诊结果
type MachineHealth struct {
	MachineId   uint64     `json:"machineId"`
	Name        string     `json:"name"`
	Ip          string     `json:"ip"`
	Port        int        `json:"port"`
	Code        string     `json:"code"`
	Status      int8       `json:"status"` // 1在线 0离线
	CpuUsage    float64    `json:"cpuUsage"`
	MemUsage    float64    `json:"memUsage"`
	DiskUsage   float64    `json:"diskUsage"`
	CollectTime *time.Time `json:"collectTime"`

	// Priority 命中的最差规则优先级：NoHealthHit 表示无命中；0..3 对应 P0..P3（越小越严重）
	Priority int8 `json:"priority"`

	// Triaged=false 表示有规则覆盖该机器但取不到当前值、本次未能判定，界面必须显示「未判定」而不是「正常」
	Triaged bool `json:"triaged"`

	Hits []*HealthHit `json:"hits"`
}

// MachineMetric 机器指标历史：采集快照落库、趋势区间查询、全机器健康总览、定期清理。
//
// 独立 app，不膨胀 Machine 接口（接口隔离）；采集侧通过包内 getter 调用，避免与 Machine 双向字段注入
type MachineMetric interface {
	base.App[*entity.MachineMetric]

	// SaveSnapshot 采集成功落一个在线快照点（best-effort，失败仅日志）
	SaveSnapshot(ctx context.Context, machineId uint64, stats *mcm.Stats)

	// SaveOfflineSnapshot 采集失败落一个离线点，保证趋势不断线
	SaveOfflineSnapshot(ctx context.Context, machineId uint64)

	// GetRange 查询某机器时间区间内的趋势点（按采集时间升序）
	GetRange(machineId uint64, start, end time.Time) ([]*entity.MachineMetric, error)

	// HealthOverview 全机器最新健康态（启用中的 SSH 机器 + 各自最新指标）
	HealthOverview(ctx context.Context) ([]*MachineHealth, error)

	// TimerDeleteMetric 定期清理过期指标点
	TimerDeleteMetric()
}

type machineMetricAppImpl struct {
	base.AppImpl[*entity.MachineMetric, repository.MachineMetric]

	tagApp tagapp.TagTreeService `inject:"T"`
}

var _ MachineMetric = (*machineMetricAppImpl)(nil)

func (m *machineMetricAppImpl) SaveSnapshot(ctx context.Context, machineId uint64, stats *mcm.Stats) {
	if stats == nil {
		m.SaveOfflineSnapshot(ctx, machineId)
		return
	}
	now := time.Now()
	metric := &entity.MachineMetric{
		MachineId:   machineId,
		CollectTime: now,
		CpuUsage:    cpuUsage(stats),
		MemUsage:    memUsage(stats),
		DiskUsage:   diskUsage(stats),
		Load1:       parseLoad(stats.Load1),
		Load5:       parseLoad(stats.Load5),
		Load10:      parseLoad(stats.Load10),
		NetRx:       totalNetRx(stats),
		NetTx:       totalNetTx(stats),
		Status:      1,
	}
	if err := m.Insert(ctx, metric); err != nil {
		// 落点失败绝不影响采集主流程，仅记录日志
		logx.Errorf("failed to save machine[%d] metric snapshot: %s", machineId, err.Error())
	}
}

func (m *machineMetricAppImpl) SaveOfflineSnapshot(ctx context.Context, machineId uint64) {
	metric := &entity.MachineMetric{
		MachineId:   machineId,
		CollectTime: time.Now(),
		Status:      0,
	}
	if err := m.Insert(ctx, metric); err != nil {
		logx.Errorf("failed to save machine[%d] offline metric point: %s", machineId, err.Error())
	}
}

func (m *machineMetricAppImpl) GetRange(machineId uint64, start, end time.Time) ([]*entity.MachineMetric, error) {
	qd := model.NewCond().Eq("machine_id", machineId).Ge("collect_time", start).Le("collect_time", end).OrderBy("collect_time asc")
	return m.ListByCond(qd)
}

func (m *machineMetricAppImpl) HealthOverview(ctx context.Context) ([]*MachineHealth, error) {
	// 启用中的 SSH 机器（与 TimerUpdateStats 采集口径一致）
	machines, err := GetMachineApp().ListByCond(model.NewModelCond(&entity.Machine{Status: entity.MachineStatusEnable, Protocol: entity.MachineProtocolSsh}))
	if err != nil {
		return nil, err
	}

	latest, err := m.GetRepo().LatestPerMachine()
	if err != nil {
		return nil, err
	}
	metricMap := make(map[uint64]*entity.MachineMetric, len(latest))
	for _, mt := range latest {
		metricMap[mt.MachineId] = mt
	}

	res := make([]*MachineHealth, 0, len(machines))
	for _, ma := range machines {
		// 仅返回登录账号可访问的机器，避免向低权限用户泄露全量机器清单
		if err := m.tagApp.CanAccessByCode(ctx, int8(tagentity.TagTypeMachine), ma.Code); err != nil {
			continue
		}
		item := &MachineHealth{
			MachineId: ma.Id,
			Name:      ma.Name,
			Ip:        ma.Ip,
			Port:      ma.Port,
			Code:      ma.Code,
			Status:    0,           // 无指标点视为离线
			Priority:  NoHealthHit, // 未命中；Triage 命中时下调为更好的优先级
			Triaged:   true,
		}
		if mt, ok := metricMap[ma.Id]; ok {
			item.Status = mt.Status
			item.CpuUsage = mt.CpuUsage
			item.MemUsage = mt.MemUsage
			item.DiskUsage = mt.DiskUsage
			ct := mt.CollectTime
			item.CollectTime = &ct
		}
		res = append(res, item)
	}
	return res, nil
}

func (m *machineMetricAppImpl) TimerDeleteMetric() {
	logx.Debug("start deleting expired machine metrics every day...")
	_ = taskx.BindCronTaskWithLock("machine-metric-cleanup", "@every 6h", 30*time.Minute, true, func() {
		defer gox.Recover()
		startDate := time.Now().AddDate(0, 0, -config.GetMachine().MetricSaveDays)
		// CreateModelNLD 非逻辑删除，此处为物理删除过期时序点
		if err := m.DeleteByCond(context.Background(), model.NewCond().Lt("collect_time", startDate)); err != nil {
			logx.Errorf("failed to delete expired machine metrics: %s", err.Error())
		}
	})
}

// ---- 指标口径换算：与告警评估器 MachineEvaluator 保持一致 ----

func cpuUsage(stats *mcm.Stats) float64 {
	return 100 - float64(stats.CPU.Idle)
}

func memUsage(stats *mcm.Stats) float64 {
	total := stats.MemInfo.Total
	if total == 0 {
		return 0
	}
	return float64(total-stats.MemInfo.Available) / float64(total) * 100
}

// diskUsage 根分区使用率，无根分区则取最满分区（与 extractDiskUsage 同口径）
func diskUsage(stats *mcm.Stats) float64 {
	var maxUsage float64
	for _, fs := range stats.FSInfos {
		total := fs.Used + fs.Free
		if total == 0 {
			continue
		}
		usage := float64(fs.Used) / float64(total) * 100
		if fs.MountPoint == "/" {
			return usage
		}
		if usage > maxUsage {
			maxUsage = usage
		}
	}
	return maxUsage
}

func parseLoad(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// countableNetIntf 判定该网卡的字节计数是否应计入「机器网络流量」。
//
// 排除两类，否则总量与真实外发流量不符：
//   - 回环（lo / 127.*）：本机进程互访，实测占单机总量 67%，会把纯本地流量看成流量尖峰；
//   - 容器侧虚拟接口（docker0 / veth* / br-*）：这些包同时会计入收发两端物理网卡，
//     叠加即重复计数（实测某机 eth0 44.5GB，而全网卡求和 141.5GB）。
func countableNetIntf(name string, intf mcm.NetIntfInfo) bool {
	if name == "lo" || strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "veth") || strings.HasPrefix(name, "br-") {
		return false
	}
	return !strings.HasPrefix(intf.IPv4, "127.")
}

func totalNetRx(stats *mcm.Stats) uint64 {
	var sum uint64
	for name, intf := range stats.NetIntf {
		if countableNetIntf(name, intf) {
			sum += intf.Rx
		}
	}
	return sum
}

func totalNetTx(stats *mcm.Stats) uint64 {
	var sum uint64
	for name, intf := range stats.NetIntf {
		if countableNetIntf(name, intf) {
			sum += intf.Tx
		}
	}
	return sum
}
