package rediscli

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
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
func Set(key string, val string, expiration time.Duration) {
	cli.Set(context.TODO(), key, val, expiration)
}

func Del(key string) {
	cli.Del(context.TODO(), key)
}

func HSet(key string, field string, val interface{}) {
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
