package api

import (
	"mayfly-go/internal/mq/kafka/api/form"
	"mayfly-go/internal/mq/kafka/api/vo"
	"mayfly-go/internal/mq/kafka/application"
	"mayfly-go/internal/mq/kafka/domain/entity"
	"mayfly-go/internal/mq/kafka/imsg"
	"mayfly-go/internal/mq/kafka/kfm"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/spf13/cast"
)

type Kafka struct {
	kafkaApp   application.Kafka    `inject:"T"`
	tagTreeApp tagapp.TagTreeReader `inject:"T"`
}

func (k *Kafka) ReqConfs() *req.Confs {
	createTopicPerm := req.NewPermission("kafka:topic:create")
	deleteTopicPerm := req.NewPermission("kafka:topic:delete")
	deleteGroupPerm := req.NewPermission("kafka:group:delete")

	reqs := [...]*req.Conf{
		// 获取所有kafka列表
		req.NewGet("", k.Kafkas),

		req.NewPost("/test-conn", k.TestConn),

		req.NewPost("", k.Save).Log(req.NewLogSaveI(imsg.LogKafkaSave)),

		req.NewDelete(":id", k.DeleteById).Log(req.NewLogSaveI(imsg.LogKafkaDelete)),

		req.NewGet(":id/getTopics", k.GetTopics),
		//CreateTopics 创建主题
		req.NewPost(":id/createTopic", k.CreateTopic).RequiredPermission(createTopicPerm).Log(req.NewLogSaveI(imsg.LogKafkaCreateTopic)),

		req.NewDelete(":id/:topic/deleteTopic", k.DeleteTopic).RequiredPermission(deleteTopicPerm).Log(req.NewLogSaveI(imsg.LogKafkaDeleteTopic)),

		req.NewGet(":id/:topic/getTopicConfig", k.GetTopicConfig),
		// CreatePartitions 添加分区
		req.NewPost(":id/createPartitions", k.CreatePartitions),
		req.NewPost(":id/:topic/produce", k.Produce),
		req.NewPost(":id/:topic/consume", k.Consume),

		// 获取集群信息
		req.NewGet(":id/getBrokers", k.GetBrokers),

		// GetBrokerConfig 获取Broker配置
		req.NewGet(":id/getBrokerConfig/:brokerId", k.GetBrokerConfig),

		// GetGroups 获取消费组信息
		req.NewGet(":id/getGroups", k.GetGroups),
		req.NewGet(":id/getGroupMembers/:group", k.GetGroupMembers),
		req.NewDelete(":id/deleteGroup/:group", k.DeleteGroup).RequiredPermission(deleteGroupPerm).Log(req.NewLogSaveI(imsg.LogKafkaDeleteGroup)),
	}

	return req.NewConfs("mq/kafka", reqs[:]...)
}

func (k *Kafka) Kafkas(rc *req.Ctx) {
	queryCond := rc.BindQuery[entity.KafkaQuery]()

	// 不存在可访问标签id，即没有可操作数据
	tags := k.tagTreeApp.GetAccountTags(rc.GetLoginAccount().Id, &tagentity.TagTreeQuery{
		TypePaths:     collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeMqKafka)),
		CodePathLikes: []string{queryCond.TagPath},
	})
	if len(tags) == 0 {
		rc.ResData = model.NewEmptyPageResult[any]()
		return
	}
	queryCond.Codes = tags.GetCodes()

	res, err := k.kafkaApp.GetPageList(queryCond)
	biz.ErrIsNil(err)
	resVo := model.PageResultConv[*entity.Kafka, *vo.Kafka](res)

	rc.ResData = resVo
}

func (k *Kafka) TestConn(rc *req.Ctx) {
	_, kafka := rc.BindJsonAndCopyTo[form.Kafka, entity.Kafka]()
	biz.ErrIsNilAppendErr(k.kafkaApp.TestConn(kafka), "connection error: %s")
}

func (k *Kafka) Save(rc *req.Ctx) {
	f, kafka := rc.BindJsonAndCopyTo[form.Kafka, entity.Kafka]()

	// 密码脱敏记录日志
	f.Password = func(str *string) *string {
		str1 := "***"
		return &str1
	}(f.Password)

	rc.ReqParam = f

	biz.ErrIsNil(k.kafkaApp.SaveKafka(rc.MetaCtx, kafka, f.TagCodePaths...))
}

func (k *Kafka) DeleteById(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		k.kafkaApp.Delete(rc.MetaCtx, cast.ToUint64(v))
	}
}

