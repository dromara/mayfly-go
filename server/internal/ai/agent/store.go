package agent

import (
	"context"
	"encoding/base64"
	"mayfly-go/pkg/cache"
	"sync"
	"time"

	"github.com/cloudwego/eino/compose"
)

type CheckPointStore interface {
	compose.CheckPointStore

	Delete(ctx context.Context, key string) error
}

// checkpointTTL checkpoint 存活时长。
// 中断挂起超过该时长后 checkpoint 失效，恢复时会返回友好提示（见 Agent.Run 的预检查）。
const checkpointTTL = 24 * time.Hour

var (
	checkPointStore   CheckPointStore
	checkPointOnce    sync.Once
	checkPointInitErr error
)

// GetDefaultCheckPointStore 获取默认的 checkpoint 存储单例（并发安全）
func GetDefaultCheckPointStore() (CheckPointStore, error) {
	checkPointOnce.Do(func() {
		checkPointStore = NewCheckPointStore()
	})
	return checkPointStore, checkPointInitErr
}

func NewCheckPointStore() CheckPointStore {
	return &cacheStore{}
}

type cacheStore struct {
}

var _ CheckPointStore = (*cacheStore)(nil)

func (i *cacheStore) Set(ctx context.Context, key string, value []byte) error {
	return cache.Set(key, base64.StdEncoding.EncodeToString(value), checkpointTTL)
}

func (i *cacheStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	encoded := cache.GetStr(key)
	if encoded == "" {
		return nil, false, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, false, err
	}
	return decoded, true, nil
}

func (i *cacheStore) Delete(ctx context.Context, key string) error {
	cache.Del(key)
	return nil
}
