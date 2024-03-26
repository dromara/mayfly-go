package api

import (
	"context"
	"fmt"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/req"
	"strconv"
	"strings"
	"time"
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

func (d *DbTransferTask) Logs(rc *req.Ctx) {
	queryCond, page := req.BindQueryAndPage[*entity.DbTransferLogQuery](rc, new(entity.DbTransferLogQuery))
	res, err := d.DbTransferTask.GetTaskLogList(queryCond, page, new([]vo.DbTransferLogListVO))
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
	start := time.Now()
	taskId := d.changeState(rc, entity.DbTransferTaskRunStateRunning)
	go d.DbTransferTask.Run(taskId, func(msg string, err error) {
		// 修改状态为停止
		if err != nil {
			logx.Error(msg, err)
		} else {
			logx.Info(fmt.Sprintf("执行迁移完成，%s, 耗时：%v", msg, time.Since(start)))
		}
		// 修改任务状态
		task := new(entity.DbTransferTask)
		task.Id = taskId
		task.RunningState = entity.DbTransferTaskRunStateStop
		biz.ErrIsNil(d.DbTransferTask.UpdateById(context.Background(), task))
	})

}

func (d *DbTransferTask) Stop(rc *req.Ctx) {
	taskId := d.changeState(rc, entity.DbTransferTaskRunStateStop)
	d.DbTransferTask.Stop(taskId)
}

func (d *DbTransferTask) changeState(rc *req.Ctx, RunningState int) uint64 {
	reqForm := &form.DbTransferTaskStatusForm{RunningState: RunningState}
	task := req.BindJsonAndCopyTo[*entity.DbTransferTask](rc, reqForm, new(entity.DbTransferTask))
	biz.ErrIsNil(d.DbTransferTask.UpdateById(rc.MetaCtx, task))
	// 记录请求日志
	rc.ReqParam = reqForm
	return task.Id
}
