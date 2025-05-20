package application

import (
	"context"
	"fmt"
	fileapp "mayfly-go/internal/file/application"
	"mayfly-go/internal/machine/config"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/internal/machine/imsg"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/timex"
	"time"

	"github.com/gorilla/websocket"
)

type MachineTermOp interface {
	base.App[*entity.MachineTermOp]

	// 终端连接操作
	TermConn(ctx context.Context, cli *mcm.Cli, wsConn *websocket.Conn, rows, cols int) error

	GetPageList(condition *entity.MachineTermOp, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineTermOp], error)

	// 定时删除终端文件回放记录
	TimerDeleteTermOp()
}

type machineTermOpAppImpl struct {
	base.AppImpl[*entity.MachineTermOp, repository.MachineTermOp]

	machineCmdConfApp MachineCmdConf `inject:"T"`
	fileApp           fileapp.File   `inject:"T"`
}

func (m *machineTermOpAppImpl) TermConn(ctx context.Context, cli *mcm.Cli, wsConn *websocket.Conn, rows, cols int) error {
	var recorder *mcm.Recorder
	var termOpRecord *entity.MachineTermOp
	var err error

	// 开启终端操作记录
	if cli.Info.EnableRecorder == 1 {
		now := time.Now()
		la := contextx.GetLoginAccount(ctx)

		termOpRecord = new(entity.MachineTermOp)

		termOpRecord.CreateTime = &now
		termOpRecord.Creator = la.Username
		termOpRecord.CreatorId = la.Id

		termOpRecord.MachineId = cli.Info.Id
		termOpRecord.Username = cli.Info.Username

		fileKey, wc, saveFileFunc, err := m.fileApp.NewWriter(ctx, "", fmt.Sprintf("mto_%d_%s.cast", termOpRecord.MachineId, timex.TimeNo()))
		if err != nil {
			return errorx.NewBiz("failed to create a terminal playback log file: %v", err)
		}
		defer saveFileFunc(&err)

		termOpRecord.FileKey = fileKey
		recorder = mcm.NewRecorder(wc)
	}

	createTsParam := &mcm.CreateTerminalSessionParam{
		SessionId: stringx.Rand(16),
		Cli:       cli,
		WsConn:    wsConn,
		Rows:      rows,
		Cols:      cols,
		Recorder:  recorder,
		LogCmd:    cli.Info.EnableRecorder == 1,
	}

	cmdConfs := m.machineCmdConfApp.GetCmdConfsByMachineTags(ctx, cli.Info.CodePath...)
	if len(cmdConfs) > 0 {
		createTsParam.CmdFilterFuncs = []mcm.CmdFilterFunc{func(cmd string) error {
			for _, cmdConf := range cmdConfs {
				if cmdConf.CmdRegexp.Match([]byte(cmd)) {
					return errorx.NewBizI(ctx, imsg.TerminalCmdDisable)
				}
			}
			return nil
		}}
	}

	mts, err := mcm.NewTerminalSession(createTsParam)
	if err != nil {
		return err
	}

	mts.Start()
	defer mts.Stop()

	if termOpRecord != nil {
		now := time.Now()
		termOpRecord.EndTime = &now
		termOpRecord.ExecCmds = jsonx.ToStr(mts.GetExecCmds())
		return m.Insert(ctx, termOpRecord)
	}
	return nil
}

func (m *machineTermOpAppImpl) GetPageList(condition *entity.MachineTermOp, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineTermOp], error) {
	return m.GetRepo().GetPageList(condition, pageParam)
}

func (m *machineTermOpAppImpl) TimerDeleteTermOp() {
	logx.Debug("start deleting machine terminal playback records every hour...")
	scheduler.AddFun("@every 60m", func() {
		startDate := time.Now().AddDate(0, 0, -config.GetMachine().TermOpSaveDays)
		cond := &entity.MachineTermOpQuery{
			StartCreateTime: &startDate,
		}
		termOps, err := m.GetRepo().SelectByQuery(cond)
		if err != nil {
			return
		}

		for _, termOp := range termOps {
			if err := m.DeleteTermOp(termOp); err != nil {
				logx.Warnf("failed to delete terminal playback record: %s", err.Error())
			}
		}
	})
}

// 删除终端记录即对应文件
func (m *machineTermOpAppImpl) DeleteTermOp(termOp *entity.MachineTermOp) error {
	if err := m.DeleteById(context.Background(), termOp.Id); err != nil {
		return err
	}

	return m.fileApp.Remove(context.TODO(), termOp.FileKey)
}
