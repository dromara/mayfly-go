package application

import (
	"context"
	flowapp "mayfly-go/internal/flow/application"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/internal/pkg/utils"
	"mayfly-go/internal/redis/application/dto"
	"mayfly-go/internal/redis/domain/entity"
	"mayfly-go/internal/redis/domain/repository"
	"mayfly-go/internal/redis/imsg"
	"mayfly-go/internal/redis/rdm"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/stringx"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
)

type Redis interface {
	base.App[*entity.Redis]
	flowapp.FlowBizHandler

	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.RedisQuery, orderBy ...string) (*model.PageResult[*entity.Redis], error)

	// 测试连接
	TestConn(re *dto.SaveRedis) error

	SaveRedis(ctx context.Context, param *dto.SaveRedis) error

	// 删除数据库信息
	Delete(ctx context.Context, id uint64) error

	// 获取数据库连接实例
	// id: 数据库实例id
	// db: 库号
	GetRedisConn(ctx context.Context, id uint64, db int) (*rdm.RedisConn, error)

	// 执行redis命令
	RunCmd(ctx context.Context, redisConn *rdm.RedisConn, cmdParam *dto.RunCmd) (any, error)

	// CheckCmdFlow 按已绑定流程定义的触发策略判定该命令能否直接执行：命令名与完整命令文本作为策略字段，
	// 所有会改动 redis 数据的入口（控制台命令、类型化数据操作）都必须先过这道校验。
	//
	// ack 由**调用入口**声明：命令控制台与 key 面板是两个不同的入口，谁能弹确认在这里各写各的
	//（数据库侧相反——同一个 exec-sql 端点既服务单条也服务批量选区，只有客户端知道这次是不是一批，
	// 所以那边改由请求声明）。提醒命中时命令尚未执行，此刻转审批才是「只执行一次」；
	// 不能追问的入口保持「不阻断、只回显」
	CheckCmdFlow(ctx context.Context, redisConn *rdm.RedisConn, cmdArgs []any, ack WarnAck) error

	// TriggerProcdefOf / CheckCmdTrigger 是给批量命令的两段式入口：先解析一次流程定义，再逐条判定
	TriggerProcdefOf(ctx context.Context, redisConn *rdm.RedisConn) *flowentity.Procdef
	CheckCmdTrigger(ctx context.Context, procdef *flowentity.Procdef, cmdArgs []any, ack WarnAck) error
}

// WarnAck 「仅提醒」命中的处理方式，由每个操作入口显式声明，不从请求形状推断
// （曾按「一条请求=单条执行」推断，而前端批量是逐条发请求的，结果批量一条都没执行）。
// Ask = 该入口能否弹确认框（取决于有没有操作者在等这次结果）；
// Acknowledged = 操作者已经选过「直接执行」
// WarnAckMode 「仅提醒」命中时该入口的处置能力：能不能问，以及问完之后有没有路可走
type WarnAckMode int

const (
	// WarnNoAsk 无法弹确认（结果会被广播给所有客户端，或没人在界面上等这次操作）：只回显不阻断
	WarnNoAsk WarnAckMode = iota
	// WarnAskTicketable 能弹确认，且拦截处可以就地转工单审批（命令控制台）
	WarnAskTicketable
	// WarnAskDirect 能弹确认，但该入口没有提单表单（key 面板的类型化操作）
	WarnAskDirect
)

type WarnAck struct {
	Mode         WarnAckMode
	Acknowledged bool
}

// askable 该入口是否需要操作者先确认
func (a WarnAck) askable() bool { return a.Mode != WarnNoAsk }

// warnNoAsk 不弹确认的入口：提醒只回显/落日志、不阻断。
// 用于结果会被广播给所有客户端、或没有人在界面上等这次操作的路径（如订阅 key 变更后的批量处理）
var warnNoAsk = WarnAck{}

// WarnAckOf 能弹确认，且该入口给得出提单（命令控制台的一键提单、key 面板读内容的「申请查看」）
func WarnAckOf(acknowledged bool) WarnAck {
	return WarnAck{Mode: WarnAskTicketable, Acknowledged: acknowledged}
}

