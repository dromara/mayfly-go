package kfm

import (
	"context"
	"fmt"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/pool"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

var (
	poolGroup = pool.NewPoolGroup[*KafkaConn]()

	// consumerGroup 客户端缓存，key: kafkaId:group, value: *kgo.Client
	consumerGroupClients = make(map[string]*kgo.Client)
	cgClientMu           sync.RWMutex
	cgClientCleanupDelay = 5 * time.Minute // 空闲 5 分钟后清除
)

// 从缓存中获取 kafka 连接信息，若缓存中不存在则会使用回调函数获取 kafkaInfo 进行连接并缓存
func GetKafkaConn(ctx context.Context, kafkaId uint64, getKafkaInfo func() (*KafkaInfo, error)) (*KafkaConn, error) {
	cachePool, err := poolGroup.GetCachePool(getConnId(kafkaId), func() (*KafkaConn, error) {
		// 若缓存中不存在，则从回调函数中获取 KafkaInfo
		mi, err := getKafkaInfo()
		if err != nil {
			return nil, err
		}

		// 连接 kafka
		return mi.Conn()
	})

	if err != nil {
		return nil, err
	}
	// 从连接池中获取一个可用的连接
	return cachePool.Get(ctx)
}

// 获取或创建带 group 的消费者客户端（带缓存）
func GetOrCreateConsumerGroupClient(ctx context.Context, kafkaId uint64, group string, info *KafkaInfo, consumeOpts []kgo.Opt) (*kgo.Client, error) {
	cacheKey := getConsumerGroupCacheKey(kafkaId, group)

	cgClientMu.RLock()
	if cl, exists := consumerGroupClients[cacheKey]; exists {
		cgClientMu.RUnlock()
		return cl, nil
	}
	cgClientMu.RUnlock()

	cgClientMu.Lock()
	defer cgClientMu.Unlock()

	// double check
	if cl, exists := consumerGroupClients[cacheKey]; exists {
		return cl, nil
	}

	// 创建新客户端
	// 构建基础配置
	opts := info.BuildBaseOpts()
	opts = append(opts, kgo.ConsumerGroup(group))
	opts = append(opts, consumeOpts...)

	cl, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	// 缓存客户端
	consumerGroupClients[cacheKey] = cl

	// 设置定时清理
	gox.Go(func() { scheduleConsumerGroupCleanup(cacheKey, cl) })

	logx.Debugf("create consumer group client: %s", cacheKey)
	return cl, nil
}

// 安排消费者组客户端清理
func scheduleConsumerGroupCleanup(cacheKey string, cl *kgo.Client) {
	time.Sleep(cgClientCleanupDelay)

	cgClientMu.Lock()
	defer cgClientMu.Unlock()

	// 检查是否仍为同一个客户端
	if cachedCl, exists := consumerGroupClients[cacheKey]; exists && cachedCl == cl {
		cl.Close()
		delete(consumerGroupClients, cacheKey)
		logx.Debugf("cleanup idle consumer group client: %s", cacheKey)
	}
}

// 生成消费者组缓存 key
func getConsumerGroupCacheKey(kafkaId uint64, group string) string {
	return fmt.Sprintf("%s:group:%s", getConnId(kafkaId), group)
}

// 关闭连接，并移除缓存连接
func CloseConn(kafkaId uint64) {
	err := poolGroup.Close(getConnId(kafkaId))
	if err != nil {
		logx.Errorf("关闭kafka连接失败：%v", err)
		return
	}
}
