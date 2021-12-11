package api

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/ginx"
	"mayfly-go/base/utils"
	"mayfly-go/server/devops/api/form"
	"mayfly-go/server/devops/api/vo"
	"mayfly-go/server/devops/application"
	"mayfly-go/server/devops/domain/entity"
	"strconv"
	"strings"
	"time"
)

type Redis struct {
	RedisApp application.Redis
}

func (r *Redis) RedisList(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	m := &entity.Redis{EnvId: uint64(ginx.QueryInt(g, "envId", 0)),
		ProjectId: uint64(ginx.QueryInt(g, "projectId", 0)),
	}
	m.CreatorId = rc.LoginAccount.Id
	rc.ResData = r.RedisApp.GetPageList(m, ginx.GetPageParam(rc.GinCtx), new([]vo.Redis))
}

func (r *Redis) Save(rc *ctx.ReqCtx) {
	form := &form.Redis{}
	ginx.BindJsonAndValid(rc.GinCtx, form)

	rc.ReqParam = form

	redis := new(entity.Redis)
	utils.Copy(redis, form)
	redis.SetBaseInfo(rc.LoginAccount)
	r.RedisApp.Save(redis)
}

func (r *Redis) DeleteRedis(rc *ctx.ReqCtx) {
	r.RedisApp.Delete(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}

func (r *Redis) RedisInfo(rc *ctx.ReqCtx) {
	res, _ := r.RedisApp.GetRedisInstance(uint64(ginx.PathParamInt(rc.GinCtx, "id"))).Cli.Info().Result()

	datas := strings.Split(res, "\r\n")
	i := 0
	length := len(datas)

	parseMap := make(map[string]map[string]string)
	for {
		if i >= length {
			break
		}
		if strings.Contains(datas[i], "#") {
			key := utils.SubString(datas[i], strings.Index(datas[i], "#")+1, utils.StrLen(datas[i]))
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

// scan获取redis的key列表信息
func (r *Redis) Scan(rc *ctx.ReqCtx) {
	g := rc.GinCtx

	ri := r.RedisApp.GetRedisInstance(uint64(ginx.PathParamInt(g, "id")))
	keys, cursor := ri.Scan(uint64(ginx.PathParamInt(g, "cursor")), g.Query("match"), int64(ginx.PathParamInt(g, "count")))

	var keyInfoSplit []string
	if len(keys) > 0 {
		keyInfoLua := `
			local result = {}
			-- KEYS[1]为第1个参数，lua数组下标从1开始
			local ttl = redis.call('ttl', KEYS[1]);
			local keyType = redis.call('type', KEYS[1]);
			for i = 1, #KEYS do
				local ttl = redis.call('ttl', KEYS[i]);
				local keyType = redis.call('type', KEYS[i]);
				table.insert(result, string.format("%d,%s", ttl, keyType['ok']));
			end;
			return table.concat(result, ".");`
		// 通过lua获取 ttl,type.ttl2,type2格式，以便下面切割获取ttl和type。避免多次调用ttl和type函数
		keyInfos, _ := ri.Cli.Eval(keyInfoLua, keys).Result()
		keyInfoSplit = strings.Split(keyInfos.(string), ".")
	}

	kis := make([]*vo.KeyInfo, 0)
	for i, k := range keys {
		ttlType := strings.Split(keyInfoSplit[i], ",")
		ttl, _ := strconv.Atoi(ttlType[0])
		ki := &vo.KeyInfo{Key: k, Type: ttlType[1], Ttl: uint64(ttl)}
		kis = append(kis, ki)
	}

	size, _ := ri.Cli.DBSize().Result()
	rc.ResData = &vo.Keys{Cursor: cursor, Keys: kis, DbSize: size}
}

func (r *Redis) DeleteKey(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	key := g.Query("key")
	biz.NotEmpty(key, "key不能为空")

	ri := r.RedisApp.GetRedisInstance(uint64(ginx.PathParamInt(g, "id")))
	rc.ReqParam = key
	ri.Cli.Del(key)
}

func (r *Redis) checkKey(rc *ctx.ReqCtx) (*application.RedisInstance, string) {
	g := rc.GinCtx
	key := g.Query("key")
	biz.NotEmpty(key, "key不能为空")

	return r.RedisApp.GetRedisInstance(uint64(ginx.PathParamInt(g, "id"))), key
}

func (r *Redis) GetStringValue(rc *ctx.ReqCtx) {
	ri, key := r.checkKey(rc)
	str, err := ri.Cli.Get(key).Result()
	biz.ErrIsNilAppendErr(err, "获取字符串值失败: %s")
	rc.ResData = str
}

func (r *Redis) GetHashValue(rc *ctx.ReqCtx) {
	ri, key := r.checkKey(rc)
	res, err := ri.Cli.HGetAll(key).Result()
	biz.ErrIsNilAppendErr(err, "获取hash值失败: %s")
	rc.ResData = res
}

func (r *Redis) GetSetValue(rc *ctx.ReqCtx) {
	ri, key := r.checkKey(rc)
	res, err := ri.Cli.SMembers(key).Result()
	biz.ErrIsNilAppendErr(err, "获取set值失败: %s")
	rc.ResData = res
}

func (r *Redis) SetStringValue(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	keyValue := new(form.KeyValue)
	ginx.BindJsonAndValid(g, keyValue)

	ri := r.RedisApp.GetRedisInstance(uint64(ginx.PathParamInt(g, "id")))
	str, err := ri.Cli.Set(keyValue.Key, keyValue.Value, time.Second*time.Duration(keyValue.Timed)).Result()
	biz.ErrIsNilAppendErr(err, "保存字符串值失败: %s")
	rc.ResData = str
}
