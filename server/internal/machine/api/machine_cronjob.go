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
	MachineCronJobApp application.MachineCronJob `inject:""`
}

func (m *MachineCronJob) MachineCronJobs(rc *req.Ctx) {
	cond, pageParam := ginx.BindQueryAndPage(rc.GinCtx, new(entity.MachineCronJob))

	vos := new([]*vo.MachineCronJobVO)
	pageRes, err := m.MachineCronJobApp.GetPageList(cond, pageParam, vos)
	biz.ErrIsNil(err)
	for _, mcj := range *vos {
		mcj.Running = scheduler.ExistKey(mcj.Key)
	}

	rc.ResData = pageRes
}

func (m *MachineCronJob) Save(rc *req.Ctx) {
	jobForm := new(form.MachineCronJobForm)
	mcj := ginx.BindJsonAndCopyTo[*entity.MachineCronJob](rc.GinCtx, jobForm, new(entity.MachineCronJob))
	rc.ReqParam = jobForm

	cronJobId, err := m.MachineCronJobApp.SaveMachineCronJob(rc.MetaCtx, mcj)
	biz.ErrIsNil(err)

	// 关联机器
	m.MachineCronJobApp.CronJobRelateMachines(rc.MetaCtx, cronJobId, jobForm.MachineIds)
}

func (m *MachineCronJob) Delete(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "ids")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		m.MachineCronJobApp.Delete(rc.MetaCtx, uint64(value))
	}
}

func (m *MachineCronJob) GetRelateMachineIds(rc *req.Ctx) {
	rc.ResData = m.MachineCronJobApp.GetRelateMachineIds(uint64(ginx.QueryInt(rc.GinCtx, "cronJobId", -1)))
}

func (m *MachineCronJob) GetRelateCronJobIds(rc *req.Ctx) {
	rc.ResData = m.MachineCronJobApp.GetRelateMachineIds(uint64(ginx.QueryInt(rc.GinCtx, "machineId", -1)))
}

func (m *MachineCronJob) RunCronJob(rc *req.Ctx) {
	cronJobKey := ginx.PathParam(rc.GinCtx, "key")
	biz.NotEmpty(cronJobKey, "cronJob key不能为空")
	m.MachineCronJobApp.RunCronJob(cronJobKey)
}

func (m *MachineCronJob) CronJobExecs(rc *req.Ctx) {
	cond, pageParam := ginx.BindQueryAndPage[*entity.MachineCronJobExec](rc.GinCtx, new(entity.MachineCronJobExec))
	res, err := m.MachineCronJobApp.GetExecPageList(cond, pageParam, new([]entity.MachineCronJobExec))
	biz.ErrIsNil(err)
	rc.ResData = res
}
