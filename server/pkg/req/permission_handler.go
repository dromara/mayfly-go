package req

import (
	"encoding/json"
	"fmt"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/rediscli"
	"mayfly-go/pkg/utils/anyx"
	"strings"
	"time"
)

type Permission struct {
	NeedToken bool   // 是否需要token
	Code      string // 权限code
}

func NewPermission(code string) *Permission {
	return &Permission{NeedToken: true, Code: code}
}

var (
	permissionCodeRegistry PermissionCodeRegistry
)

func PermissionHandler(rc *Ctx) error {
	if permissionCodeRegistry == nil {
		if rediscli.GetCli() == nil {
			permissionCodeRegistry = new(DefaultPermissionCodeRegistry)
		} else {
			permissionCodeRegistry = new(RedisPermissionCodeRegistry)
		}
	}

	var permission *Permission
	if rc.Conf != nil {
		permission = rc.Conf.requiredPermission
	}
	// 如果需要的权限信息不为空，并且不需要token，则不返回错误，继续后续逻辑
	if permission != nil && !permission.NeedToken {
		return nil
	}
	tokenStr := rc.GinCtx.Request.Header.Get("Authorization")
	// 删除前缀 Bearer, 以支持 Bearer Token
	tokenStr, _ = strings.CutPrefix(tokenStr, "Bearer ")
	// header不存在则从查询参数token中获取
	if tokenStr == "" {
		tokenStr = rc.GinCtx.Query("token")
	}
	if tokenStr == "" {
		return errorx.PermissionErr
	}
	userId, userName, err := ParseToken(tokenStr)
	if err != nil || userId == 0 {
		return errorx.PermissionErr
	}
	// 权限不为nil，并且permission code不为空，则校验是否有权限code
	if permission != nil && permission.Code != "" {
		if !permissionCodeRegistry.HasCode(userId, permission.Code) {
			return errorx.PermissionErr
		}
	}
	rc.MetaCtx = contextx.WithLoginAccount(rc.MetaCtx, &model.LoginAccount{
		Id:       userId,
		Username: userName,
	})
	return nil
}

// 保存用户权限code
func SavePermissionCodes(userId uint64, codes []string) {
	permissionCodeRegistry.SaveCodes(userId, codes)
}

// 删除用户权限code
func DeletePermissionCodes(userId uint64) {
	permissionCodeRegistry.Remove(userId)
}

// 设置权限code注册器
func SetPermissionCodeRegistery(pcr PermissionCodeRegistry) {
	permissionCodeRegistry = pcr
}

func GetPermissionCodeRegistery() PermissionCodeRegistry {
	return permissionCodeRegistry
}

type PermissionCodeRegistry interface {
	// 保存用户权限code
	SaveCodes(userId uint64, codes []string)

	// 判断用户是否拥有该code的权限
	HasCode(userId uint64, code string) bool

	Remove(userId uint64)
}

type DefaultPermissionCodeRegistry struct {
	cache *cache.TimedCache
}

func (r *DefaultPermissionCodeRegistry) SaveCodes(userId uint64, codes []string) {
	if r.cache == nil {
		r.cache = cache.NewTimedCache(time.Minute*time.Duration(config.Conf.Jwt.ExpireTime), 5*time.Second)
	}
	r.cache.Put(fmt.Sprintf("%v", userId), codes)
}

func (r *DefaultPermissionCodeRegistry) HasCode(userId uint64, code string) bool {
	if r.cache == nil {
		return false
	}
	codes, found := r.cache.Get(fmt.Sprintf("%v", userId))
	if !found {
		return false
	}
	for _, v := range codes.([]string) {
		if v == code {
			return true
		}
	}
	return false
}

func (r *DefaultPermissionCodeRegistry) Remove(userId uint64) {
	if r.cache != nil {
		r.cache.Delete(fmt.Sprintf("%v", userId))
	}
}

type RedisPermissionCodeRegistry struct {
}

func (r *RedisPermissionCodeRegistry) SaveCodes(userId uint64, codes []string) {
	rediscli.Set(fmt.Sprintf("mayfly:%v:codes", userId), anyx.ToString(codes), time.Minute*time.Duration(config.Conf.Jwt.ExpireTime))
}

func (r *RedisPermissionCodeRegistry) HasCode(userId uint64, code string) bool {
	str, err := rediscli.Get(fmt.Sprintf("mayfly:%v:codes", userId))
	if err != nil || str == "" {
		return false
	}

	var codes []string
	_ = json.Unmarshal([]byte(str), &codes)
	for _, v := range codes {
		if v == code {
			return true
		}
	}
	return false
}

func (r *RedisPermissionCodeRegistry) Remove(userId uint64) {
	rediscli.Del(fmt.Sprintf("mayfly:%v:codes", userId))
}