// WarnAckDirectOf key 面板的类型化写操作：能弹确认，但没有提单表单，话术要另配
func WarnAckDirectOf(acknowledged bool) WarnAck {
	return WarnAck{Mode: WarnAskDirect, Acknowledged: acknowledged}
}

var _ Redis = (*redisAppImpl)(nil)

type redisAppImpl struct {
	base.AppImpl[*entity.Redis, repository.Redis]

	tagApp              tagapp.TagTreeService   `inject:"T"`
	procdefApp          flowapp.Procdef         `inject:"T"`
	resourceAuthCertApp tagapp.ResourceAuthCert `inject:"T"`
}

// 分页获取redis列表
func (r *redisAppImpl) GetPageList(condition *entity.RedisQuery, orderBy ...string) (*model.PageResult[*entity.Redis], error) {
	return r.GetRepo().GetRedisList(condition, orderBy...)
}

func (r *redisAppImpl) TestConn(param *dto.SaveRedis) error {
	db := 0
	re := param.Redis
	if re.Db != "" {
		db = cast.ToInt(strings.Split(re.Db, ",")[0])
	}

	authCert, err := r.resourceAuthCertApp.GetRealAuthCert(param.AuthCert)
	if err != nil {
		return err
	}

	rc, err := re.ToRedisInfo(db, authCert).Conn()
	if err != nil {
		return err
	}
	rc.Close()
	return nil
}

func (r *redisAppImpl) SaveRedis(ctx context.Context, param *dto.SaveRedis) error {
	re := param.Redis
	tagCodePaths := param.TagCodePaths
	// 查找是否存在该库
	oldRedis := &entity.Redis{
		Host:               re.Host,
		SshTunnelMachineId: re.SshTunnelMachineId,
	}
	err := r.GetByCond(oldRedis)

	if re.Id == 0 {
		if err == nil {
			return errorx.NewBizI(ctx, imsg.ErrRedisInfoExist)
		}
		// 生成随机编号
		re.Code = stringx.Rand(10)

		return r.Tx(ctx, func(ctx context.Context) error {
			return r.Insert(ctx, re)
		}, func(ctx context.Context) error {
			return r.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
				ResourceTag: &tagdto.ResourceTag{
					Type: tagentity.TagTypeRedis,
					Code: re.Code,
					Name: re.Name,
				},
				ParentTagCodePaths: tagCodePaths,
			})
		}, func(ctx context.Context) error {
			return r.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
				ResourceCode: re.Code,
				ResourceType: tagentity.TagTypeRedis,
				AuthCerts:    []*tagentity.ResourceAuthCert{param.AuthCert},
			})
		})
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && oldRedis.Id != re.Id {
		return errorx.NewBizI(ctx, imsg.ErrRedisInfoExist)
	}
	// 如果修改了redis实例的库信息，则关闭旧库的连接
	if oldRedis.Db != re.Db || oldRedis.SshTunnelMachineId != re.SshTunnelMachineId {
		for _, dbStr := range strings.Split(oldRedis.Db, ",") {
			db, _ := strconv.Atoi(dbStr)
			rdm.CloseConn(re.Id, db)
		}
	}
	// 如果调整了host sshid等会查不到旧数据，故需要根据id获取旧信息将code赋值给标签进行关联
	if oldRedis.Code == "" {
		oldRedis, _ = r.GetById(re.Id)
	}

	// 校验当前操作者是否有权操作该资源，防止越权修改他人资源信息
	if err := r.tagApp.CanAccessByCode(ctx, consts.ResourceTypeRedis, oldRedis.Code); err != nil {
		return err
	}

	re.Code = ""
	return r.Tx(ctx, func(ctx context.Context) error {
		return r.UpdateById(ctx, re)
	}, func(ctx context.Context) error {
		if oldRedis.Name != re.Name {
			if err := r.tagApp.UpdateTagName(ctx, tagentity.TagTypeRedis, oldRedis.Code, re.Name); err != nil {
				return err
			}
		}
		return r.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
			ResourceTag: &tagdto.ResourceTag{
				Type: tagentity.TagTypeRedis,
				Code: oldRedis.Code,
				Name: re.Name,
			},
			ParentTagCodePaths: tagCodePaths,
		})
	}, func(ctx context.Context) error {
		return r.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
			ResourceCode: oldRedis.Code,
			ResourceType: tagentity.TagTypeRedis,
			AuthCerts:    []*tagentity.ResourceAuthCert{param.AuthCert},
		})
	})
}

