package kfm

import (
	"cmp"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/stringx"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

type KafkaConn struct {
	Id   string
	Info *KafkaInfo

	Configs []kgo.Opt
	Client  *kgo.Client
	Ac      *kadm.Client
}

/******************* pool.Conn impl *******************/

func (kc *KafkaConn) Close() error {
	if kc.Client != nil {
		kc.Client.Close()
		kc.Client = nil
	}

	if kc.Ac != nil {
		kc.Ac.Close()
		kc.Ac = nil
	}
	return nil
}

func (kc *KafkaConn) Ping() error {
	if kc == nil {
		return errorx.NewBiz("kafka connection is nil")
	}

	if kc.Client == nil {
		return errorx.NewBiz("kafka client is nil")
	}
	if kc.Ac == nil {
		return errorx.NewBiz("kafka admin client is nil")
	}

	brokers, err := kc.Ac.ListBrokers(context.Background())
	if err != nil {
		return err
	}
	if len(brokers) == 0 {
		return errorx.NewBiz("no available brokers")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := kc.Client.Ping(ctx); err != nil {
		return err
	}

	return nil
}

func (kc *KafkaConn) GetTopics() ([]string, error) {
	if kc.Ac == nil {
		return nil, errorx.NewBiz("kafka admin client is closed")
	}

	list, err := kc.Ac.ListTopics(context.Background())
	if err != nil {
		return nil, err
	}

	topics := make([]string, 0, len(list))
	for topic := range list {
		topics = append(topics, topic)
	}
	return topics, nil
}

// GetTopicConfig 获取 Topic 配置信息
func (kc *KafkaConn) GetTopicConfig(topic string) (*kadm.ResourceConfigs, error) {
	configs, err := kc.Ac.DescribeTopicConfigs(context.Background(), topic)
	if err != nil {
		return nil, err
	}

	return &configs, nil
}

// GetTopicDetails 获取 Topic 详细信息
func (kc *KafkaConn) GetTopicDetails() ([]any, error) {
	topics, err := kc.Ac.ListTopics(context.Background())
	if err != nil {
		return nil, err
	}
	return buildTopicsResp(topics), nil
}

func buildTopicsResp(topics kadm.TopicDetails) []any {
	// FIX: 对 map 进行排序以保证输出顺序稳定
	topicNames := make([]string, 0, len(topics))
	for name := range topics {
		topicNames = append(topicNames, name)
	}
	sort.Strings(topicNames)

	result := make([]any, 0, len(topicNames))
	for _, topicName := range topicNames {
		topicDetail := topics[topicName]
		partitionErrs := ""
		var partitions []any
		for _, partition := range topicDetail.Partitions {
			errMsg := ""
			if partition.Err != nil {
				errMsg = partition.Err.Error()
				partitionErrs += fmt.Sprintf("partition %d: %s\n", partition.Partition, errMsg)
			}
			partitions = append(partitions, map[string]any{
				"partition":       partition.Partition,
				"leader":          partition.Leader,
				"replicas":        partition.Replicas,
				"isr":             partition.ISR,
				"err":             errMsg,
				"LeaderEpoch":     partition.LeaderEpoch,
				"OfflineReplicas": partition.OfflineReplicas,
			})
		}
		if topicDetail.Err != nil {
			partitionErrs = topicDetail.Err.Error() + "\n" + partitionErrs
		}
		replicationFactor := 0
		if len(topicDetail.Partitions) > 0 {
			replicationFactor = len(topicDetail.Partitions[0].Replicas)
		}
		result = append(result, map[string]any{
			"ID":                 topicDetail.ID,
			"topic":              topicName,
			"partition_count":    len(topicDetail.Partitions),
			"replication_factor": replicationFactor,
			"IsInternal":         topicDetail.IsInternal,
			"Err":                partitionErrs,
			"partitions":         partitions,
		})
	}
	return result
}

// GetConsumerGroups 获取消费者组列表
func (kc *KafkaConn) GetConsumerGroups() ([]kadm.ListedGroup, error) {
	groups, err := kc.Ac.ListGroups(context.Background())
	if err != nil {
		return nil, err
	}

	sortedGroups := groups.Sorted()
	return sortedGroups, nil
}

// CreateTopic 创建 Topic
func (kc *KafkaConn) CreateTopic(p *CreateTopicParam) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := kc.Ac.CreateTopics(ctx, p.NumPartitions, p.ReplicationFactor, p.ConfigEntries, p.TopicName)
	if err != nil {
		return err
	}
	err = resp.Error()
	if err != nil {
		return err
	}

	return nil
}

// DeleteTopic 删除 Topic
func (kc *KafkaConn) DeleteTopic(topic string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := kc.Ac.DeleteTopics(ctx, topic)
	if err != nil {
		return err
	}

	if resp.Error() != nil {
		return resp.Error()
	}

	return nil
}

// ConsumeMessage 消费 Topic 消息
func (kc *KafkaConn) ConsumeMessage(ctx context.Context, param *ConsumeMessageParam) ([]*ConsumeMessageResult, error) {
	param.Number = cmp.Or(param.Number, 10)
	param.PullTimeout = cmp.Or(param.PullTimeout, 10)
	param.Group = cmp.Or(param.Group, "__mayfly-server__"+stringx.RandUUID())
	st := time.Now()

	// 构建消费配置
	consumeOpts := []kgo.Opt{
		kgo.ConsumeTopics(param.Topic),
		kgo.DisableAutoCommit(),
	}

	// 配置隔离级别
	if strings.ToLower(param.IsolationLevel) == "read_committed" {
		consumeOpts = append(consumeOpts, kgo.FetchIsolationLevel(kgo.ReadCommitted()))
	} else {
		consumeOpts = append(consumeOpts, kgo.FetchIsolationLevel(kgo.ReadUncommitted()))
	}

	if param.StartTime != "" {
		parse, err := time.Parse(time.DateTime, param.StartTime)
		if err != nil {
			return nil, err
		}
		consumeOpts = append(consumeOpts, kgo.ConsumeResetOffset(kgo.NewOffset().AfterMilli(parse.UnixMilli())))
	} else if param.Earliest {
		consumeOpts = append(consumeOpts, kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	} else {
		consumeOpts = append(consumeOpts, kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()))
	}

	// 获取或创建带缓存的消费者组客户端
	cl, err := GetOrCreateConsumerGroupClient(ctx, kc.Info.Id, param.Group, kc.Info, consumeOpts)
	if err != nil {
		return nil, err
	}

	// 开始 poll msg
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(param.PullTimeout)*time.Second)
	defer cancel()
	fetches := cl.PollRecords(ctx, param.Number)
	if fetches.IsClientClosed() {
		return nil, errorx.NewBiz("Client Closed, Please Retry")
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		if len(fetches.Records()) == 0 {
			return nil, errorx.NewBiz("Consume Timeout, Maybe No Message")
		}
	}
	if errs := fetches.Errors(); len(errs) > 0 {
		return nil, errorx.NewBiz(fmt.Sprint(errs))
	}

	logx.Infof("poll 完成... %v", len(fetches.Records()))

	res := make([]*ConsumeMessageResult, 0)
	for i, v := range fetches.Records() {
		if v == nil {
			continue
		}
		var data []byte
		var err error
		switch param.Decompression {
		case "gzip":
			data, err = GzipDecompress(v.Value)
		case "lz4":
			data, err = Lz4Decompress(v.Value)
		case "zstd":
			data, err = ZstdDecompress(v.Value)
		case "snappy":
			data, err = SnappyDecompress(v.Value)
		default:
			data = v.Value
		}
		if err != nil {
			return nil, errorx.NewBiz(fmt.Sprintf("failed to decompress data: %s", err.Error()))
		}

		// 根据 decode 参数进行解码
		var decodedData []byte
		switch strings.ToLower(param.Decode) {
		case "base64":
			decodedData, err = base64.StdEncoding.DecodeString(string(data))
			if err != nil {
				return nil, errorx.NewBiz(fmt.Sprintf("Failed to decode base64 data:  %s", err.Error()))
			}
		default:
			decodedData = data
		}

		res = append(res, &ConsumeMessageResult{
			Id:            i,
			Offset:        v.Offset,
			Partition:     v.Partition,
			Key:           string(v.Key),
			Value:         string(decodedData),
			Timestamp:     v.Timestamp.Format(time.DateTime),
			Topic:         v.Topic,
			Headers:       getHeadersString(v.Headers),
			LeaderEpoch:   v.LeaderEpoch,
			ProducerEpoch: v.ProducerEpoch,
			ProducerID:    v.ProducerID,
		})
	}

	logx.Infof("耗时：%.4f秒 , topic: %s, group: %s, num: %v", time.Since(st).Seconds(), param.Topic, param.Group, param.Number)
	if param.Group != "" && param.CommitOffset {
		logx.Infof("开始提交 offset...")
		if err := cl.CommitUncommittedOffsets(context.Background()); err != nil {
			return nil, err
		}
		logx.Infof("提交 offset 完成...")
	}

	return res, nil

}

