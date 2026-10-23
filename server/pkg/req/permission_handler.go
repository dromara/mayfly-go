package req

import (
	"encoding/json"
	"fmt"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/rediscli"
	"mayfly-go/pkg/utils/anyx"
	"strings"
	"sync"
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
	permissionCodeRegistryMu sync.RWMutex
	permissionCodeRegistry   PermissionCodeRegistry
)

func PermissionHandler(rc *Ctx) error {
	var permission *Permission
	if rc.Conf != nil {
		permission = rc.Conf.requiredPermission
	}
	// 如果需要的权限信息不为空，并且不需要token，则不返回错误，继续后续逻辑
	if permission != nil && !permission.NeedToken {
		return nil
	}
	tokenStr := rc.GetHeader("Authorization")
	// 删除前缀 Bearer, 以支持 Bearer Token
	tokenStr, _ = strings.CutPrefix(tokenStr, "Bearer ")
	// header不存在则从查询参数token中获取
	if tokenStr == "" {
		tokenStr = rc.Query("token")
	}
	if tokenStr == "" {
		return errorx.ErrPermission
	}
	userId, userName, err := ParseToken(tokenStr)
	if err != nil || userId == 0 {
		return errorx.ErrInvalidToken
	}
	// 权限不为nil，并且permission code不为空，则校验是否有权限code
	if permission != nil && permission.Code != "" {
		if !getPermissionCodeRegistry().HasCode(userId, permission.Code) {
			return errorx.ErrPermission
		}
	}
	rc.MetaCtx = contextx.WithLoginAccount(rc.MetaCtx, &model.LoginAccount{
		Id:       userId,
		Username: userName,
		Token:    tokenStr,
	})
	return nil
}

// 保存用户权限code
func SavePermissionCodes(userId uint64, codes []string) {
	getPermissionCodeRegistry().SaveCodes(userId, codes)
}

// 删除用户权限code
func DeletePermissionCodes(userId uint64) {
	getPermissionCodeRegistry().Remove(userId)
}

// 设置权限code注册器
func SetPermissionCodeRegistery(pcr PermissionCodeRegistry) {
	permissionCodeRegistryMu.Lock()
	permissionCodeRegistry = pcr
	permissionCodeRegistryMu.Unlock()
}

func GetPermissionCodeRegistery() PermissionCodeRegistry {
	return getPermissionCodeRegistry()
}

// getPermissionCodeRegistry 惰性选择注册器：未配置 Redis 时用进程内缓存，否则用 Redis。
// 用读写锁保护首次选择，避免并发请求同时改写注册器指针（原实现是裸 nil 判断）。
func getPermissionCodeRegistry() PermissionCodeRegistry {
	permissionCodeRegistryMu.RLock()
	registry := permissionCodeRegistry
	permissionCodeRegistryMu.RUnlock()
	if registry != nil {
		return registry
	}

	permissionCodeRegistryMu.Lock()
	defer permissionCodeRegistryMu.Unlock()
	if permissionCodeRegistry == nil {
		if rediscli.GetCli() == nil {
			permissionCodeRegistry = new(DefaultPermissionCodeRegistry)
		} else {
			permissionCodeRegistry = new(RedisPermissionCodeRegistry)
		}
	}
	return permissionCodeRegistry
}

type PermissionCodeRegistry interface {
	// 保存用户权限code
	SaveCodes(userId uint64, codes []string)

	// 判断用户是否拥有该code的权限
	HasCode(userId uint64, code string) bool

	Remove(userId uint64)
}

// PermissionCodeLoader 从持久层加载用户权限码；ok=false 表示加载失败（按无权限处理）。
// 由业务模块在启动时通过 SetPermissionCodeLoader 注入，避免 pkg/req 反向依赖业务层。
type PermissionCodeLoader func(userId uint64) (codes []string, ok bool)

var (
	permissionCodeLoaderMu sync.RWMutex
	permissionCodeLoader   PermissionCodeLoader
)

// SetPermissionCodeLoader 注册权限码回源加载器。
func SetPermissionCodeLoader(loader PermissionCodeLoader) {
	permissionCodeLoaderMu.Lock()
	permissionCodeLoader = loader
	permissionCodeLoaderMu.Unlock()
}

// loadPermissionCodes 在缓存/Redis 未命中时回源加载权限码。
func loadPermissionCodes(userId uint64) ([]string, bool) {
	permissionCodeLoaderMu.RLock()
	loader := permissionCodeLoader
	permissionCodeLoaderMu.RUnlock()
	if loader == nil {
		return nil, false
	}
	return loader(userId)
}

// hasPermissionCode 判断权限码集合是否包含目标 code。
func hasPermissionCode(codes []string, code string) bool {
	for _, v := range codes {
		if v == code {
			return true
		}
	}
	return false
}

type DefaultPermissionCodeRegistry struct {
	cacheOnce sync.Once
	cache     *cache.TimedCache
}

func (r *DefaultPermissionCodeRegistry) ensureCache() *cache.TimedCache {
	r.cacheOnce.Do(func() {
		r.cache = cache.NewTimedCache(time.Minute*time.Duration(jwtConf.ExpireTime), 5*time.Second)
	})
	return r.cache
}

func (r *DefaultPermissionCodeRegistry) SaveCodes(userId uint64, codes []string) {
	r.ensureCache().Put(fmt.Sprintf("%v", userId), codes)
}

func (r *DefaultPermissionCodeRegistry) HasCode(userId uint64, code string) bool {
	key := fmt.Sprintf("%v", userId)
	if cached, found := r.ensureCache().Get(key); found {
		codes, ok := cached.([]string)
		return ok && hasPermissionCode(codes, code)
	}
	// 进程内缓存是纯内存的：后端重启后旧 token 仍有效但缓存为空。
	// 未命中时回源加载并回填，避免把「缓存冷」误判成「无权限」。
	codes, ok := loadPermissionCodes(userId)
	if !ok {
		return false
	}
	// 空权限集也要回填：有效账号但无任何按钮权限时，不能每次请求都回源查库
	r.SaveCodes(userId, codes)
	return hasPermissionCode(codes, code)
}

func (r *DefaultPermissionCodeRegistry) Remove(userId uint64) {
	r.ensureCache().Delete(fmt.Sprintf("%v", userId))
}

type RedisPermissionCodeRegistry struct {
}

func (r *RedisPermissionCodeRegistry) SaveCodes(userId uint64, codes []string) {
	rediscli.Set(fmt.Sprintf("mayfly:%v:codes", userId), anyx.ToString(codes), time.Minute*time.Duration(jwtConf.ExpireTime))
}

func (r *RedisPermissionCodeRegistry) HasCode(userId uint64, code string) bool {
	key := fmt.Sprintf("mayfly:%v:codes", userId)
	str, err := rediscli.Get(key)
	if err == nil && str != "" {
		var codes []string
		if json.Unmarshal([]byte(str), &codes) == nil {
			// 键存在且内容合法：命中即结论，不再回源（真正的无权限要快速拒绝）
			return hasPermissionCode(codes, code)
		}
	}
	// 键缺失/读取失败/内容损坏：回源加载并回填（Redis 重启或过期后同样不能误判无权限）
	codes, ok := loadPermissionCodes(userId)
	if !ok {
		return false
	}
	r.SaveCodes(userId, codes)
	return hasPermissionCode(codes, code)
}

func (r *RedisPermissionCodeRegistry) Remove(userId uint64) {
	rediscli.Del(fmt.Sprintf("mayfly:%v:codes", userId))
}
