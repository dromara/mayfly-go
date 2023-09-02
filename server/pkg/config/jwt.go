package config

import (
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/assert"
	"mayfly-go/pkg/utils/stringx"
)

type Jwt struct {
	Key        string `yaml:"key"`
	ExpireTime uint64 `yaml:"expire-time"` // 过期时间，单位分钟
}

func (j *Jwt) Default() {
	if j.Key == "" {
		// 如果配置文件中的jwt key为空，则随机生成字符串
		j.Key = stringx.Rand(32)
		logx.Warnf("未配置jwt.key, 随机生成key为: %s", j.Key)
	}

	if j.ExpireTime == 0 {
		j.ExpireTime = 1440
		logx.Warnf("未配置jwt.expire-time, 默认值: %d", j.ExpireTime)
	}
}

func (j *Jwt) Valid() {
	assert.IsTrue(j.ExpireTime != 0, "config.yml之[jwt.expire-time] 不能为空")
}
