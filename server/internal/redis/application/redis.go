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

	"github.com/may-fly/cast"
	"github.com/redis/go-redis/v9"
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
}

var _ Redis = (*redisAppImpl)(nil)

type redisAppImpl struct {
	base.AppImpl[*entity.Redis, repository.Redis]

	tagApp              tagapp.TagTree          `inject:"T"`
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
	return rdm.GetRedisConn(ctx, id, db, func() (*rdm.RedisInfo, error) {
		// 缓存不存在，则回调获取redis信息
		re, err := r.GetById(id)
		if err != nil {
			return nil, errorx.NewBiz("redis not found")
		}
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

	// 开启工单流程，则校验该流程是否需要校验
	if procdef := r.procdefApp.GetProcdefByCodePath(ctx, redisConn.Info.CodePath...); procdef != nil {
		cmd := cmdParam.Cmd[0]
		cmdType := "read"
		if rdm.IsWriteCmd(cmd) {
			cmdType = "write"
		}
		if needStartProc := procdef.MatchCondition(RedisRunCmdFlowBizType, collx.Kvs("cmdType", cmdType, "cmd", cmd)); needStartProc {
			return nil, errorx.NewBizI(ctx, imsg.ErrSubmitFlowRunCmd)
		}
	}

	res, err := redisConn.RunCmd(ctx, cmdParam.Cmd...)
	// 获取的key不存在，不报错
	if err == redis.Nil {
		return nil, nil
	}
	return res, err
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

	runCmdParam, err := jsonx.To[*FlowRedisRunCmdBizForm](procinst.BizForm)
	if err != nil {
		return nil, errorx.NewBiz("failed to parse the business form information: %s", err.Error())
	}

	redisConn, err := r.GetRedisConn(ctx, runCmdParam.Id, runCmdParam.Db)
	if err != nil {
		return nil, err
	}

	handleRes := make([]map[string]any, 0)
	hasErr := false

	utils.SplitStmts(strings.NewReader(runCmdParam.Cmd), func(stmt string) error {
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
