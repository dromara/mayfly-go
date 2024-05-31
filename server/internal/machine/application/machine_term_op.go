package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/machine/config"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/stringx"
	"os"
	"path"
	"time"

	"github.com/gorilla/websocket"
)

type MachineTermOp interface {
	base.App[*entity.MachineTermOp]

	// 终端连接操作
	TermConn(ctx context.Context, cli *mcm.Cli, wsConn *websocket.Conn, rows, cols int) error

	GetPageList(condition *entity.MachineTermOp, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// 定时删除终端文件回放记录
	TimerDeleteTermOp()
}

type machineTermOpAppImpl struct {
	base.AppImpl[*entity.MachineTermOp, repository.MachineTermOp]

	machineCmdConfApp MachineCmdConf `inject:"MachineCmdConfApp"`
}

// 注入MachineTermOpRepo
func (m *machineTermOpAppImpl) InjectMachineTermOpRepo(repo repository.MachineTermOp) {
	m.Repo = repo
}

func (m *machineTermOpAppImpl) TermConn(ctx context.Context, cli *mcm.Cli, wsConn *websocket.Conn, rows, cols int) error {
	var recorder *mcm.Recorder
	var termOpRecord *entity.MachineTermOp

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

		// 回放文件路径为: 基础配置路径/机器编号/操作日期(202301)/day/hour/randstr.cast
		recRelPath := path.Join(cli.Info.Code, now.Format("200601"), fmt.Sprintf("%d", now.Day()), fmt.Sprintf("%d", now.Hour()))
		// 文件绝对路径
		recAbsPath := path.Join(config.GetMachine().TerminalRecPath, recRelPath)
		os.MkdirAll(recAbsPath, 0766)
		filename := fmt.Sprintf("%s.cast", stringx.RandByChars(18, stringx.LowerChars))
		f, err := os.OpenFile(path.Join(recAbsPath, filename), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0766)
		if err != nil {
			return errorx.NewBiz("创建终端回放记录文件失败: %s", err.Error())
		}
		defer f.Close()

		termOpRecord.RecordFilePath = path.Join(recRelPath, filename)
		recorder = mcm.NewRecorder(f)
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
					return errorx.NewBiz("该命令已被禁用...")
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

func (m *machineTermOpAppImpl) GetPageList(condition *entity.MachineTermOp, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return m.GetRepo().GetPageList(condition, pageParam, toEntity)
}

func (m *machineTermOpAppImpl) TimerDeleteTermOp() {
	logx.Debug("开始定时删除机器终端回放记录...")
	scheduler.AddFun("@every 60m", func() {
		startDate := time.Now().AddDate(0, 0, -config.GetMachine().TermOpSaveDays)
		cond := &entity.MachineTermOpQuery{
			StartCreateTime: &startDate,
		}
		termOps, err := m.GetRepo().SelectByQuery(cond)
		if err != nil {
			return
		}

		basePath := config.GetMachine().TerminalRecPath
		for _, termOp := range termOps {
			if err := m.DeleteTermOp(basePath, termOp); err != nil {
				logx.Warnf("删除终端操作记录失败: %s", err.Error())
			}
		}
	})
}

// 删除终端记录即对应文件
func (m *machineTermOpAppImpl) DeleteTermOp(basePath string, termOp *entity.MachineTermOp) error {
	if err := m.DeleteById(context.Background(), termOp.Id); err != nil {
		return err
	}

	return os.Remove(path.Join(basePath, termOp.RecordFilePath))
}