// 删除Redis信息
func (r *redisAppImpl) Delete(ctx context.Context, id uint64) error {
	re, err := r.GetById(id)
	if err != nil {
		return errorx.NewBiz("redis not found")
	}
	// 如果存在连接，则关闭所有库连接信息
	for _, dbStr := range strings.Split(re.Db, ",") {
		rdm.CloseConn(re.Id, cast.ToInt(dbStr))
	}

	return r.Tx(ctx, func(ctx context.Context) error {
		return r.DeleteById(ctx, id)
	}, func(ctx context.Context) error {
		return r.tagApp.SaveResourceTag(ctx, &tagdto.SaveResourceTag{
			ResourceTag: &tagdto.ResourceTag{
				Type: tagentity.TagTypeRedis,
				Code: re.Code,
			},
		})
	}, func(ctx context.Context) error {
		return r.resourceAuthCertApp.RelateAuthCert(ctx, &tagdto.RelateAuthCert{
			ResourceCode: re.Code,
			ResourceType: tagentity.TagTypeRedis,
		})
	})
}

// 获取数据库连接实例
func (r *redisAppImpl) GetRedisConn(ctx context.Context, id uint64, db int) (*rdm.RedisConn, error) {
	// 连接层统一进行数据权限校验，避免各操作接口遗漏鉴权
	re, err := r.GetById(id)
	if err != nil {
		return nil, errorx.NewBiz("redis not found")
	}
	if err := r.tagApp.CanAccessByCode(ctx, consts.ResourceTypeRedis, re.Code); err != nil {
		return nil, err
	}

	return rdm.GetRedisConn(ctx, id, db, func() (*rdm.RedisInfo, error) {
		authCert, err := r.resourceAuthCertApp.GetResourceAuthCert(tagentity.TagTypeRedis, re.Code)
		if err != nil {
			return nil, err
		}
		return re.ToRedisInfo(db, authCert, r.tagApp.ListTagPathByTypeAndCode(consts.ResourceTypeRedis, re.Code)...), nil
	})
}

func (r *redisAppImpl) RunCmd(ctx context.Context, redisConn *rdm.RedisConn, cmdParam *dto.RunCmd) (any, error) {
	if redisConn == nil {
		return nil, errorx.NewBiz("redis connection not exist")
	}

	if err := r.CheckCmdFlow(ctx, redisConn, cmdParam.Cmd, WarnAckOf(cmdParam.AckWarn)); err != nil {
		return nil, err
	}

	res, err := redisConn.RunCmd(ctx, cmdParam.Cmd...)
	// 获取的key不存在，不报错
	if err == redis.Nil {
		return nil, nil
	}
	return res, err
}

// CheckCmdFlow 判定单条命令能否直接执行：解析流程定义 + 求值，给「一次只发一条命令」的入口用
func (r *redisAppImpl) CheckCmdFlow(ctx context.Context, redisConn *rdm.RedisConn, cmdArgs []any, ack WarnAck) error {
	if redisConn == nil {
		return errorx.NewBiz("redis connection not exist")
	}
	return r.CheckCmdTrigger(ctx, r.TriggerProcdefOf(ctx, redisConn), cmdArgs, ack)
}

// TriggerProcdefOf 该连接生效的流程定义（未绑定流程时返回 nil）。
//
// 同一连接上的多条命令解析结果相同，所以批量入口只需解析一次；
// 每条命令都解析等于每条命令多查两趟库（曾经如此：key 面板一次改十几个 key 就是二十几趟查询）
func (r *redisAppImpl) TriggerProcdefOf(ctx context.Context, redisConn *rdm.RedisConn) *flowentity.Procdef {
	if redisConn == nil {
		return nil
	}
	return r.procdefApp.GetProcdefByCodePath(ctx, redisConn.Info.CodePath...)
}

