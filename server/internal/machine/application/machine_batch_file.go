package application

import (
	"context"
	"errors"
	fileapp "mayfly-go/internal/file/application"
	"mayfly-go/internal/machine/application/dto"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/imsg"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// BatchFileResult 单台机器的批量文件分发结果。单台失败不影响他台，逐台聚合返回
type BatchFileResult struct {
	MachineId    uint64 `json:"machineId"`
	Name         string `json:"name"`
	Ip           string `json:"ip"`
	Port         int    `json:"port"`
	AuthCertName string `json:"authCertName"`
	Success      bool   `json:"success"`
	Bytes        int64  `json:"bytes"`
	Error        string `json:"error"`
	CostMs       int64  `json:"costMs"`
}

// MachineBatchFile 多机批量文件分发。独立 app，接口隔离（不塞进命令执行的 MachineBatchExec）。
//
// 与批量命令执行共用 runBatchConcurrently 调度；文件先落到平台文件服务（fileKey），
// 逐台从文件服务重新打开读流分发（io.Reader 单次消费，绝不能多机共用同一 reader）
type MachineBatchFile interface {
	// DispatchFile 把文件服务中的文件（fileKey）分发到多台机器的目标目录
	DispatchFile(ctx context.Context, machineIds []uint64, fileKey, remotePath string) ([]*BatchFileResult, error)
}

type machineBatchFileAppImpl struct {
	machineApp     Machine               `inject:"T"`
	machineFileApp MachineFile           `inject:"T"`
	tagApp         tagapp.TagTreeService `inject:"T"`
	fileApp        fileapp.File          `inject:"T"`
}

var _ MachineBatchFile = (*machineBatchFileAppImpl)(nil)

// batchRemotePathPattern 目标目录允许的字符集：绝对路径，可含 Unicode 字母/数字与空格及 . _ / @ + -
// （中文目录名属合法场景）；拒绝 ; | & $ ` < > ( ) * ? ' " \\ 等 shell/通配元字符与 .. 上跳
var batchRemotePathPattern = regexp.MustCompile(`^/[\p{L}\p{N} ._/@+-]*$`)

// validateRemotePath 校验目标目录。
//
// 不校验时 SFTP 会把形如 `/tmp; touch /x` 的路径当字面量拼接到建目录/建文件，
// 既不报错也无人知道文件最终落在哪，界面却回「分发成功」：虚假成功比报错更难排查
func validateRemotePath(ctx context.Context, remotePath string) error {
	if strings.Contains(remotePath, "..") || !batchRemotePathPattern.MatchString(remotePath) {
		return errorx.NewBizI(ctx, imsg.ErrBatchRemotePathInvalid, "path", remotePath)
	}
	return nil
}

func (b *machineBatchFileAppImpl) DispatchFile(ctx context.Context, machineIds []uint64, fileKey, remotePath string) ([]*BatchFileResult, error) {
	if len(machineIds) == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrBatchMachineIdsEmpty)
	}
	if fileKey == "" {
		return nil, errorx.NewBizI(ctx, imsg.ErrBatchFileKeyEmpty)
	}
	if remotePath == "" {
		return nil, errorx.NewBizI(ctx, imsg.ErrBatchRemotePathEmpty)
	}
	if err := validateRemotePath(ctx, remotePath); err != nil {
		return nil, err
	}
	if len(machineIds) > maxBatchMachines {
		return nil, errorx.NewBizI(ctx, imsg.ErrBatchFileMaxMachines, "max", maxBatchMachines)
	}

	results := runBatchConcurrently(ctx, machineIds, batchTotalTimeout,
		func(rctx context.Context, id uint64) *BatchFileResult {
			return b.dispatchOne(rctx, id, fileKey, remotePath)
		},
		func(id uint64, err error) *BatchFileResult {
			return &BatchFileResult{MachineId: id, Error: err.Error()}
		},
	)

	// 分发完成后清理中转文件（用户经文件服务上传，仅用于本次分发）
	if rmErr := b.fileApp.Remove(context.Background(), fileKey); rmErr != nil {
		// 清理失败不影响分发结果，仅日志
		logx.Warnf("failed to cleanup batch dispatch temp file [%s]: %s", fileKey, rmErr.Error())
	}
	return results, nil
}

// dispatchOne 分发到单台机器：任一步骤失败都记入该台结果并立即返回，绝不影响他台
func (b *machineBatchFileAppImpl) dispatchOne(ctx context.Context, machineId uint64, fileKey, remotePath string) *BatchFileResult {
	res := &BatchFileResult{MachineId: machineId}
	start := time.Now()
	defer func() { res.CostMs = time.Since(start).Milliseconds() }()

	// 整体超时后不再发起新分发
	if ctx.Err() != nil {
		res.Error = i18n.TC(ctx, imsg.BatchDispatchTotalTimeout)
		return res
	}

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

	// 仅支持 SSH 机器（SFTP 分发）
	if me.Protocol != entity.MachineProtocolSsh {
		res.Error = i18n.TC(ctx, imsg.BatchDispatchOnlySsh)
		return res
	}

	// 逐台访问权校验：与批量执行同一资源级隔离
	if err := b.tagApp.CanAccessByCode(ctx, int8(tagentity.TagTypeMachine), me.Code); err != nil {
		res.Error = err.Error()
		return res
	}

	// 取该机器默认授权凭证（复用连接池，同时校验机器可用）
	cli, err := b.machineApp.GetCli(ctx, machineId)
	if err != nil {
		res.Error = i18n.TC(ctx, imsg.ErrBatchConnFailed, "reason", err.Error())
		return res
	}
	res.AuthCertName = cli.Info.AuthCertName

	// 关键：逐台从文件服务重新打开读流（reader 单次消费）
	filename, reader, err := b.fileApp.GetReader(ctx, fileKey)
	if err != nil {
		res.Error = i18n.TC(ctx, imsg.BatchDispatchOpenSourceFailed, "reason", err.Error())
		return res
	}
	defer reader.Close()

	opParam := &dto.MachineFileOp{
		MachineId:    machineId,
		Protocol:     entity.MachineProtocolSsh,
		AuthCertName: cli.Info.AuthCertName,
		Path:         remotePath,
	}
	written, _, err := b.machineFileApp.UploadFile(ctx, opParam, filename, reader)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Bytes = written
	res.Success = true
	return res
}
