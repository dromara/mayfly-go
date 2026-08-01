<template>
    <div class="kafka-topic-manage h-full card !p-1 flex flex-col gap-2">
        <div class="toolbar flex items-center justify-between mb-2">
            <div class="flex items-center">
                <el-input v-model="searchTopic" :placeholder="$t('mq.kafka.searchTopic')" clearable size="small" class="w-60" @clear="loadTopics" />
                <el-button @click="loadTopics" icon="refresh" :loading="loading" size="small" plain class="ml-2">
                    {{ $t('common.refresh') }}
                </el-button>
                <el-button @click="showCreateTopicDialog" type="primary" size="small" icon="plus" v-auth="'kafka:topic:create'">
                    {{ $t('mq.kafka.createTopic') }}
                </el-button>
                <el-button @click="openViewPartitions" type="primary" size="small" icon="data-line">
                    {{ $t('mq.kafka.partitions') }}
                </el-button>
            </div>
            <span class="text-sm text-gray-500">{{ `count: ${filteredTopics.length}` }}</span>
        </div>

        <el-table :data="filteredTopics" stripe style="width: 100%" v-loading="loading" @row-contextmenu="handleRowContextmenu">
            <el-table-column prop="name" :label="$t('mq.kafka.topicName')" min-width="200">
                <template #default="{ row }">
                    <el-link type="primary" @click="viewTopicConfig(row)">{{ row.name }}</el-link>
                </template>
            </el-table-column>
            <el-table-column prop="partitionCount" :label="$t('mq.kafka.partitions')" min-width="100">
                <template #default="{ row }">
                    <el-link type="primary" @click="viewPartitions(row)">{{ row.partitionCount }}</el-link>
                </template>
            </el-table-column>
            <el-table-column prop="replicationFactor" :label="$t('mq.kafka.replicationFactor')" min-width="120" />
            <el-table-column prop="status" :label="$t('mq.kafka.topicStatus')" min-width="100">
                <template #default="{ row }">
                    <el-tag :type="row.status === 'HEALTHY' ? 'success' : 'danger'" size="small">{{ row.status }}</el-tag>
                </template>
            </el-table-column>
            <el-table-column prop="isInternal" :label="$t('mq.kafka.topicIsInternal')" min-width="100">
                <template #default="{ row }">
                    <el-tag :type="row.isInternal ? 'success' : 'danger'" size="small">{{ row.isInternal ? 'Y' : 'N' }}</el-tag>
                </template>
            </el-table-column>
            <el-table-column :label="$t('common.operation')" width="130" align="center">
                <template #default="{ row }">
                    <el-dropdown trigger="click" @command="handleTopicCommand($event, row)" >
                        <el-button type="primary" size="small" link>
                            {{ $t('common.operation') }}
                        </el-button>
                        <template #dropdown>
                            <el-dropdown-menu>
                                <el-dropdown-item command="produce" icon="upload" v-auth="'kafka:topic:produce'">
                                    {{ $t('mq.kafka.produceMessage') }}
                                </el-dropdown-item>
                                <el-dropdown-item command="consume" icon="download" v-auth="'kafka:topic:consume'">
                                    {{ $t('mq.kafka.consumeMessage') }}
                                </el-dropdown-item>
                                <el-dropdown-item command="partitions" icon="data-line">
                                    {{ $t('mq.kafka.viewPartitions') }}
                                </el-dropdown-item>
                                <el-dropdown-item command="config" icon="setting">
                                    {{ $t('mq.kafka.viewConfig') }}
                                </el-dropdown-item>
                                <el-dropdown-item command="delete" icon="delete" v-auth="'kafka:topic:delete'" divided>
                                    {{ $t('common.delete') }}
                                </el-dropdown-item>
                            </el-dropdown-menu>
                        </template>
                    </el-dropdown>
                </template>
            </el-table-column>
        </el-table>

        <contextmenu :dropdown="contextmenu.dropdown" :items="contextmenu.items" ref="contextmenuRef" />

        <!-- 创建 Topic 对话框 -->
        <el-dialog :title="$t('mq.kafka.createTopic')" v-model="createTopicDialog.visible" width="600px" :close-on-click-modal="false">
            <el-form ref="createTopicFormRef" :model="createTopicDialog.form" :rules="createTopicFormRules" label-width="auto">
                <el-form-item :label="$t('mq.kafka.topicName')" prop="topic">
                    <el-input v-model="createTopicDialog.form.topic" :placeholder="$t('mq.kafka.topicNamePlaceholder')" />
                </el-form-item>
                <el-form-item :label="$t('mq.kafka.partitions')" prop="numPartitions">
                    <el-input-number v-model="createTopicDialog.form.numPartitions" :min="1" :max="100" />
                </el-form-item>
                <el-form-item :label="$t('mq.kafka.replicationFactor')" prop="replicationFactor">
                    <el-input-number v-model="createTopicDialog.form.replicationFactor" :min="1" :max="10" />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="createTopicDialog.visible = false">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" @click="confirmCreateTopic" :loading="createTopicDialog.loading">
                    {{ $t('common.confirm') }}
                </el-button>
            </template>
        </el-dialog>

        <!-- 创建 partitions 对话框 -->
        <el-dialog :title="$t('mq.kafka.createPartitions')" v-model="createPartitionsDialog.visible" width="600px" :close-on-click-modal="false">
            <el-form ref="createPartitionsFormRef" :model="createPartitionsDialog.form" :rules="createPartitionsFormRules" label-width="auto">
                <el-form-item :label="$t('mq.kafka.partitions')" prop="numPartitions">
                    <el-input-number v-model="createPartitionsDialog.form.numPartitions" :min="1" :max="100" />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="createPartitionsDialog.visible = false">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" @click="confirmCreatePartitions" :loading="createPartitionsDialog.loading">
                    {{ $t('common.confirm') }}
                </el-button>
            </template>
        </el-dialog>

        <!-- Topic 配置对话框 -->
        <el-drawer
            :append-to-body="false"
            v-model="topicConfigDialog.visible"
            :before-close="cancelViewTopicConfig"
            :destroy-on-close="true"
            :close-on-click-modal="false"
            size="80%"
            :title="`${$t('mq.kafka.topicConfig')} [${topicConfigDialog.topic}] `"
        >
            <div class="toolbar">
                <div class="">
                    <el-input v-model="searchTopicConfig" :placeholder="$t('mq.kafka.configName')" clearable size="small" class="mb-2" />
                </div>
                <span class="text-sm text-gray-500">{{ `count: ${filteredTopicConfigs.length}` }}</span>
            </div>

            <el-table :data="filteredTopicConfigs" stripe style="width: 100%" v-loading="loading">
                <el-table-column type="index" label="#" width="50" />
                <el-table-column prop="Key" :label="$t('mq.kafka.configName')" min-width="200" />
                <el-table-column prop="Value" :label="$t('mq.kafka.configValue')" min-width="300" />
                <el-table-column prop="Source" :label="$t('mq.kafka.configSource')" min-width="150" />
                <el-table-column prop="Sensitive" :label="$t('mq.kafka.configSensitive')" min-width="150" />
            </el-table>
        </el-drawer>
        <!--    Topic　分区信息    -->
        <el-drawer
            :append-to-body="false"
            v-model="topicPartitionsDialog.visible"
            :before-close="cancelViewTopicPartitions"
            :destroy-on-close="true"
            :close-on-click-modal="false"
            size="80%"
            :title="`${$t('mq.kafka.topicPartitions')} [${topicPartitionsDialog.topic}] `"
        >
            <div class="toolbar flex items-center justify-between mb-2">
                <div class="flex items-center">
                    <el-select
                        v-model="topicPartitionsDialog.topic"
                        :placeholder="$t('mq.kafka.selectTopicPlaceholder')"
                        clearable
                        size="small"
                        filterable
                        style="width: 150px"
                        @change="loadTopicPartitions"
                    >
                        <el-option v-for="topic in topics" :key="topic.name" :label="topic.name" :value="topic.name" @select="viewPartitions(topic)" />
                    </el-select>

                    <el-button @click="showCreatePartitionsDialog" class="ml-3" type="primary" size="small" icon="plus" v-auth="'kafka:topic:create'">
                        {{ $t('mq.kafka.createPartitions') }}
                    </el-button>

                    <el-select
                        v-model="topicPartitionsDialog.group"
                        :placeholder="$t('mq.kafka.selectGroupPlaceholder')"
                        clearable
                        size="small"
                        filterable
                        class="ml-3"
                        style="width: 150px"
                        @change="loadOffsets"
                    >
                        <el-option v-for="g in groups" :key="g.Group" :label="g.Group" :value="g.Group" />
                    </el-select>

                    <el-button @click="loadGroups" icon="refresh" :loading="groupLoading" size="small" plain class="ml-3">
                        {{ $t('common.refresh') }}group
                    </el-button>

                    <el-button @click="loadOffsets" icon="refresh" :loading="groupLoading" size="small" plain class="ml-3">
                        {{ $t('mq.kafka.loadOffsets') }}
                    </el-button>
                </div>
                <span class="text-sm text-gray-500">{{ `count: ${topicPartitionsDialog.topicPartitions.length}` }}</span>
            </div>

            <el-table :data="topicPartitionsDialog.topicPartitions" style="width: 100%" v-loading="loading">
                <el-table-column type="index" label="ID" width="50" :index="topicPartitionsIndexMethod" />
                <el-table-column prop="leader" label="Leader" min-width="150" />
                <el-table-column prop="Health" label="Health" min-width="150">
                    <template #default="{ row }">
                        <el-tag :type="row.err == 0 ? 'success' : 'danger'" size="small">{{ row.err == 0 ? 'HEALTHY' : `Error: ${row.err}` }}</el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="LeaderEpoch" label="LeaderEpoch" min-width="150" />
                <el-table-column prop="OfflineReplicas" label="OfflineReplicas" min-width="150" />
                <el-table-column prop="replicas" label="replicas" min-width="150">
                    <template #default="{ row }">
                        <el-tag v-for="(replica, index) in row.replicas" :key="index">
                            {{ replica }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="isr" label="isr" min-width="150">
                    <template #default="{ row }">
                        <el-tag v-for="(isr, index) in row.isr" :key="index">
                            {{ isr }}
                        </el-tag>
                    </template>
                </el-table-column>
            </el-table>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { computed, nextTick, reactive, ref, toRefs } from 'vue';
import { useI18n } from 'vue-i18n';
import { mqApi } from '../../api';
import type { KafkaConfigEntry, KafkaGroup, KafkaTopicPartition, KafkaTopicView } from '../../types';

const { t } = useI18n();

const props = defineProps({
    kafkaId: {
        type: Number,
        required: true,
    },
    topics: {
        type: Array as () => KafkaTopicView[],
        default: () => [],
    },
    groups: {
        type: Array as () => KafkaGroup[],
        default: () => [],
    },
    loading: {
        type: Boolean,
        default: false,
    },
});

const emits = defineEmits(['produce', 'consume', 'refresh']);

const searchTopic = ref('');
const searchTopicConfig = ref('');
const groupLoading = ref(false);

const state = reactive({
    createTopicDialog: {
        visible: false,
        loading: false,
        form: {
            topic: '',
            numPartitions: 1,
            replicationFactor: 1,
        },
    },
    createPartitionsDialog: {
        visible: false,
        loading: false,
        form: {
            topic: '',
            numPartitions: 1,
        },
    },
    topicConfigDialog: {
        visible: false,
        topic: '',
        topicConfigs: [] as KafkaConfigEntry[],
    },
    topicPartitionsDialog: {
        visible: false,
        topic: '',
        group: '',
        topicPartitions: [] as KafkaTopicPartition[],
    },
    contextmenu: {
        dropdown: {
            x: 0,
            y: 0,
        },
        items: [
            new ContextmenuItem('produce', 'kafka.produceMessage')
                .withIcon('upload')
                .withPermission('kafka:topic:produce')
                .withOnClick((data: KafkaTopicView) => handleProduceMessage(data)),
            new ContextmenuItem('consume', 'kafka.consumeMessage')
                .withIcon('download')
                .withPermission('kafka:topic:consume')
                .withOnClick((data: KafkaTopicView) => handleConsumeMessage(data)),
            new ContextmenuItem('partitions', 'kafka.viewPartitions').withIcon('data-line').withOnClick((data: KafkaTopicView) => viewPartitions(data)),
            new ContextmenuItem('config', 'kafka.viewConfig').withIcon('setting').withOnClick((data: KafkaTopicView) => viewTopicConfig(data)),
            new ContextmenuItem('delete', 'common.delete')
                .withIcon('delete')
                .withPermission('kafka:topic:delete')
                .withOnClick((data: KafkaTopicView) => handleDeleteTopic(data)),
        ] as ContextmenuItem[],
    },
});

const { createTopicDialog, createPartitionsDialog, topicConfigDialog, topicPartitionsDialog, contextmenu } = toRefs(state);
const contextmenuRef = ref();

const topicPartitionsIndexMethod = (index: number) => {
    return index;
};

// 使用 computed 包装 props，保持模板中的响应性
const topics = computed(() => props.topics);
const groups = computed(() => props.groups);
const loading = computed(() => props.loading);

const filteredTopicConfigs = computed(() => {
    if (!searchTopicConfig.value) {
        return state.topicConfigDialog.topicConfigs;
    }
    return state.topicConfigDialog.topicConfigs.filter((config) => config.Key.toLowerCase().includes(searchTopicConfig.value.toLowerCase()));
});
const createTopicFormRef = ref();
const createPartitionsFormRef = ref();

const createTopicFormRules = {
    name: [Rules.requiredInput('kafka.topicName')],
    partitions: [Rules.requiredInput('kafka.partitions')],
    replicationFactor: [Rules.requiredInput('kafka.replicationFactor')],
};
const createPartitionsFormRules = {
    partitions: [Rules.requiredInput('kafka.partitions')],
};

const filteredTopics = computed(() => {
    if (!searchTopic.value) {
        return props.topics;
    }
    return props.topics.filter((topic) => topic.name.toLowerCase().includes(searchTopic.value.toLowerCase()));
});

const loadTopics = () => {
    emits('refresh');
};

const loadGroups = () => {
    emits('refresh');
};

const loadTopicPartitions = (topicName: string) => {
    // 根据选中的 topic 名称查找对应的 topic 数据并更新分区信息
    const selectedTopic = props.topics.find((t) => t.name === topicName);
    if (selectedTopic) {
        state.topicPartitionsDialog.topicPartitions = selectedTopic.partitions;
    }
};

const loadOffsets = async () => {
    // Load offsets for topic partitions
};

const showCreateTopicDialog = () => {
    state.createTopicDialog.form = {
        topic: '',
        numPartitions: 1,
        replicationFactor: 1,
    };
    state.createTopicDialog.visible = true;
};
const showCreatePartitionsDialog = () => {
    if (!state.topicPartitionsDialog.topic) {
        Msg.warning('mq.kafka.selectTopicWarning');
        return;
    }

    state.createPartitionsDialog.form = {
        topic: state.topicPartitionsDialog.topic,
        numPartitions: 1,
    };
    state.createPartitionsDialog.visible = true;
};

const confirmCreateTopic = async () => {
    if (!createTopicFormRef.value) return;
    await createTopicFormRef.value?.validate();
    state.createTopicDialog.loading = true;
    try {
        await mqApi.kafkaTopicCreate.request({
            id: props.kafkaId,
            ...state.createTopicDialog.form,
        });
        Msg.saveSuccess();
        state.createTopicDialog.visible = false;
        emits('refresh');
    } catch (error: unknown) {
        Msg.error(error instanceof Error ? error.message : String(error));
    } finally {
        state.createTopicDialog.loading = false;
    }
};
const confirmCreatePartitions = async () => {
    if (!createPartitionsFormRef.value) return;
    await createPartitionsFormRef.value?.validate();
    state.createPartitionsDialog.loading = true;
    try {
        await mqApi.kafkaTopicCreatePartitions.request({
            id: props.kafkaId,
            ...state.createPartitionsDialog.form,
        });
        Msg.saveSuccess();
        state.createPartitionsDialog.visible = false;
        emits('refresh');
        await nextTick(() => {
            setTimeout(() => {
                loadTopicPartitions(state.createPartitionsDialog.form.topic);
            }, 200);
        });
    } catch (error: unknown) {
        Msg.error(error instanceof Error ? error.message : String(error));
    } finally {
        state.createPartitionsDialog.loading = false;
    }
};

const viewTopicConfig = async (topic: KafkaTopicView) => {
    try {
        const res = await mqApi.kafkaTopicInfo.request({
            id: props.kafkaId,
            topic: topic.name,
        });
        state.topicConfigDialog.topic = topic.name;

        if (res && res[0].Configs) {
            res[0].Configs.sort((a, b) => (a['Key'] > b['Key'] ? 1 : -1));
            state.topicConfigDialog.topicConfigs = res && res[0].Configs;
        } else {
            state.topicConfigDialog.topicConfigs = [];
        }

        state.topicConfigDialog.visible = true;
    } catch (error: unknown) {
        Msg.error(error instanceof Error ? error.message : String(error));
    } finally {
    }
};

const cancelViewTopicConfig = () => {
    state.topicConfigDialog.visible = false;
    searchTopicConfig.value = '';
    state.topicConfigDialog.topicConfigs = [];
    state.topicConfigDialog.topic = '';
};
const cancelViewTopicPartitions = () => {
    state.topicPartitionsDialog.visible = false;
    state.topicPartitionsDialog.topicPartitions = [];
    state.topicPartitionsDialog.topic = '';
};

const handleDeleteTopic = async (topic: KafkaTopicView) => {
    await useI18nDeleteConfirm(`Topic: ${topic.name}`);
    try {
        await mqApi.kafkaTopicDelete.request({
            id: props.kafkaId,
            topic: topic.name,
        });
        Msg.saveSuccess();
        emits('refresh');
    } catch (error: unknown) {
        Msg.error(error instanceof Error ? error.message : String(error));
    }
};

const viewPartitions = (topic: KafkaTopicView) => {
    state.topicPartitionsDialog.visible = true;
    state.topicPartitionsDialog.topicPartitions = topic.partitions;
    state.topicPartitionsDialog.topic = topic.name;
    openViewPartitions();
};

const openViewPartitions = () => {
    state.topicPartitionsDialog.visible = true;
};

const handleProduceMessage = (topic: KafkaTopicView) => {
    emits('produce', topic.name);
};

const handleConsumeMessage = (topic: KafkaTopicView) => {
    emits('consume', topic.name);
};

const handleTopicCommand = (command: string, topic: KafkaTopicView) => {
    switch (command) {
        case 'produce':
            handleProduceMessage(topic);
            break;
        case 'consume':
            handleConsumeMessage(topic);
            break;
        case 'partitions':
            viewPartitions(topic);
            break;
        case 'config':
            viewTopicConfig(topic);
            break;
        case 'delete':
            handleDeleteTopic(topic);
            break;
    }
};

const handleRowContextmenu = (row: unknown, _column: unknown, event: MouseEvent) => {
    event.preventDefault();
    event.stopPropagation();
    const { clientX, clientY } = event;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    contextmenuRef.value?.openContextmenu(row);
};
</script>

<style lang="scss" scoped>
.kafka-topic-manage {
    :deep(.el-table) {
        flex: 1;
        min-height: 0;
    }

    .toolbar {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
}
</style>
