package api

import (
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/application/dto"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/imsg"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"strings"

	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/utils/collx"

	"github.com/may-fly/cast"
)

type MachineCronJob struct {
	machineCronJobApp application.MachineCronJob `inject:"T"`
	tagTreeRelateApp  tagapp.TagTreeRelate       `inject:"T"`
}

func (mcj *MachineCronJob) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 获取机器任务列表
		req.NewGet("", mcj.MachineCronJobs),

		req.NewPost("", mcj.Save).Log(req.NewLogSaveI(imsg.LogMachineCronJobSave)),

		req.NewDelete(":ids", mcj.Delete).Log(req.NewLogSaveI(imsg.LogMachineCronJobDelete)),

		req.NewPost("/run/:key", mcj.RunCronJob).Log(req.NewLogSaveI(imsg.LogMachineCronJobRun)),

		req.NewGet("/execs", mcj.CronJobExecs),
	}

	return req.NewConfs("machine-cronjobs", reqs[:]...)
}

func (m *MachineCronJob) MachineCronJobs(rc *req.Ctx) {
	cond, pageParam := req.BindQueryAndPage(rc, new(entity.MachineCronJob))

	var vos []*vo.MachineCronJobVO
	pageRes, err := m.machineCronJobApp.GetPageList(cond, pageParam, &vos)
	biz.ErrIsNil(err)

	for _, mcj := range vos {
		mcj.Running = scheduler.ExistKey(mcj.Key)
	}

	m.tagTreeRelateApp.FillTagInfo(tagentity.TagRelateTypeMachineCronJob, collx.ArrayMap(vos, func(mvo *vo.MachineCronJobVO) tagentity.IRelateTag {
		return mvo
	})...)

	rc.ResData = pageRes
}

func (m *MachineCronJob) Save(rc *req.Ctx) {
	jobForm := new(form.MachineCronJobForm)
	mcj := req.BindJsonAndCopyTo[*entity.MachineCronJob](rc, jobForm, new(entity.MachineCronJob))
	rc.ReqParam = jobForm

	err := m.machineCronJobApp.SaveMachineCronJob(rc.MetaCtx, &dto.SaveMachineCronJob{
		CronJob:   mcj,
		CodePaths: jobForm.CodePaths,
	})
	biz.ErrIsNil(err)
}

func (m *MachineCronJob) Delete(rc *req.Ctx) {
	idsStr := rc.PathParam("ids")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		m.machineCronJobApp.Delete(rc.MetaCtx, cast.ToUint64(v))
	}
}

func (m *MachineCronJob) RunCronJob(rc *req.Ctx) {
	cronJobKey := rc.PathParam("key")
	biz.NotEmpty(cronJobKey, "cronJob key cannot be empty")
	m.machineCronJobApp.RunCronJob(cronJobKey)
}

func (m *MachineCronJob) CronJobExecs(rc *req.Ctx) {
	cond, pageParam := req.BindQueryAndPage[*entity.MachineCronJobExec](rc, new(entity.MachineCronJobExec))
	res, err := m.machineCronJobApp.GetExecPageList(cond, pageParam, new([]entity.MachineCronJobExec))
	biz.ErrIsNil(err)
	rc.ResData = res
}
