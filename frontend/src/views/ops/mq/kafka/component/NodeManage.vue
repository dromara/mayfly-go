<template>
    <div class="kafka-node-manage h-full card p-1! flex flex-col gap-2">
        <data-table :data="brokers" :pageable="false" :columns="brokerColumns" :card="false" stripe v-loading="loading">
            <template #tableHeader>
                <el-button @click="refreshBrokers" icon="refresh" :loading="loading" size="small" plain>{{ $t('common.refresh') }}</el-button>
            </template>
        </data-table>

        <el-drawer
            :append-to-body="false"
            v-model="openDrawer"
            :before-close="cancel"
            :destroy-on-close="true"
            :close-on-click-modal="true"
            size="80%"
            :title="$t('mq.kafka.brokerConfig') + selectedBroker?.addr"
            class="broker-config-drawer"
            :with-header="false"
        >
            <div class="drawer-body">
                <div class="toolbar">
                    <div class="">
                        <el-input v-model="searchConfig" :placeholder="$t('mq.kafka.configName')" clearable size="small" class="w-60 mb-2" />
                    </div>
                    <span class="text-sm text-gray-500">{{ `count: ${filteredBrokerConfigs.length}` }}</span>
                </div>

                <el-table :data="filteredBrokerConfigs" stripe style="width: 100%" v-loading="loading" height="100%">
                    <el-table-column type="index" label="#" width="50" />
                    <el-table-column prop="Key" :label="$t('mq.kafka.configName')" min-width="200" />
                    <el-table-column prop="Value" :label="$t('mq.kafka.configValue')" min-width="300" />
                    <el-table-column prop="Source" :label="$t('mq.kafka.configSource')" min-width="150" />
                    <el-table-column prop="Sensitive" :label="$t('mq.kafka.configSensitive')" min-width="150" />
                </el-table>
            </div>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { Msg } from '@/hooks/useI18n';
import DataTable from '@/components/data-table/DataTable.vue';
import { TableColumn } from '@/components/data-table';
import { computed, nextTick, onMounted, reactive, ref, toRefs } from 'vue';
import { mqApi } from '../../api';
import type { KafkaBroker, KafkaConfigEntry } from '../../types';

const props = defineProps({
    kafkaId: {
        type: Number,
        required: true,
    },
});

const loading = ref(false);
const selectedBroker = ref<KafkaBroker | null>(null);
const openDrawer = ref(false);
const searchConfig = ref('');

const state = reactive({
    brokers: [] as KafkaBroker[],
    brokerConfigs: [] as KafkaConfigEntry[],
});

const cancel = () => {
    state.brokerConfigs = [];
    openDrawer.value = false;
    searchConfig.value = '';
};

const { brokers, brokerConfigs } = toRefs(state);

// broker 列表声明式列：操作列由 setActions 统一渲染（抽屉内的 broker 配置表为二级明细，仍用裸 el-table）
const brokerColumns = [
    TableColumn.new('id', 'mq.kafka.nodeId'),
    TableColumn.new('addr', 'mq.kafka.addr'),
    TableColumn.new('rac', 'mq.kafka.rack'),
    TableColumn.new('action', 'common.operation')
        .setActions<KafkaBroker>([{ key: 'viewConfig', label: 'mq.kafka.viewConfig', type: 'primary', icon: 'setting', onClick: (row) => viewBrokerConfig(row) }], { size: 'small' })
        .fixedRight()
        .alignCenter(),
];

const filteredBrokerConfigs = computed(() => {
    if (!searchConfig.value) {
        return state.brokerConfigs;
    }
    return state.brokerConfigs.filter((config) => config.Key.toLowerCase().includes(searchConfig.value.toLowerCase()));
});

onMounted(() => setTimeout(() => nextTick(refreshBrokers), 500));

const refreshBrokers = async () => {
    loading.value = true;
    try {
        const res = await mqApi.kafkaTopicBrokers.request({ id: props.kafkaId });
        state.brokers = res || [];
    } catch (error: unknown) {
        Msg.error((error instanceof Error ? error.message : String(error)) || 'common.requestFail');
    } finally {
        loading.value = false;
    }
};

const viewBrokerConfig = async (broker: KafkaBroker) => {
    selectedBroker.value = broker;
    openDrawer.value = true;
    loading.value = true;
    try {
        const res = await mqApi.kafkaTopicBrokerConfig.request({
            id: props.kafkaId,
            brokerId: broker.id,
        });
        try {
            if (res && res[broker.id] && res[broker.id].Configs) {
                res[broker.id].Configs.sort((a, b) => (a.Key > b.Key ? 1 : -1));
                state.brokerConfigs = res[broker.id].Configs;
            } else if (res && res.length > 0) {
                state.brokerConfigs = res.filter((a) => a.Name === '1')[0]?.Configs ?? [];
            } else {
                state.brokerConfigs = [];
            }
        } catch (e) {
            Msg.error('解析kafka配置信息失败,请查看控制台日志');
            console.error('解析kafka配置信息失败', e, res);
        }
    } catch (error: unknown) {
        Msg.error((error instanceof Error ? error.message : String(error)) || 'common.requestFail');
    } finally {
        loading.value = false;
    }
};
</script>

<style lang="scss" scoped>
.broker-config-drawer :deep(.el-drawer__body) {
    padding: 8px;
    overflow: hidden;
}

.drawer-body {
    display: flex;
    flex-direction: column;
    height: 100%;
    gap: 8px;
}

.drawer-body .el-table {
    flex: 1;
    min-height: 0;
}

.kafka-node-manage :deep(.el-table) {
    flex: 1;
    min-height: 0;
}

.toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-shrink: 0;
}
</style>
