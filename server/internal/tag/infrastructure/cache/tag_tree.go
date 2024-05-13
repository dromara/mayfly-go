package cache

import (
	"errors"
	"fmt"
	global_cache "mayfly-go/pkg/cache"
	"time"
)

const AccountTagsKey = "mayfly:tag:account:%d"

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
