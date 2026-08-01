import Api from '@/common/Api';
import type { PageResult } from '@/types/common';
import type { Kafka, KafkaTopicDetail, KafkaResourceConfigs, KafkaBroker, KafkaGroup, KafkaGroupMember, KafkaMessage, KafkaListParam } from './types';

export const mqApi = {
    kafkaList: Api.newGet<PageResult<Kafka>, KafkaListParam>('/mq/kafka'),
    KafkaTestConn: Api.newPost<void>('/mq/kafka/test-conn'),
    kafkaSave: Api.newPost<void>('/mq/kafka'),
    kafkaDel: Api.newDelete<void>('/mq/kafka/{id}'),
    kafkaGetPwd: Api.newGet<Record<string, string>>('/mq/kafka/{id}/pwd'),
    kafkaTopicList: Api.newGet<KafkaTopicDetail[]>('/mq/kafka/{id}/getTopics'),
    kafkaTopicCreate: Api.newPost<void>('/mq/kafka/{id}/createTopic'),
    kafkaTopicInfo: Api.newGet<KafkaResourceConfigs[]>('/mq/kafka/{id}/{topic}/getTopicConfig'),
    kafkaTopicDelete: Api.newDelete<void>('/mq/kafka/{id}/{topic}/deleteTopic'),
    kafkaTopicCreatePartitions: Api.newPost<void>('/mq/kafka/{id}/createPartitions'),
    kafkaTopicProduce: Api.newPost<void>('/mq/kafka/{id}/{topic}/produce'),
    kafkaTopicConsume: Api.newPost<KafkaMessage[]>('/mq/kafka/{id}/{topic}/consume'),
    kafkaTopicBrokers: Api.newGet<KafkaBroker[]>('/mq/kafka/{id}/getBrokers'),
    kafkaTopicBrokerConfig: Api.newGet<KafkaResourceConfigs[]>('/mq/kafka/{id}/getBrokerConfig/{brokerId}'),
    kafkaGetGroups: Api.newGet<KafkaGroup[]>('/mq/kafka/{id}/getGroups'),
    kafkaDeleteGroup: Api.newDelete<void>('/mq/kafka/{id}/deleteGroup/{group}'),
    kafkaGetGroupMembers: Api.newGet<KafkaGroupMember[]>('/mq/kafka/{id}/getGroupMembers/{group}'),
};
