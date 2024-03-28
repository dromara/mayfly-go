package api

import (
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"strconv"
	"strings"
)

type DbTransferTask struct {
	DbTransferTask application.DbTransferTask `inject:"DbTransferTaskApp"`
}

func (d *DbTransferTask) Tasks(rc *req.Ctx) {
	queryCond, page := req.BindQueryAndPage[*entity.DbTransferTaskQuery](rc, new(entity.DbTransferTaskQuery))
	res, err := d.DbTransferTask.GetPageList(queryCond, page, new([]vo.DbTransferTaskListVO))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (d *DbTransferTask) SaveTask(rc *req.Ctx) {
	reqForm := &form.DbTransferTaskForm{}
	task := req.BindJsonAndCopyTo[*entity.DbTransferTask](rc, reqForm, new(entity.DbTransferTask))

	rc.ReqParam = reqForm
	biz.ErrIsNil(d.DbTransferTask.Save(rc.MetaCtx, task))
}

func (d *DbTransferTask) DeleteTask(rc *req.Ctx) {
	taskId := rc.PathParam("taskId")
	rc.ReqParam = taskId
	ids := strings.Split(taskId, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		biz.ErrIsNil(d.DbTransferTask.Delete(rc.MetaCtx, uint64(value)))
	}
}

func (d *DbTransferTask) Run(rc *req.Ctx) {
	go d.DbTransferTask.Run(rc.MetaCtx, uint64(rc.PathParamInt("taskId")))
}

func (d *DbTransferTask) Stop(rc *req.Ctx) {
	biz.ErrIsNil(d.DbTransferTask.Stop(rc.MetaCtx, uint64(rc.PathParamInt("taskId"))))
}
