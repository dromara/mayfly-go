package api

import (
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/domain/entity"
	"strconv"
	"strings"

	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/scheduler"
)

type MachineCronJob struct {
	MachineCronJobApp application.MachineCronJob
}

func (m *MachineCronJob) MachineCronJobs(rc *req.Ctx) {
	cond, pageParam := ginx.BindQueryAndPage(rc.GinCtx, new(entity.MachineCronJob))

	vos := new([]*vo.MachineCronJobVO)
	pr := m.MachineCronJobApp.GetPageList(cond, pageParam, vos)
	for _, mcj := range *vos {
		mcj.Running = scheduler.ExistKey(mcj.Key)
	}

	rc.ResData = pr
}

func (m *MachineCronJob) Save(rc *req.Ctx) {
	jobForm := new(form.MachineCronJobForm)
	mcj := ginx.BindJsonAndCopyTo[*entity.MachineCronJob](rc.GinCtx, jobForm, new(entity.MachineCronJob))
	rc.ReqParam = jobForm
	mcj.SetBaseInfo(rc.LoginAccount)
	cronJobId := m.MachineCronJobApp.Save(mcj)

	// 关联机器
	m.MachineCronJobApp.CronJobRelateMachines(cronJobId, jobForm.MachineIds, rc.LoginAccount)
}

func (m *MachineCronJob) Delete(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "ids")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		m.MachineCronJobApp.Delete(uint64(value))
	}
}

func (m *MachineCronJob) GetRelateMachineIds(rc *req.Ctx) {
	rc.ResData = m.MachineCronJobApp.GetRelateMachineIds(uint64(ginx.QueryInt(rc.GinCtx, "cronJobId", -1)))
}

func (m *MachineCronJob) GetRelateCronJobIds(rc *req.Ctx) {
	rc.ResData = m.MachineCronJobApp.GetRelateMachineIds(uint64(ginx.QueryInt(rc.GinCtx, "machineId", -1)))
}

func (m *MachineCronJob) CronJobExecs(rc *req.Ctx) {
	cond, pageParam := ginx.BindQueryAndPage[*entity.MachineCronJobExec](rc.GinCtx, new(entity.MachineCronJobExec))
	rc.ResData = m.MachineCronJobApp.GetExecPageList(cond, pageParam, new([]entity.MachineCronJobExec))
}
