package api

import (
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/domain/entity"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"strconv"
	"strings"

	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/utils/collx"
)

type MachineCronJob struct {
	MachineCronJobApp application.MachineCronJob `inject:""`
	TagTreeRelateApp  tagapp.TagTreeRelate       `inject:"TagTreeRelateApp"`
}

func (m *MachineCronJob) MachineCronJobs(rc *req.Ctx) {
	cond, pageParam := req.BindQueryAndPage(rc, new(entity.MachineCronJob))

	var vos []*vo.MachineCronJobVO
	pageRes, err := m.MachineCronJobApp.GetPageList(cond, pageParam, &vos)
	biz.ErrIsNil(err)

	for _, mcj := range vos {
		mcj.Running = scheduler.ExistKey(mcj.Key)
	}

	m.TagTreeRelateApp.FillTagInfo(tagentity.TagRelateTypeMachineCronJob, collx.ArrayMap(vos, func(mvo *vo.MachineCronJobVO) tagentity.IRelateTag {
		return mvo
	})...)

	rc.ResData = pageRes
}

func (m *MachineCronJob) Save(rc *req.Ctx) {
	jobForm := new(form.MachineCronJobForm)
	mcj := req.BindJsonAndCopyTo[*entity.MachineCronJob](rc, jobForm, new(entity.MachineCronJob))
	rc.ReqParam = jobForm

	err := m.MachineCronJobApp.SaveMachineCronJob(rc.MetaCtx, &application.SaveMachineCronJobParam{
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
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		m.MachineCronJobApp.Delete(rc.MetaCtx, uint64(value))
	}
}

func (m *MachineCronJob) RunCronJob(rc *req.Ctx) {
	cronJobKey := rc.PathParam("key")
	biz.NotEmpty(cronJobKey, "cronJob key不能为空")
	m.MachineCronJobApp.RunCronJob(cronJobKey)
}

func (m *MachineCronJob) CronJobExecs(rc *req.Ctx) {
	cond, pageParam := req.BindQueryAndPage[*entity.MachineCronJobExec](rc, new(entity.MachineCronJobExec))
	res, err := m.MachineCronJobApp.GetExecPageList(cond, pageParam, new([]entity.MachineCronJobExec))
	biz.ErrIsNil(err)
	rc.ResData = res
}
