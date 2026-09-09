package cache

import (
	"errors"
	"fmt"
	global_cache "mayfly-go/pkg/cache"
	"time"
)

const AccountTagsKey = "mayfly:tag:account:%d"

// 资源标签路径短缓存 key：CanAccessByCode 每次资源操作鉴权都会查资源标签路径，
// 加短 TTL 缓存减少 DB 往返；资源标签变更时须主动失效
const ResourceTagPathsKey = "mayfly:tag:resource-paths:%d:%s"

func SaveAccountTagPaths(accountId uint64, tags []string) error {
	return global_cache.Set(fmt.Sprintf(AccountTagsKey, accountId), tags, 2*time.Minute)
}

func GetAccountTagPaths(accountId uint64) ([]string, error) {
	var res []string
	if !global_cache.Get(fmt.Sprintf(AccountTagsKey, accountId), &res) {
		return nil, errors.New("不存在该值")
	}
	return res, nil
}

func DelAccountTagPaths(accountId uint64) {
	global_cache.Del(fmt.Sprintf(AccountTagsKey, accountId))
}

func SaveResourceTagPaths(resourceType int8, resourceCode string, paths []string) error {
	return global_cache.Set(fmt.Sprintf(ResourceTagPathsKey, resourceType, resourceCode), paths, 30*time.Second)
}

func GetResourceTagPaths(resourceType int8, resourceCode string) ([]string, error) {
	var res []string
	if !global_cache.Get(fmt.Sprintf(ResourceTagPathsKey, resourceType, resourceCode), &res) {
		return nil, errors.New("不存在该值")
	}
	return res, nil
}

func DelResourceTagPaths(resourceType int8, resourceCode string) {
	global_cache.Del(fmt.Sprintf(ResourceTagPathsKey, resourceType, resourceCode))
}