func getHeadersString(headers []kgo.RecordHeader) map[string]string {
	headersMap := make(map[string]string)

	if len(headers) == 0 {
		return headersMap
	}

	for _, h := range headers {
		headersMap[h.Key] = string(h.Value)
	}

	return headersMap
}

// ProduceMessage 生产消息到 Topic
func (kc *KafkaConn) ProduceMessage(ctx context.Context, param *ProduceMessageParam) error {
	logx.Infof("开始生产消息...")
	st := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	headers2 := make([]kgo.RecordHeader, len(param.Headers))
	for i := 0; i < len(param.Headers); i++ {
		headers2[i] = kgo.RecordHeader{
			Key:   param.Headers[i]["key"],
			Value: []byte(param.Headers[i]["value"]),
		}
	}
	var data []byte
	var err error
	switch param.Compression {
	case "gzip":
		data, err = Gzip([]byte(param.Value))
	case "lz4":
		data, err = Lz4([]byte(param.Value))
	case "zstd":
		data, err = Zstd([]byte(param.Value))
	case "snappy":
		data, err = Snappy([]byte(param.Value))
	default:
		data = []byte(param.Value)
	}
	if err != nil {
		return errorx.NewBiz("Failed to compress data: " + err.Error())
	}
	var records []*kgo.Record
	for i := 0; i < param.Times; i++ {
		records = append(records, &kgo.Record{
			Topic:     param.Topic,
			Value:     data,
			Key:       []byte(param.Key),
			Headers:   headers2,
			Partition: param.Partition,
		})
	}
	res := kc.Client.ProduceSync(ctx, records...)
	if err := res.FirstErr(); err != nil {
		return errorx.NewBiz("Produce Error：" + err.Error())
	}
	logx.Infof("耗时：%.4f秒 topic: %s, key:%s, num:%v", time.Since(st).Seconds(), param.Topic, param.Key, param.Times)
	return nil
}

