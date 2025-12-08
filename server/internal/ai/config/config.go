package config

import (
	"cmp"
	sysapp "mayfly-go/internal/sys/application"

	"github.com/spf13/cast"
)

const (
	ConfigKeyAi string = "AiModelConfig"
)

type AIModelConfig struct {
	ModelType   string  `json:"modelType"`
	Model       string  `json:"model"`
	BaseUrl     string  `json:"baseUrl"`
	ApiKey      string  `json:"apiKey"`  // api key
	TimeOut     int     `json:"timeOut"` // 请求超时时间，单位秒
	Temperature float32 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
}

func GetAiModel() *AIModelConfig {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyAi)
	jm := c.GetJsonMap()

	conf := new(AIModelConfig)
	conf.ModelType = cast.ToString(jm["modelType"])
	conf.Model = cast.ToString(jm["model"])
	conf.BaseUrl = cast.ToString(jm["baseUrl"])
	conf.ApiKey = cast.ToString(jm["apiKey"])
	conf.TimeOut = cmp.Or(cast.ToInt(jm["timeOut"]), 60)
	conf.Temperature = cmp.Or(cast.ToFloat32(jm["temperature"]), 0.7)
	conf.MaxTokens = cmp.Or(cast.ToInt(jm["maxTokens"]), 2048)
	return conf
}
