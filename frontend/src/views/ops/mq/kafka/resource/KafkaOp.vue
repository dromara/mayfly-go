<template>
    <div class="kafka-op h-full">
        <el-tabs v-model="activeTab" class="h-full" @tab-click="handleTabClick" v-loading="loading">
            <el-tab-pane :label="$t('mq.kafka.nodeManage')" name="node">
                <node-manage v-show="activeTab === 'node'" :kafka-id="kafkaId" />
            </el-tab-pane>
            <el-tab-pane :label="$t('mq.kafka.topicManage')" name="topic">
                <topic-manage
                    v-show="activeTab === 'topic'"
                    :kafka-id="kafkaId"
                    :topics="topics"
                    :groups="groups"
                    :loading="loading"
                    @produce="handleProduceMessage"
                    @consume="handleConsumeMessage"
                    @refresh="loadData"
                />
            </el-tab-pane>
            <el-tab-pane :label="$t('mq.kafka.produceMessage')" name="produce">
                <produce-message v-show="activeTab === 'produce'" :kafka-id="kafkaId" :default-topic="selectedTopic" :topics="topicNames" />
            </el-tab-pane>
            <el-tab-pane :label="$t('mq.kafka.consumeMessage')" name="consume">
                <consume-message v-show="activeTab === 'consume'" :kafka-id="kafkaId" :default-topic="selectedTopic" :topics="topicNames" :groups="groups" />
            </el-tab-pane>
            <el-tab-pane :label="$t('mq.kafka.consumerGroup')" name="group">
                <consumer-group v-show="activeTab === 'group'" :kafka-id="kafkaId" :groups="groups" :loading="loading" @refresh="loadData" />
            </el-tab-pane>
        </el-tabs>
    </div>
</template>

<script lang="ts" setup>
import { Msg } from '@/hooks/useI18n';
import { computed, defineAsyncComponent, onMounted, ref } from 'vue';
import { mqApi } from '../../api';
import type { KafkaGroup, KafkaTopicView } from '../../types';

const NodeManage = defineAsyncComponent(() => import('../component/NodeManage.vue'));
const TopicManage = defineAsyncComponent(() => import('../component/TopicManage.vue'));
const ProduceMessage = defineAsyncComponent(() => import('../component/ProduceMessage.vue'));
const ConsumeMessage = defineAsyncComponent(() => import('../component/ConsumeMessage.vue'));
const ConsumerGroup = defineAsyncComponent(() => import('../component/ConsumerGroup.vue'));

const props = defineProps<{
    kafkaId?: number;
    tabKey?: string;
}>();

const activeTab = ref('node');
const kafkaId = ref<number>(props.kafkaId || 0);
const selectedTopic = ref<string>('');
const loading = ref(false);
const topics = ref<KafkaTopicView[]>([]);
const groups = ref<KafkaGroup[]>([]);

// 计算属性：提取 topic 名称列表
const topicNames = computed(() => topics.value.map((item) => item.name));

const emits = defineEmits(['init']);

const initKafka = (params: Record<string, unknown>) => {
    kafkaId.value = Number(params.id);
    selectedTopic.value = '';
    loadData();
};

const loadData = async () => {
    if (!kafkaId.value) return;
    loading.value = true;
    try {
        const [topicsRes, groupsRes] = await Promise.all([
            mqApi.kafkaTopicList.request({ id: kafkaId.value }),
            mqApi.kafkaGetGroups.request({ id: kafkaId.value }),
        ]);
        // 转换 topics 数据格式
        topics.value = (topicsRes || []).map(
            (topic) =>
                ({
                    name: topic.topic,
                    partitionCount: topic.partition_count || 0,
                    replicationFactor: topic.replication_factor || 0,
                    partitions: topic.partitions || [],
                    isInternal: topic.IsInternal,
                    status: topic.Err === '' ? 'HEALTHY' : `ERROR：${topic.Err}`,
                }) as KafkaTopicView
        );
        groups.value = groupsRes || [];
    } catch (error: unknown) {
        Msg.error((error instanceof Error ? error.message : String(error)) || 'common.requestFail');
    } finally {
        loading.value = false;
    }
};

const handleTabClick = (tab: { props: { name: string } }) => {
    // 切换 tab 时清空选中的 topic
    if (tab.props.name !== 'produce' && tab.props.name !== 'consume') {
        selectedTopic.value = '';
    }
};

const handleProduceMessage = (topic: string) => {
    selectedTopic.value = topic;
    activeTab.value = 'produce';
};

const handleConsumeMessage = (topic: string) => {
    selectedTopic.value = topic;
    activeTab.value = 'consume';
};

defineExpose({
    initKafka,
    onRefresh: ()=>{initKafka({id:kafkaId.value})},
    onClose: ()=>{
        kafkaId.value = 0;
        selectedTopic.value = '';
        topics.value = [];
        groups.value = [];
    }
});
</script>

<style lang="scss" scoped>
.kafka-op {
    :deep(.el-tabs) {
        height: 100%;

        .el-tabs__content {
            height: calc(100% - 55px);
            overflow: visible;
        }

        .el-tab-pane {
            height: 100%;
        }
    }
}
</style>