func (kc *KafkaConn) CreatePartitions(p *CreatePartitionsParam) error {
	res, err := kc.Ac.CreatePartitions(context.Background(), p.NumPartitions, p.TopicName)
	if err != nil {
		return err
	}
	err = res.Error()
	if err != nil {
		return errorx.NewBiz("CreatePartitions Error：" + err.Error())
	}
	return nil
}

func (kc *KafkaConn) GetBrokers() []BrokerInfo {

	brokers, err := kc.Ac.ListBrokers(context.Background())
	if err != nil {
		return nil
	}

	var bs []BrokerInfo
	for _, b := range brokers {
		bs = append(bs, BrokerInfo{
			Id:   b.NodeID,
			Addr: b.Host + ":" + strconv.Itoa(int(b.Port)),
			Rack: b.Rack,
		})
	}
	return bs
}

func (kc *KafkaConn) GetBrokerConfig(id int) (*kadm.ResourceConfigs, error) {

	configs, err := kc.Ac.DescribeBrokerConfigs(context.Background(), int32(id))
	if err != nil {
		return nil, err
	}

	return &configs, nil
}

func (kc *KafkaConn) DeleteGroup(group string) error {

	resp, err := kc.Ac.DeleteGroup(context.Background(), group)
	if err != nil {
		return errorx.NewBiz("DeleteGroup Error：" + err.Error())
	}
	if resp.Err != nil {
		return errorx.NewBiz("DeleteGroup Error：" + resp.Err.Error())
	}
	return err
}

func (kc *KafkaConn) GetGroupMembers(group string) (any, any) {

	ctx := context.Background()
	resp, err := kc.Ac.DescribeGroups(ctx, group)
	if err != nil {
		return nil, errorx.NewBiz("DescribeGroups Error：" + err.Error())
	}
	err = resp.Error()
	if err != nil {
		return nil, errorx.NewBiz("DescribeGroups Error：" + err.Error())
	}
	sortedGroups := resp.Sorted()

	membersLst := make([]any, 0)
	for _, describedGroup := range sortedGroups {
		if describedGroup.Err != nil {
			return nil, errorx.NewBiz(fmt.Sprintf("Error describing group %s: %v", describedGroup.Group, describedGroup.Err))
		}
		for _, member := range describedGroup.Members {
			subscribedTPs := make(map[string][]int32)
			if consumerMetadata, ok := member.Assigned.AsConsumer(); ok {
				tps := consumerMetadata.Topics
				for _, tp := range tps {
					subscribedTPs[tp.Topic] = tp.Partitions
				}
			}
			membersLst = append(membersLst, map[string]any{
				"MemberID":   member.MemberID,
				"InstanceID": member.InstanceID,
				"ClientID":   member.ClientID,
				"ClientHost": member.ClientHost,
				"TPs":        subscribedTPs, // TPs:map[topicName:[0]]]]
			})
		}

		return membersLst, nil
	}

	return membersLst, nil
}
