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
