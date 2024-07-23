package application

import (
	"context"
	"mayfly-go/internal/common/consts"
	flowapp "mayfly-go/internal/flow/application"
	flowdto "mayfly-go/internal/flow/application/dto"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/redis/application/dto"
	"mayfly-go/internal/redis/domain/entity"
	"mayfly-go/internal/redis/domain/repository"
	"mayfly-go/internal/redis/rdm"
	tagapp "mayfly-go/internal/tag/application"
	tagdto "mayfly-go/internal/tag/application/dto"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
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
	GetPageList(condition *entity.RedisQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// 测试连接
	TestConn(re *dto.SaveRedis) error

	SaveRedis(ctx context.Context, param *dto.SaveRedis) error

	// 删除数据库信息
	Delete(ctx context.Context, id uint64) error

	// 获取数据库连接实例
	// id: 数据库实例id
	// db: 库号
	GetRedisConn(id uint64, db int) (*rdm.RedisConn, error)

	// 执行redis命令
	RunCmd(ctx context.Context, redisConn *rdm.RedisConn, cmdParam *dto.RunCmd) (any, error)
}

type redisAppImpl struct {
	base.AppImpl[*entity.Redis, repository.Redis]

	tagApp              tagapp.TagTree          `inject:"TagTreeApp"`
	procdefApp          flowapp.Procdef         `inject:"ProcdefApp"`
	procinstApp         flowapp.Procinst        `inject:"ProcinstApp"`
	resourceAuthCertApp tagapp.ResourceAuthCert `inject:"ResourceAuthCertApp"`
}

// 注入RedisRepo
func (r *redisAppImpl) InjectRedisRepo(repo repository.Redis) {
	r.Repo = repo
}

// 分页获取redis列表
func (r *redisAppImpl) GetPageList(condition *entity.RedisQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return r.GetRepo().GetRedisList(condition, pageParam, toEntity, orderBy...)
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
			return errorx.NewBiz("该实例已存在")
		}
		if r.CountByCond(&entity.Redis{Code: re.Code}) > 0 {
			return errorx.NewBiz("该编码已存在")
		}

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
		return errorx.NewBiz("该实例已存在")
	}
	// 如果修改了redis实例的库信息，则关闭旧库的连接
	if oldRedis.Db != re.Db || oldRedis.SshTunnelMachineId != re.SshTunnelMachineId {
		for _, dbStr := range strings.Split(oldRedis.Db, ",") {
			db, _ := strconv.Atoi(dbStr)
			rdm.CloseConn(re.Id, db)
		}
	}
	// 如果调整了ssh等会查不到旧数据，故需要根据id获取旧信息将code赋值给标签进行关联
	if oldRedis.Code == "" {
		oldRedis, _ = r.GetById(re.Id)
	}

	re.Code = ""
	return r.Tx(ctx, func(ctx context.Context) error {
		return r.UpdateById(ctx, re)
	}, func(ctx context.Context) error {
		if oldRedis.Name != re.Name {
			if err := r.tagApp.UpdateTagName(ctx, tagentity.TagTypeMachine, oldRedis.Code, re.Name); err != nil {
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
		return errorx.NewBiz("该redis信息不存在")
	}
	// 如果存在连接，则关闭所有库连接信息
	for _, dbStr := range strings.Split(re.Db, ",") {
		db, _ := strconv.Atoi(dbStr)
		rdm.CloseConn(re.Id, db)
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
func (r *redisAppImpl) GetRedisConn(id uint64, db int) (*rdm.RedisConn, error) {
	return rdm.GetRedisConn(id, db, func() (*rdm.RedisInfo, error) {
		// 缓存不存在，则回调获取redis信息
		re, err := r.GetById(id)
		if err != nil {
			return nil, errorx.NewBiz("redis信息不存在")
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
		return nil, errorx.NewBiz("redis连接不存在")
	}

	// 开启工单流程，并且为写入命令，则开启对应审批流程
	if procdefId := r.procdefApp.GetProcdefIdByCodePath(ctx, redisConn.Info.CodePath...); procdefId != 0 && rdm.IsWriteCmd(cmdParam.Cmd[0]) {
		_, err := r.procinstApp.StartProc(ctx, procdefId, &flowdto.StarProc{
			BizType: RedisRunWriteCmdFlowBizType,
			BizKey:  stringx.Rand(24),
			BizForm: jsonx.ToStr(cmdParam),
			Remark:  cmdParam.Remark,
		})
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	res, err := redisConn.RunCmd(ctx, cmdParam.Cmd...)
	// 获取的key不存在，不报错
	if err == redis.Nil {
		return nil, nil
	}
	return res, err
}

func (r *redisAppImpl) FlowBizHandle(ctx context.Context, bizHandleParam *flowapp.BizHandleParam) error {
	bizKey := bizHandleParam.BizKey
	procinstStatus := bizHandleParam.ProcinstStatus

	logx.Debugf("RedisRunWriteCmd FlowBizHandle -> bizKey: %s, procinstStatus: %s", bizKey, flowentity.ProcinstStatusEnum.GetDesc(procinstStatus))
	// 流程非完成状态，不处理
	if procinstStatus != flowentity.ProcinstStatusCompleted {
		return nil
	}

	runCmdParam, err := jsonx.To(bizHandleParam.BizForm, new(dto.RunCmd))
	if err != nil {
		return errorx.NewBiz("业务表单信息解析失败: %s", err.Error())
	}

	redisConn, err := r.GetRedisConn(runCmdParam.Id, runCmdParam.Db)
	if err != nil {
		return err
	}

	_, err = redisConn.RunCmd(ctx, runCmdParam.Cmd...)
	return err
}
