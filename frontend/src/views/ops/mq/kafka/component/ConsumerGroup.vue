<template>
    <div class="kafka-consumer-group h-full card p-1! flex flex-col gap-2">
        <div class="toolbar flex items-center justify-between mb-2">
            <div class="flex items-center">
                <el-input v-model="searchGroup" :placeholder="$t('mq.kafka.searchGroup')" clearable size="small" class="w-60" @clear="loadGroups" />
                <el-button @click="loadGroups" icon="refresh" :loading="loading" size="small" plain class="ml-2">
                    {{ $t('common.refresh') }}
                </el-button>
            </div>
            <div class="flex items-center">
                <span class="text-sm text-gray-500 mr-2">{{ $t('count') + ` ${groups.length}` }}</span>
            </div>
        </div>

        <el-table :data="filteredGroups" stripe v-loading="loading">
            <el-table-column prop="Group" :label="$t('mq.kafka.groupId')" min-width="250" />
            <el-table-column prop="Coordinator" :label="$t('mq.kafka.coordinator')" min-width="150" />
            <el-table-column prop="State" :label="$t('mq.kafka.state')" min-width="120">
                <template #default="{ row }">
                    <el-tag :type="getStateTagType(row.State)" size="small">{{ row.State }}</el-tag>
                </template>
            </el-table-column>
            <el-table-column prop="ProtocolType" :label="$t('mq.kafka.protocolType')" min-width="150" />
            <el-table-column v-bind="actionColumnProps(groupActions, groupActionOptions)">
                <template #default="{ row }">
                    <RowActions :items="groupActions" :options="groupActionOptions" :data="row" />
                </template>
            </el-table-column>
        </el-table>

        <el-drawer
            :append-to-body="false"
            v-model="membersDrawerVisible"
            :destroy-on-close="true"
            :close-on-click-modal="true"
            size="70%"
            :title="$t('mq.kafka.groupMembers') + ' - ' + selectedGroup?.Group"
            class="members-drawer"
        >
            <div class="drawer-body">
                <el-table :data="groupMembers" stripe v-loading="membersLoading" height="100%">
                    <el-table-column type="index" label="#" width="50" />
                    <el-table-column prop="ClientHost" :label="$t('mq.kafka.clientHost')" min-width="140" />
                    <el-table-column prop="ClientID" :label="$t('mq.kafka.clientID')" min-width="100" />
                    <el-table-column prop="InstanceID" :label="$t('mq.kafka.instanceID')" min-width="120">
                        <template #default="{ row }">
                            {{ row.InstanceID ?? '-' }}
                        </template>
                    </el-table-column>
                    <el-table-column prop="MemberID" :label="$t('mq.kafka.memberID')" min-width="280" />
                    <el-table-column :label="$t('mq.kafka.assignedTopics')" min-width="260">
                        <template #default="{ row }">
                            <div v-if="row.TPs && Object.keys(row.TPs).length" class="flex flex-col gap-1">
                                <div v-for="(partitions, topic) in row.TPs" :key="topic" class="text-xs">
                                    <span class="font-medium">{{ topic }}</span
                                    >:
                                    <el-tag v-for="p in partitions" :key="p" size="small" class="ml-1" type="info">{{ p }}</el-tag>
                                </div>
                            </div>
                            <span v-else class="text-gray-400">-</span>
                        </template>
                    </el-table-column>
                </el-table>
            </div>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { actionColumnProps, type ActionItem, type ActionOptions } from '@/components/data-table';
import RowActions from '@/components/data-table/RowActions.vue';
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { mqApi } from '../../api';
import type { KafkaGroup, KafkaGroupMember } from '../../types';

const { t } = useI18n();

const props = defineProps({
    kafkaId: {
        type: Number,
        required: true,
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

const emits = defineEmits(['refresh']);

const searchGroup = ref('');
const membersDrawerVisible = ref(false);
const membersLoading = ref(false);
const selectedGroup = ref<KafkaGroup | null>(null);
const groupMembers = ref<KafkaGroupMember[]>([]);

const filteredGroups = computed(() => {
    if (!searchGroup.value) {
        return props.groups;
    }
    return props.groups.filter((group) => group.Group.toLowerCase().includes(searchGroup.value.toLowerCase()));
});

const loadGroups = () => {
    emits('refresh');
};

const handleDeleteGroup = async (group: KafkaGroup) => {
    if (!(await useI18nDeleteConfirm(`Group: ${group.Group}`))) {
        return;
    }
    try {
        await mqApi.kafkaDeleteGroup.request({
            id: props.kafkaId,
            group: group.Group,
        });
        Msg.saveSuccess();
        emits('refresh');
    } catch (error: unknown) {
        Msg.error((error instanceof Error ? error.message : String(error)) || 'common.requestFail');
    }
};
const handleGetGroupMembers = async (group: KafkaGroup) => {
    selectedGroup.value = group;
    membersDrawerVisible.value = true;
    membersLoading.value = true;
    try {
        const res = await mqApi.kafkaGetGroupMembers.request({
            id: props.kafkaId,
            group: group.Group,
        });
        groupMembers.value = res || [];
    } catch (error: unknown) {
        Msg.error((error instanceof Error ? error.message : String(error)) || 'common.requestFail');
    } finally {
        membersLoading.value = false;
    }
};

/**
 * 行内操作：不走 DataTable 的面板表也用同一份声明——列宽由「文案 + 图标 + 按钮内边距」算出，
 * 不再手写 width（英文下不够用、中文下白留一截）。options 只定义一份，列属性与渲染都读它。
 */
const groupActionOptions: ActionOptions = { size: 'small' };
const groupActions: ActionItem[] = [
    { key: 'members', label: 'mq.kafka.Members', icon: 'setting', onClick: (row: KafkaGroup) => handleGetGroupMembers(row) },
    {
        key: 'delete',
        label: 'common.delete',
        type: 'danger',
        icon: 'delete',
        perm: 'kafka:group:delete',
        onClick: (row: KafkaGroup) => handleDeleteGroup(row),
    },
];

const getStateTagType = (state: string) => {
    switch (state?.toLowerCase()) {
        case 'stable':
            return 'success';
        default:
            return '';
    }
};
</script>

<style lang="scss" scoped>
.kafka-consumer-group {
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

.members-drawer :deep(.el-drawer__body) {
    padding: 8px;
    overflow: hidden;
}

.drawer-body {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.drawer-body .el-table {
    flex: 1;
    min-height: 0;
}
</style>