func (k *Kafka) GetTopics(rc *req.Ctx) {
	id := k.GetKafkaId(rc)

	conn, err := k.kafkaApp.GetKafkaConn(rc, id)
	if err != nil {
		rc.Error = err
		return
	}

	rc.ResData, rc.Error = conn.GetTopicDetails()

}
func (k *Kafka) CreateTopic(rc *req.Ctx) {
	id := k.GetKafkaId(rc)

	param := rc.BindJson[kfm.CreateTopicParam]()
	conn, err := k.kafkaApp.GetKafkaConn(rc, id)
	if err != nil {
		rc.Error = err
		return
	}
	rc.Error = conn.CreateTopic(param)

}
func (k *Kafka) DeleteTopic(rc *req.Ctx) {
	id := k.GetKafkaId(rc)
	topic := rc.PathParam("topic")
	conn, err := k.kafkaApp.GetKafkaConn(rc, id)
	if err != nil {
		rc.Error = err
		return
	}
	rc.Error = conn.DeleteTopic(topic)
}

func (k *Kafka) GetTopicConfig(rc *req.Ctx) {
	id := k.GetKafkaId(rc)
	topic := rc.PathParam("topic")
	conn, err := k.kafkaApp.GetKafkaConn(rc, id)
	if err != nil {
		rc.Error = err
		return
	}
	rc.ResData, rc.Error = conn.GetTopicConfig(topic)
}
func (k *Kafka) CreatePartitions(rc *req.Ctx) {
	id := k.GetKafkaId(rc)
	param := rc.BindJson[kfm.CreatePartitionsParam]()

	conn, err := k.kafkaApp.GetKafkaConn(rc, id)
	if err != nil {
		rc.Error = err
		return
	}
	rc.Error = conn.CreatePartitions(param)

}
func (k *Kafka) Produce(rc *req.Ctx) {
	id := k.GetKafkaId(rc)
	topic := rc.PathParam("topic")

	param := rc.BindJson[kfm.ProduceMessageParam]()
	param.Topic = topic

	conn, err := k.kafkaApp.GetKafkaConn(rc, id)
	if err != nil {
		rc.Error = err
		return
	}

	rc.Error = conn.ProduceMessage(rc.MetaCtx, param)

}
func (k *Kafka) Consume(rc *req.Ctx) {
	id := k.GetKafkaId(rc)
	topic := rc.PathParam("topic")

	conn, err := k.kafkaApp.GetKafkaConn(rc, id)
	if err != nil {
		rc.Error = err
		return
	}

	param := rc.BindJson[kfm.ConsumeMessageParam]()
	param.Topic = topic

	rc.ResData, rc.Error = conn.ConsumeMessage(rc.MetaCtx, param)
}
func (k *Kafka) GetBrokers(rc *req.Ctx) {
	id := k.GetKafkaId(rc)
	conn, err := k.kafkaApp.GetKafkaConn(rc, id)
	if err != nil {
		rc.Error = err
		return
	}
	rc.ResData = conn.GetBrokers()
}
func (k *Kafka) GetBrokerConfig(rc *req.Ctx) {
	id := k.GetKafkaId(rc)
	brokerId := rc.PathParamInt("brokerId")
	conn, err := k.kafkaApp.GetKafkaConn(rc, id)
	if err != nil {
		rc.Error = err
		return
	}
	rc.ResData, rc.Error = conn.GetBrokerConfig(brokerId)
}

func (k *Kafka) GetGroups(rc *req.Ctx) {
	id := k.GetKafkaId(rc)
	conn, err := k.kafkaApp.GetKafkaConn(rc, id)
	if err != nil {
		rc.Error = err
		return
	}
	rc.ResData, rc.Error = conn.GetConsumerGroups()
}
func (k *Kafka) GetGroupMembers(rc *req.Ctx) {
	id := k.GetKafkaId(rc)
	group := rc.PathParam("group")
	conn, err := k.kafkaApp.GetKafkaConn(rc, id)
	if err != nil {
		rc.Error = err
		return
	}
	rc.ResData, rc.Error = conn.GetGroupMembers(group)
}
func (k *Kafka) DeleteGroup(rc *req.Ctx) {
	id := k.GetKafkaId(rc)
	group := rc.PathParam("group")
	conn, err := k.kafkaApp.GetKafkaConn(rc, id)
	if err != nil {
		rc.Error = err
		return
	}
	rc.Error = conn.DeleteGroup(group)
}

// 获取请求路径上的kafka id
func (k *Kafka) GetKafkaId(rc *req.Ctx) uint64 {
	dbId := rc.PathParamInt("id")
	biz.IsTrue(dbId > 0, "kafkaId error")
	return uint64(dbId)
}