// CheckCmdTrigger 用已解析好的流程定义判定单条命令，与 CheckCmdFlow 同一套判定口径，只是省掉重复解析
func (r *redisAppImpl) CheckCmdTrigger(ctx context.Context, procdef *flowentity.Procdef, cmdArgs []any, ack WarnAck) error {
	if len(cmdArgs) == 0 {
		return errorx.NewBiz("redis cmd cannot be empty")
	}
	if procdef == nil {
		return nil
	}
	return r.checkCmdTrigger(ctx, procdef, cast.ToString(cmdArgs[0]), redisCmdText(cmdArgs), ack)
}

type FlowRedisRunCmdBizForm struct {
	Id  uint64 `json:"id"`  // redis id
	Db  int    `json:"db"`  // redis db
	Cmd string `json:"cmd"` // redis cmd
}

func (r *redisAppImpl) FlowBizHandle(ctx context.Context, bizHandleParam *flowapp.BizHandleParam) (any, error) {
	procinst := bizHandleParam.Procinst
	bizKey := procinst.BizKey
	procinstStatus := procinst.Status

	logx.Debugf("RedisRunWriteCmd FlowBizHandle -> bizKey: %s, procinstStatus: %s", bizKey, flowentity.ProcinstStatusEnum.GetDesc(procinstStatus))
	// 流程非完成状态，不处理
	if procinstStatus != flowentity.ProcinstStatusCompleted {
		return nil, nil
	}

	runCmdParam, err := jsonx.ToByStr[FlowRedisRunCmdBizForm](procinst.BizForm)
	if err != nil {
		return nil, errorx.NewBizf("failed to parse the business form information: %s", err.Error())
	}

	// 回放以工单发起人身份执行：审批人未必有这台实例的运维权限，用审批人身份会在鉴权处失败，
	// 留下「审批通过却执行失败」的结果；执行归属也应记在发起人身上
	ctx = flowapp.BizOperatorContext(ctx, bizHandleParam)

	redisConn, err := r.GetRedisConn(ctx, runCmdParam.Id, runCmdParam.Db)
	if err != nil {
		return nil, err
	}

	handleRes := make([]map[string]any, 0)
	hasErr := false

	utils.SplitStmts(strings.NewReader(runCmdParam.Cmd), ';', func(stmt string) error {
		cmd := strings.TrimSpace(stmt)
		runRes := collx.Kvs("cmd", cmd)
		if res, err := redisConn.RunCmd(ctx, collx.ArrayMap[string, any](parseRedisCommand(cmd), func(val string) any { return val })...); err != nil {
			runRes["res"] = err.Error()
			hasErr = true
		} else {
			runRes["res"] = res
		}
		handleRes = append(handleRes, runRes)
		return nil
	})

	if hasErr {
		return handleRes, errorx.NewBizI(ctx, imsg.ErrHasRunFailCmd)
	}
	return handleRes, nil
}

// parseRedisCommand 解析 Redis 命令字符串到数组
func parseRedisCommand(commandStr string) []string {
	var args []string
	inSingleQuote := false
	inDoubleQuote := false
	currentArg := ""

	for _, char := range commandStr {
		switch char {
		case '\'':
			if !inDoubleQuote && !inSingleQuote {
				inSingleQuote = true
			} else if inSingleQuote && !inDoubleQuote {
				inSingleQuote = false
				args = append(args, strings.TrimSpace(currentArg))
				currentArg = ""
			}
		case '"':
			if !inSingleQuote && !inDoubleQuote {
				inDoubleQuote = true
			} else if !inSingleQuote && inDoubleQuote {
				inDoubleQuote = false
				args = append(args, strings.TrimSpace(currentArg))
				currentArg = ""
			}
		case ' ':
			if !inSingleQuote && !inDoubleQuote {
				if strings.TrimSpace(currentArg) != "" {
					args = append(args, strings.TrimSpace(currentArg))
					currentArg = ""
				}
			} else {
				currentArg += string(char)
			}
		default:
			currentArg += string(char)
		}
	}

	if strings.TrimSpace(currentArg) != "" {
		args = append(args, strings.TrimSpace(currentArg))
	}

	return args
}
