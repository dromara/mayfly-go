package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"mayfly-go/internal/machine/application/dto"
	"mayfly-go/internal/machine/imsg"
	"mayfly-go/internal/machine/mcm"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"

	"gorm.io/gorm"
)

// 批量执行限额集中收口：如未来需要可配，从这里升级为系统配置扩展点即可，不散落调用点
const (
	maxBatchMachines  = 50                // 单次批量执行的最大机器数
	batchConcurrency  = 10                // 并发执行的机器数上限
	batchCmdTimeout   = 120 * time.Second // 单台命令执行超时
	batchTotalTimeout = 5 * time.Minute   // 整体超时：到期后未执行的机器直接按超时记账
)

// MachineBatchExec 多机批量命令执行。
//
// 与单机 Machine 接口隔离的独立能力：一次提交，逐台并发执行并聚合结果，
// 任一台的失败/超时都不影响他台
type MachineBatchExec interface {
	RunBatchCmd(ctx context.Context, machineIds []uint64, cmd string) ([]*dto.BatchCmdResult, error)
}

type machineBatchExecImpl struct {
	machineApp Machine               `inject:"T"`
	tagApp     tagapp.TagTreeService `inject:"T"`
}

var _ MachineBatchExec = (*machineBatchExecImpl)(nil)

// RunBatchCmd 在多台机器上执行同一条命令，逐台聚合结果。
//
// 参数校验返回错误由调用方处置（入参问题是整体性的）；单台机器的失败记入该台结果
func (b *machineBatchExecImpl) RunBatchCmd(ctx context.Context, machineIds []uint64, cmd string) ([]*dto.BatchCmdResult, error) {
	if len(machineIds) == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrBatchMachineIdsEmpty)
	}
	if strings.TrimSpace(cmd) == "" {
		return nil, errorx.NewBizI(ctx, imsg.ErrBatchCmdEmpty)
	}
	if len(machineIds) > maxBatchMachines {
		return nil, errorx.NewBizI(ctx, imsg.ErrBatchExecMaxMachines, "max", maxBatchMachines)
	}

	results := runBatchConcurrently(ctx, machineIds, batchTotalTimeout,
		func(rctx context.Context, id uint64) *dto.BatchCmdResult { return b.runOne(rctx, id, cmd) },
		func(id uint64, err error) *dto.BatchCmdResult {
			return &dto.BatchCmdResult{MachineId: id, Error: err.Error()}
		},
	)
	return results, nil
}

// runOne 在单台机器上执行命令：任一步骤失败都记入该台结果并立即返回，绝不影响他台
func (b *machineBatchExecImpl) runOne(ctx context.Context, machineId uint64, cmd string) *dto.BatchCmdResult {
	res := &dto.BatchCmdResult{MachineId: machineId}
	start := time.Now()
	defer func() { res.CostMs = time.Since(start).Milliseconds() }()

	// 整体超时后不再发起新执行，避免不可达主机把请求拖成超长请求
	if ctx.Err() != nil {
		res.Timeout = true
		res.Error = i18n.TC(ctx, imsg.BatchExecTotalTimeout)
		return res
	}

	// 先取机器（存在性与展示信息），越权者不触发到目标机的任何连接
	me, err := b.machineApp.GetById(machineId)
	if err != nil {
		// GORM 原始错误（如 record not found）不能直抬给用户：界面会多出无上下文的英文短语
		if errors.Is(err, gorm.ErrRecordNotFound) {
			res.Error = i18n.TC(ctx, imsg.ErrMachineNotFoundById, "machineId", machineId)
		} else {
			res.Error = err.Error()
		}
		return res
	}
	res.Name = me.Name
	res.Ip = me.Ip
	res.Port = me.Port

	// 批量执行天然跨资源，必须逐台校验操作者对该机器的访问权
	if err := b.tagApp.CanAccessByCode(ctx, int8(tagentity.TagTypeMachine), me.Code); err != nil {
		res.Error = err.Error()
		return res
	}

	// 连接（池化复用，与终端/单机命令执行同一连接池）
	cli, err := b.machineApp.GetCli(ctx, machineId)
	if err != nil {
		res.Error = i18n.TC(ctx, imsg.ErrBatchConnFailed, "reason", err.Error())
		return res
	}
	res.AuthCertName = cli.Info.AuthCertName
	res.Username = cli.Info.Username

	// 命令治理：与终端/API/AI 同一入口（禁止级拒绝该台，提醒级随结果回传）
	notice, cmdErr := CheckMachineCmd(ctx, cli.Info.CodePath, cmd)
	if cmdErr != nil {
		res.Error = cmdErr.Error()
		return res
	}
	res.PolicyNotice = notice

	output, runErr := cli.RunWithTimeout(batchCmdTimeout, cmd)
	res.Output = output
	if runErr != nil {
		res.Error = runErr.Error()
		res.Timeout = errors.Is(runErr, mcm.ErrRunTimeout)
		return res
	}
	res.Success = true
	return res
}
