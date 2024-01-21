package api

import (
	"context"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/redis/api/form"
	"mayfly-go/internal/redis/api/vo"
	"mayfly-go/internal/redis/application"
	"mayfly-go/internal/redis/domain/entity"
	"mayfly-go/internal/redis/rdm"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/cryptox"
	"mayfly-go/pkg/utils/stringx"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Redis struct {
	RedisApp application.Redis `inject:""`
	TagApp   tagapp.TagTree    `inject:"TagTreeApp"`
}

func (r *Redis) RedisList(rc *req.Ctx) {
	queryCond, page := ginx.BindQueryAndPage[*entity.RedisQuery](rc.GinCtx, new(entity.RedisQuery))

	// 不存在可访问标签id，即没有可操作数据
	codes := r.TagApp.GetAccountResourceCodes(rc.GetLoginAccount().Id, consts.TagResourceTypeRedis, queryCond.TagPath)
	if len(codes) == 0 {
		rc.ResData = model.EmptyPageResult[any]()
		return
	}
	queryCond.Codes = codes

	res, err := r.RedisApp.GetPageList(queryCond, page, new([]vo.Redis))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (r *Redis) TestConn(rc *req.Ctx) {
	form := &form.Redis{}
	redis := ginx.BindJsonAndCopyTo[*entity.Redis](rc.GinCtx, form, new(entity.Redis))

	// 密码解密，并使用解密后的赋值
	originPwd, err := cryptox.DefaultRsaDecrypt(redis.Password, true)
	biz.ErrIsNilAppendErr(err, "解密密码错误: %s")
	redis.Password = originPwd

	biz.ErrIsNil(r.RedisApp.TestConn(redis))
}

func (r *Redis) Save(rc *req.Ctx) {
	form := &form.Redis{}
	redis := ginx.BindJsonAndCopyTo[*entity.Redis](rc.GinCtx, form, new(entity.Redis))

	// 密码解密，并使用解密后的赋值
	originPwd, err := cryptox.DefaultRsaDecrypt(redis.Password, true)
	biz.ErrIsNilAppendErr(err, "解密密码错误: %s")
	redis.Password = originPwd

	// 密码脱敏记录日志
	form.Password = "****"
	rc.ReqParam = form

	biz.ErrIsNil(r.RedisApp.SaveRedis(rc.MetaCtx, redis, form.TagId...))
}

// 获取redis实例密码，由于数据库是加密存储，故提供该接口展示原文密码
func (r *Redis) GetRedisPwd(rc *req.Ctx) {
	rid := uint64(ginx.PathParamInt(rc.GinCtx, "id"))
	re, err := r.RedisApp.GetById(new(entity.Redis), rid, "Password")
	biz.ErrIsNil(err, "redis信息不存在")
	if err := re.PwdDecrypt(); err != nil {
		biz.ErrIsNil(err)
	}
	rc.ResData = re.Password
}

func (r *Redis) DeleteRedis(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		r.RedisApp.Delete(rc.MetaCtx, uint64(value))
	}
}

func (r *Redis) RedisInfo(rc *req.Ctx) {
	g := rc.GinCtx
	ri, err := r.RedisApp.GetRedisConn(uint64(ginx.PathParamInt(g, "id")), 0)
	biz.ErrIsNil(err)

	section := rc.GinCtx.Query("section")
	mode := ri.Info.Mode
	ctx := context.Background()
	var redisCli *redis.Client

	if mode == "" || mode == rdm.StandaloneMode || mode == rdm.SentinelMode {
		redisCli = ri.Cli
	} else if mode == rdm.ClusterMode {
		host := rc.GinCtx.Query("host")
		biz.NotEmpty(host, "集群模式host信息不能为空")
		clusterClient := ri.ClusterCli
		// 遍历集群的master节点找到该redis client
		clusterClient.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
			if host == client.Options().Addr {
				redisCli = client
			}
			return nil
		})
		if redisCli == nil {
			// 遍历集群的slave节点找到该redis client
			clusterClient.ForEachSlave(ctx, func(ctx context.Context, client *redis.Client) error {
				if host == client.Options().Addr {
					redisCli = client
				}
				return nil
			})
		}
		biz.NotNil(redisCli, "该实例不在该集群中")
	}

	var res string
	if section == "" {
		res, err = redisCli.Info(ctx).Result()
	} else {
		res, err = redisCli.Info(ctx, section).Result()
	}

	biz.ErrIsNilAppendErr(err, "获取redis info失败: %s")

	datas := strings.Split(res, "\r\n")
	i := 0
	length := len(datas)

	parseMap := make(map[string]map[string]string)
	for {
		if i >= length {
			break
		}
		if strings.Contains(datas[i], "#") {
			key := stringx.SubString(datas[i], strings.Index(datas[i], "#")+1, stringx.Len(datas[i]))
			i++
			key = strings.Trim(key, " ")

			sectionMap := make(map[string]string)
			for {
				if i >= length || !strings.Contains(datas[i], ":") {
					break
				}
				pair := strings.Split(datas[i], ":")
				i++
				if len(pair) != 2 {
					continue
				}
				sectionMap[pair[0]] = pair[1]
			}
			parseMap[key] = sectionMap
		} else {
			i++
		}
	}
	rc.ResData = parseMap
}

