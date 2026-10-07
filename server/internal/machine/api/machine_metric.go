package api

import (
	alertapp "mayfly-go/internal/alert/application"
	"mayfly-go/internal/machine/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"time"

	"github.com/spf13/cast"
)

// MachineMetric 机器指标历史与全机器健康总览查询接口（只读监控视图）
type MachineMetric struct {
	metricApp     application.MachineMetric `inject:"T"`
	machineTriage alertapp.MachineTriage    `inject:"T"`
}

func (mm *MachineMetric) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 单机指标趋势点（与 stats 同为查看类，仅需 token）
		req.NewGet(":machineId/metrics", mm.Metrics),

		// 全机器健康总览（静态段，与 :machineId 动态段不同层级无冲突）
		req.NewGet("health-overview", mm.HealthOverview),
	}

	return req.NewConfs("machines", reqs[:]...)
}

// Metrics 查询某机器时间区间内的指标趋势点，start/end 为 unix 毫秒；缺省取最近 24 小时
func (mm *MachineMetric) Metrics(rc *req.Ctx) {
	machineId := cast.ToUint64(rc.PathParam("machineId"))

	end := cast.ToInt64(rc.Query("end"))
	if end <= 0 {
		end = time.Now().UnixMilli()
	}
	start := cast.ToInt64(rc.Query("start"))
	if start <= 0 || start >= end {
		start = end - 24*int64(time.Hour/time.Millisecond)
	}

	list, err := mm.metricApp.GetRange(machineId, time.UnixMilli(start), time.UnixMilli(end))
	biz.ErrIsNil(err)
	rc.ResData = list
}

// HealthOverview 全机器最新健康态（按登录账号可访问范围），并附带告警规则分诊结果
func (mm *MachineMetric) HealthOverview(rc *req.Ctx) {
	res, err := mm.metricApp.HealthOverview(rc.MetaCtx)
	biz.ErrIsNil(err)
	// 阈值真源在告警规则：由告警侧按已启用规则判定越线与优先级，机器侧不自带第二套判定
	biz.ErrIsNil(mm.machineTriage.TriageMachineHealth(rc.MetaCtx, res))
	rc.ResData = res
}
