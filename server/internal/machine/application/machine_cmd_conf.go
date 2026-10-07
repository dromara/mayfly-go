package application

import (
	"context"
	"mayfly-go/internal/machine/application/dto"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"regexp"
	"sync"
)

// cmdRulePattern 机器命令过滤规则的一条正则表达式。
//
// 处置级别（禁止/提醒）由流程定义的触发策略决定，这里只提供「命中了哪条规则」这一事实，
// 因此不再携带策略字段：策略曾长期只被复制与打日志，从未参与任何判定
type cmdRulePattern struct {
	raw      string
	compiled *regexp.Regexp
}

type MachineCmdConf interface {
	base.App[*entity.MachineCmdConf]

	SaveCmdConf(ctx context.Context, cmdConf *dto.SaveMachineCmdConf) error

	DeleteCmdConf(ctx context.Context, id uint64) error

	// MatchCmdRule 判断命令是否命中该机器标签下管理员配置的命令过滤规则，返回命中的原始表达式
	MatchCmdRule(ctx context.Context, cmd string, tagPaths ...string) (matched bool, pattern string)

	// ValidateCmdPatterns 校验命令过滤表达式可编译，供保存时拦截坏表达式
	ValidateCmdPatterns(patterns []string) error
}

type machineCmdConfAppImpl struct {
	base.AppImpl[*entity.MachineCmdConf, repository.MachineCmdConf]

	tagTreeRelateApp tagapp.TagTreeRelate `inject:"T"`
}

var _ MachineCmdConf = (*machineCmdConfAppImpl)(nil)

var _ (MachineCmdConf) = (*machineCmdConfAppImpl)(nil)

func (m *machineCmdConfAppImpl) SaveCmdConf(ctx context.Context, cmdConfParam *dto.SaveMachineCmdConf) error {
	cmdConf := cmdConfParam.CmdConf
	// 坏表达式必须在落库前拦下：一旦存进去，运行期只能选择「跳过该规则」或「按命中处理」，
	// 前者等于让安全规则静默失效，后者会误拦正常命令
	if err := m.ValidateCmdPatterns(cmdConf.Cmds); err != nil {
		return err
	}

	return m.Tx(ctx, func(ctx context.Context) error {
		return m.Save(ctx, cmdConf)
	}, func(ctx context.Context) error {
		return m.tagTreeRelateApp.RelateTag(ctx, tagentity.TagRelateTypeMachineCmd, cmdConf.Id, cmdConfParam.CodePaths...)
	})
}

func (m *machineCmdConfAppImpl) DeleteCmdConf(ctx context.Context, id uint64) error {
	_, err := m.GetById(id)
	if err != nil {
		return errorx.NewBiz("cmd config not found")
	}

	return m.Tx(ctx, func(ctx context.Context) error {
		return m.DeleteById(ctx, id)
	}, func(ctx context.Context) error {
		return m.tagTreeRelateApp.DeleteByCond(ctx, &tagentity.TagTreeRelate{
			RelateType: tagentity.TagRelateTypeMachineCmd,
			RelateId:   id,
		})
	})
}

// 命令过滤表达式编译结果缓存：终端每条命令都要过这道校验，不得每次重新编译全部规则
var cmdPatternCache sync.Map

func compileCmdPattern(pattern string) (*regexp.Regexp, error) {
	if cached, ok := cmdPatternCache.Load(pattern); ok {
		return cached.(*regexp.Regexp), nil
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	cmdPatternCache.Store(pattern, compiled)
	return compiled, nil
}

func (m *machineCmdConfAppImpl) ValidateCmdPatterns(patterns []string) error {
	for _, pattern := range patterns {
		if _, err := compileCmdPattern(pattern); err != nil {
			return errorx.NewBizf("the command pattern [%s] is invalid: %s", pattern, err.Error())
		}
	}
	return nil
}

func (m *machineCmdConfAppImpl) MatchCmdRule(ctx context.Context, cmd string, tagPaths ...string) (bool, string) {
	for _, pattern := range m.cmdRulePatterns(ctx, tagPaths) {
		if pattern.compiled == nil {
			// 保存侧已校过编译，走到这里说明规则是历史脏数据或引擎升级后语义变化
			// 宁可不放行也不能静默失效，故按命中处理并留痕
			logx.WarnfContext(ctx, "machine cmd rule [%s] cannot be compiled, treated as matched", pattern.raw)
			return true, pattern.raw
		}
		if pattern.compiled.MatchString(cmd) {
			return true, pattern.raw
		}
	}
	return false, ""
}

func (m *machineCmdConfAppImpl) cmdRulePatterns(ctx context.Context, tagPaths []string) []cmdRulePattern {
	cmdConfIds, err := m.tagTreeRelateApp.GetGovernRelateIds(ctx, tagentity.TagRelateTypeMachineCmd, tagPaths...)
	if err != nil {
		// 取不到规则不等于没有风险：无法判定时按命中一条虚拟规则处理，由调用方给出拦截提示
		logx.ErrorfContext(ctx, "failed to get machine cmd config: %s", err.Error())
		return []cmdRulePattern{{raw: "<unavailable>"}}
	}
	if len(cmdConfIds) == 0 {
		return nil
	}

	cmdConfs, err := m.GetByIds(cmdConfIds)
	if err != nil {
		logx.ErrorfContext(ctx, "failed to load machine cmd config: %s", err.Error())
		return []cmdRulePattern{{raw: "<unavailable>"}}
	}

	patterns := make([]cmdRulePattern, 0, len(cmdConfs))
	for _, cmdConf := range cmdConfs {
		for _, raw := range cmdConf.Cmds {
			compiled, compileErr := compileCmdPattern(raw)
			if compileErr != nil {
				logx.ErrorfContext(ctx, "cmd config [%s], regex compilation failed: %s", raw, compileErr.Error())
			}
			patterns = append(patterns, cmdRulePattern{raw: raw, compiled: compiled})
		}
	}
	return patterns
}