func (r *Redis) ClusterInfo(rc *req.Ctx) {
	g := rc.GinCtx
	ri, err := r.RedisApp.GetRedisConn(uint64(ginx.PathParamInt(g, "id")), 0)
	biz.ErrIsNil(err)
	biz.IsEquals(ri.Info.Mode, rdm.ClusterMode, "非集群模式")
	info, _ := ri.ClusterCli.ClusterInfo(context.Background()).Result()
	nodesStr, _ := ri.ClusterCli.ClusterNodes(context.Background()).Result()

	nodesRes := make([]map[string]string, 0)
	nodes := strings.Split(nodesStr, "\n")
	for _, node := range nodes {
		if node == "" {
			continue
		}
		nodeInfos := strings.Split(node, " ")
		node := make(map[string]string)
		node["nodeId"] = nodeInfos[0]
		// ip:port1@port2：port1指redis服务器与客户端通信的端口，port2则是集群内部节点间通信的端口
		node["ip"] = nodeInfos[1]
		node["flags"] = nodeInfos[2]
		// 如果节点是slave，并且已知master节点，则为master节点ID；否则为符号"-"
		node["masterSlaveRelation"] = nodeInfos[3]
		// 最近一次发送ping的时间，这个时间是一个unix毫秒时间戳，0代表没有发送过
		node["pingSent"] = nodeInfos[4]
		// 最近一次收到pong的时间，使用unix时间戳表示
		node["pongRecv"] = nodeInfos[5]
		// 节点的epoch值（如果该节点是从节点，则为其主节点的epoch值）。每当节点发生失败切换时，都会创建一个新的，独特的，递增的epoch。
		// 如果多个节点竞争同一个哈希槽时，epoch值更高的节点会抢夺到
		node["configEpoch"] = nodeInfos[6]
		// node-to-node集群总线使用的链接的状态，我们使用这个链接与集群中其他节点进行通信.值可以是 connected 和 disconnected
		node["linkState"] = nodeInfos[7]
		// slave节点没有插槽信息
		if len(nodeInfos) > 8 {
			// slot：master节点第9位为哈希槽值或者一个哈希槽范围，代表当前节点可以提供服务的所有哈希槽值。如果只是一个值,那就是只有一个槽会被使用。
			// 如果是一个范围，这个值表示为起始槽-结束槽，节点将处理包括起始槽和结束槽在内的所有哈希槽。
			node["slot"] = nodeInfos[8]
		}
		nodesRes = append(nodesRes, node)
	}
	rc.ResData = collx.M{
		"clusterInfo":  info,
		"clusterNodes": nodesRes,
	}
}

// 校验查询参数中的key为必填项，并返回redis实例
func (r *Redis) checkKeyAndGetRedisConn(rc *req.Ctx) (*rdm.RedisConn, string) {
	key := rc.GinCtx.Query("key")
	biz.NotEmpty(key, "key不能为空")
	return r.getRedisConn(rc), key
}

func (r *Redis) getRedisConn(rc *req.Ctx) *rdm.RedisConn {
	ri, err := r.RedisApp.GetRedisConn(getIdAndDbNum(rc.GinCtx))
	biz.ErrIsNil(err)
	biz.ErrIsNilAppendErr(r.TagApp.CanAccess(rc.GetLoginAccount().Id, ri.Info.TagPath...), "%s")
	return ri
}

// 获取redis id与要操作的库号（统一路径）
func getIdAndDbNum(g *gin.Context) (uint64, int) {
	return uint64(ginx.PathParamInt(g, "id")), ginx.PathParamInt(g, "db")
}
