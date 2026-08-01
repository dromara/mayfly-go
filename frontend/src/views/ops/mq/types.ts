/**
 * MQ 模块类型定义
 * 对应后端: mq/kafka/domain/entity/kafka.go、mq/kafka/kfm/*
 */
import type { BaseModel, PageParam } from '@/types/common';

/** Kafka 实体 (对应 entity.Kafka) */
export interface Kafka extends BaseModel {
    id: number;
    code: string;
    name: string;
    hosts: string;
    username?: string;
    password?: string;
    sshTunnelMachineId: number;
    saslMechanism?: string;
    remark?: string;
}

/** Kafka 列表查询参数 */
export interface KafkaListParam extends PageParam {
    tagPath?: string;
}

/**
 * 后端 GetTopicDetails 返回的 Topic 原始详情
 * （字段名对应 kfm.buildTopicsResp，混合大小写）
 */
export interface KafkaTopicDetail {
    ID: string;
    topic: string;
    partition_count: number;
    replication_factor: number;
    IsInternal: boolean;
    Err: string;
    partitions: KafkaTopicPartition[];
}

/** Topic 分区信息（字段名对应 kfm.buildTopicsResp 的 partitions 项） */
export interface KafkaTopicPartition {
    partition: number;
    leader: number;
    replicas: number[];
    isr: number[];
    err: string;
    LeaderEpoch: number;
    OfflineReplicas: number[] | null;
}

/** 前端转换后的 Topic 视图数据（用于列表展示） */
export interface KafkaTopicView {
    name: string;
    partitionCount: number;
    replicationFactor: number;
    status: string;
    isInternal: boolean;
    partitions: KafkaTopicPartition[];
}

/** 资源配置项（对应 franz-go kadm.Config，无 json tag，字段大写） */
export interface KafkaConfigEntry {
    Key: string;
    Value: string;
    Source: number;
    Sensitive: boolean;
    ReadOnly?: boolean;
    Default?: boolean;
}

/** 资源配置（对应 franz-go kadm.ResourceConfigs，无 json tag） */
export interface KafkaResourceConfigs {
    Name?: string;
    Configs: KafkaConfigEntry[];
}

/** Broker 信息（对应 kfm.BrokerInfo，注意 rack 的 json tag 为 rac） */
export interface KafkaBroker {
    id: number;
    addr: string;
    rac: string | null;
}

/** 消费组（对应 franz-go kadm.ListedGroup，无 json tag，字段大写） */
export interface KafkaGroup {
    Coordinator: number;
    Group: string;
    State: string;
    ProtocolType: string;
    Protocol: string;
}

/** 消费组成员（字段名对应 kfm.GetGroupMembers） */
export interface KafkaGroupMember {
    MemberID: string;
    InstanceID: string | null;
    ClientID: string;
    ClientHost: string;
    TPs: Record<string, number[]>;
}

/** Kafka 消息（对应 kfm.ConsumeMessageResult） */
export interface KafkaMessage {
    topic: string;
    partition: number;
    offset: number;
    key: string;
    value: string;
    timestamp: string;
    headers: Record<string, string>;
}
