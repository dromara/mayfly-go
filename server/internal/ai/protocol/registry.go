package protocol

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"mayfly-go/internal/ai/config"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"

	"github.com/cloudwego/eino/components/model"
)

func init() {
	Register(new(Openai))
}

var (
	protocolMap collx.SM[string, Protocol]
	chatModels  collx.SM[string, model.AgenticModel]
)

// Register 注册协议
func Register(protocol Protocol) {
	protocolMap.Store(protocol.Name(), protocol)
}

// Get 获取协议
func Get(name string) Protocol {
	protocol, _ := protocolMap.Load(name)
	return protocol
}

// GetChatModel 获取Chat模型
func GetChatModel(ctx context.Context, modelConfig *config.ModelConfig) (model.AgenticModel, error) {
	modelSpec := modelConfig.GetModelSpec()
	modelProtocol := Get(modelSpec.Protocol)
	if modelProtocol == nil {
		return nil, errors.New("no supported AI model protocol found")
	}

	cacheKey := generateCacheKey(modelConfig)
	if chatModel, ok := chatModels.Load(cacheKey); ok {
		logx.Debugf("ai model [%s/%s] - get chat model from cache", modelSpec.Protocol, modelSpec.Model)
		return chatModel, nil
	}

	// 按 modelConfig 生成的 key 缓存，不同配置共存，同一配置复用实例
	// （无容量限制：模型配置数量有限，无需淘汰）
	chatModel, err := modelProtocol.NewChatModel(ctx, modelConfig)
	if err != nil {
		return nil, err
	}
	logx.Debugf("ai model [%s/%s] - new chat model", modelSpec.Protocol, modelSpec.Model)
	chatModels.Store(cacheKey, chatModel)
	return chatModel, nil
}

// generateCacheKey 生成基于 modelConfig 关键字段的缓存键
func generateCacheKey(config *config.ModelConfig) string {
	// 序列化为 JSON 确保所有字段都被包含，且顺序一致
	data, err := json.Marshal(config)
	if err != nil {
		// 出错时 fallback 到简单 ID
		return config.Model
	}
	// 使用 MD5 缩短 Key 长度，便于存储和比较
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}
