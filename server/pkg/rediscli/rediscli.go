package rediscli

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var cli *redis.Client

func SetCli(client *redis.Client) {
	cli = client
}

func GetCli() *redis.Client {
	return cli
}

// get key value
func Get(key string) (string, error) {
	return cli.Get(context.TODO(), key).Result()
}

// set key value
func Set(key string, val string, expiration time.Duration) error {
	if expiration < 0 {
		expiration = 0
	}
	return cli.Set(context.TODO(), key, val, expiration).Err()
}

// Del 删除key
func Del(key string) (int64, error) {
	return cli.Del(context.TODO(), key).Result()
}

// DelByKeyPrefix 根据key前缀删除key
func DelByKeyPrefix(keyPrefix string) error {
	res, err := cli.Keys(context.TODO(), keyPrefix+"*").Result()
	if err != nil {
		return err
	}
	for _, key := range res {
		Del(key)
	}
	return nil
}

func HSet(key string, field string, val any) {
	cli.HSet(context.TODO(), key, field, val)
}

// hget
func HGet(key string, field string) string {
	val, _ := cli.HGet(context.TODO(), key, field).Result()
	return val
}

// hget
func HExist(key string, field string) bool {
	val, _ := cli.HExists(context.TODO(), key, field).Result()
	return val
}

// hgetall
func HGetAll(key string) map[string]string {
	vals, _ := cli.HGetAll(context.TODO(), key).Result()
	return vals
}

// hdel
func HDel(key string, fields ...string) int {
	return int(cli.HDel(context.TODO(), key, fields...).Val())
}

// CasDel 原子比较并删除：仅当 key 的当前值等于 expected 时才删除。
// 返回 true 表示成功删除（值匹配），false 表示值不匹配或 key 不存在。
// 底层通过 Lua 脚本保证原子性，用于分布式锁的所有权验证释放。
func CasDel(key string, expected string) bool {
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)
	result, err := script.Run(context.Background(), cli, []string{key}, expected).Int64()
	if err != nil {
		return false
	}
	return result > 0
}
