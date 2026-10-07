package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/machine/imsg"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/errorx"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 磁盘分析限额集中收口：目录深度上限与命令超时，避免超大目录把请求拖死
const (
	diskMaxDepth    = 3
	diskAnalyzeTo   = 30 * time.Second
	diskDefaultPath = "/"
)

// diskPathPattern 允许安全拼进远端 shell 的路径字符集（字母数字与 . _ / -），
// 其余字符（; ` | & $ ( ) 引号 空格 换行等）一律拒绝，杜绝命令注入
var diskPathPattern = regexp.MustCompile(`^[A-Za-z0-9._/\-]+$`)

// DiskNode 单个目录的占用大小
type DiskNode struct {
	Path string `json:"path"`
	Size uint64 `json:"size"` // 字节
}

// DiskAnalysisResult 磁盘分析结果
type DiskAnalysisResult struct {
	Path  string     `json:"path"`
	Depth int        `json:"depth"`
	Nodes []DiskNode `json:"nodes"` // 按大小降序的目录条目（du 输出）
}

// MachineDisk 机器磁盘占用分析。独立 app，接口隔离，不膨胀 Machine 接口。
//
// 经目标机 du 命令按目录聚合大小，用于磁盘告警后的自助定位（Top N 大目录）
type MachineDisk interface {
	// Analyze 分析指定机器某路径下的目录占用（深度受限、超时兜底、逐机访问权校验）
	Analyze(ctx context.Context, machineId uint64, path string, depth int) (*DiskAnalysisResult, error)
}

type machineDiskAppImpl struct {
	tagApp tagapp.TagTreeService `inject:"T"`
}

var _ MachineDisk = (*machineDiskAppImpl)(nil)

func (d *machineDiskAppImpl) Analyze(ctx context.Context, machineId uint64, path string, depth int) (*DiskAnalysisResult, error) {
	if strings.TrimSpace(path) == "" {
		path = diskDefaultPath
	}
	if !diskPathPattern.MatchString(path) {
		return nil, errorx.NewBizI(ctx, imsg.ErrDiskPathInvalid)
	}
	if depth <= 0 || depth > diskMaxDepth {
		depth = diskMaxDepth
	}

	// 逐机访问权校验：与终端/批量执行同一资源级隔离
	me, err := GetMachineApp().GetById(machineId)
	if err != nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrMachineNotFoundById, "machineId", machineId)
	}
	if err := d.tagApp.CanAccessByCode(ctx, int8(tagentity.TagTypeMachine), me.Code); err != nil {
		return nil, err
	}

	cli, err := GetMachineApp().GetCli(ctx, machineId)
	if err != nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrBatchConnFailed, "reason", err.Error())
	}

	// du -B1 输出「字节数<TAB>路径」，限制深度并丢弃权限不足的噪声行
	cmd := fmt.Sprintf("du -B1 --max-depth=%d '%s' 2>/dev/null", depth, path)
	out, runErr := cli.RunWithTimeout(diskAnalyzeTo, cmd)
	if out == "" {
		if runErr != nil {
			return nil, runErr
		}
		return &DiskAnalysisResult{Path: path, Depth: depth, Nodes: []DiskNode{}}, nil
	}

	nodes := parseDuOutput(out)
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Size > nodes[j].Size })
	return &DiskAnalysisResult{Path: path, Depth: depth, Nodes: nodes}, nil
}

// parseDuOutput 解析 du 的「size\tpath」逐行输出
func parseDuOutput(out string) []DiskNode {
	nodes := make([]DiskNode, 0, 64)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.IndexByte(line, '\t')
		if idx <= 0 {
			continue
		}
		size, err := strconv.ParseUint(strings.TrimSpace(line[:idx]), 10, 64)
		if err != nil {
			continue
		}
		nodes = append(nodes, DiskNode{Size: size, Path: strings.TrimSpace(line[idx+1:])})
	}
	return nodes
}
