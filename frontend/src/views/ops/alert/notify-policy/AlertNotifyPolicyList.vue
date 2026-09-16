<template>
    <div class="h-full">
        <page-table ref="pageTableRef" :page-api="alertNotifyPolicyApi.list" :search-items="searchItems" v-model:query-form="query" :columns="columns" :data-handler-fn="handleListData">
            <template #tableHeader>
                <el-button v-auth="'alert:notifyPolicy:save'" type="primary" icon="plus" @click="onAdd">{{ $t('alert.addNotifyPolicy') }}</el-button>
            </template>

            <template #matchLabels="{ data }">
                <LabelTags :labels="data.matchLabels" />
            </template>

            <template #channelIds="{ data }">
                <template v-if="data.channelIds?.length">
                    <el-tag v-for="chId in data.channelIds" :key="chId" size="small" class="mr-1 mb-1">
                        {{ channelMap.get(chId) || $t('alert.channelFallback', { id: chId }) }}
                    </el-tag>
                </template>
                <span v-else class="text-gray-400">-</span>
            </template>

            <template #receiverIds="{ data }">
                <template v-if="data.receiverIds?.length">
                    <el-tag v-for="accId in data.receiverIds" :key="accId" size="small" type="success" class="mr-1 mb-1">
                        {{ accountMap.get(accId) || $t('alert.receiverFallback', { id: accId }) }}
                    </el-tag>
                </template>
                <span v-else class="text-gray-400">-</span>
            </template>

            <template #status="{ data }">
                <enum-tag :enums="AlertRuleStatusEnum" :value="data.status" />
            </template>

            <template #action="{ data }">
                <el-button link @click="onEdit(data)" type="primary">{{ $t('alert.edit') }}</el-button>
                <el-button link :type="data.status === AlertRuleStatusEnable ? 'warning' : 'success'" @click="onToggleStatus(data)">
                    {{ data.status === AlertRuleStatusEnable ? $t('alert.disable') : $t('alert.enable') }}
                </el-button>
                <el-button link @click="onDelete(data)" type="danger">{{ $t('alert.delete') }}</el-button>
            </template>
        </page-table>

        <el-dialog v-model="editVisible" :title="$t(editTitle)" width="860px" destroy-on-close>
            <el-form :model="editForm" label-width="110px">
                <el-form-item :label="$t('alert.notifyPolicyName')" required>
                    <el-input v-model="editForm.name" :placeholder="$t('alert.notifyPolicyName')" />
                </el-form-item>
                <el-form-item>
                    <template #label>
                        <div class="flex items-center">
                            {{ $t('alert.matchLabels') }}
                            <el-tooltip placement="top">
                                <template #content><span style="white-space: pre-line">{{ $t('alert.notifyPolicyTips') }}</span></template>
                                <SvgIcon name="QuestionFilled" class="ml-1" />
                            </el-tooltip>
                        </div>
                    </template>
                    <LabelAssociation v-model="editForm.matchLabels" />
                </el-form-item>
                <el-form-item :label="$t('alert.notifyPolicyChannels')" required>
                    <ChannelSelect v-model="editForm.channelIds" />
                </el-form-item>
                <el-form-item :label="$t('alert.notifyPolicyReceivers')">
                    <ReceiverSelect v-model="editForm.receiverIds" />
                </el-form-item>
                <el-form-item :label="$t('alert.repeatInterval')">
                    <el-input-number v-model="editForm.repeatInterval" :min="0" :max="86400" :controls="false" style="width: 200px" />
                    <span class="ml-2 text-gray-400">s</span>
                </el-form-item>
                <el-form-item :label="$t('alert.remark')">
                    <el-input v-model="editForm.remark" type="textarea" :rows="3" :placeholder="$t('alert.remark')" />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="editVisible = false">{{ $t('alert.cancel') }}</el-button>
                <el-button type="primary" @click="onSave">{{ $t('alert.save') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, useTemplateRef } from 'vue';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { alertNotifyPolicyApi } from '../api';
import { AlertRuleStatusEnum } from '../enums';
import type { AlertNotifyPolicyVO, AlertNotifyPolicyForm } from '../types';
import ChannelSelect from '../components/ChannelSelect.vue';
import ReceiverSelect from '../components/ReceiverSelect.vue';
import EnumTag from '@/components/enum-tag/EnumTag.vue';
import LabelAssociation from '@/views/ops/label/components/LabelAssociation.vue';
import { LabelTags } from '@/components/label-tags';
import { useLabelFill } from '@/hooks/useLabelFill';
import { channelApi } from '@/views/msg/api';
import type { MsgChannel } from '@/views/system/msg/types';
import { accountApi } from '@/views/system/api';
import type { Account } from '@/views/system/types';

const AlertRuleStatusEnable = AlertRuleStatusEnum.Enable.value;
const AlertRuleStatusDisable = AlertRuleStatusEnum.Disable.value;

// 标签批量填充
const handleListData = useLabelFill<AlertNotifyPolicyVO>('alert_notify_policy', { labelField: 'matchLabels' });

// 渠道和接收人名称映射
const channelMap = ref<Map<number, string>>(new Map());
const accountMap = ref<Map<number, string>>(new Map());

// 加载渠道和接收人数据
onMounted(async () => {
    try {
        const [channelsRes, accountsRes] = await Promise.all([
            channelApi.list.request({ pageNum: 1, pageSize: 1000 }),
            accountApi.querySimple.request({ pageNum: 1, pageSize: 1000 }),
        ]);
        const channelMapData = new Map<number, string>();
        channelsRes.list?.forEach((ch: MsgChannel) => {
            channelMapData.set(ch.id, ch.name);
        });
        channelMap.value = channelMapData;

        const accountMapData = new Map<number, string>();
        accountsRes.list?.forEach((acc: Account) => {
            accountMapData.set(acc.id, acc.name || acc.username);
        });
        accountMap.value = accountMapData;
    } catch (e) {
        console.error('Failed to load channels or accounts:', e);
    }
});

const searchItems = [
    SearchItem.input('name', 'alert.notifyPolicyName').withPlaceholder('common.keyword'),
    SearchItem.select('status', 'alert.status').withOptions(Object.values(AlertRuleStatusEnum).map((e) => ({ value: e.value, label: e.label }))),
];

const columns = [
    TableColumn.new('name', 'alert.notifyPolicyName'),
    TableColumn.new('matchLabels', 'alert.matchLabels').isSlot().setMinWidth(200),
    TableColumn.new('channelIds', 'alert.notifyPolicyChannels').isSlot().alignCenter(),
    TableColumn.new('receiverIds', 'alert.notifyPolicyReceivers').isSlot().alignCenter(),
    TableColumn.new('status', 'alert.status').isSlot().alignCenter(),
    TableColumn.new('remark', 'alert.remark'),
    TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(180).alignCenter(),
];

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');

const query = ref({
    name: '',
    status: undefined as number | undefined,
    pageNum: 1,
    pageSize: 0,
});

const editVisible = ref(false);
const editTitle = ref('');

const createEmptyForm = (): AlertNotifyPolicyForm => ({
    id: null,
    name: '',
    status: 1,
    matchLabels: '',
    channelIds: [],
    receiverIds: [],
    repeatInterval: 300,
    remark: '',
});

const editForm = reactive<AlertNotifyPolicyForm>(createEmptyForm());

const onAdd = () => {
    Object.assign(editForm, createEmptyForm());
    editTitle.value = 'alert.addNotifyPolicy';
    editVisible.value = true;
};

const onEdit = (row: AlertNotifyPolicyVO) => {
    Object.assign(editForm, createEmptyForm(), {
        ...row,
        channelIds: [...(row.channelIds ?? [])],
        receiverIds: [...(row.receiverIds ?? [])],
    });
    editTitle.value = 'alert.editNotifyPolicy';
    editVisible.value = true;
};

const validateForm = (): string | null => {
    if (!editForm.name.trim()) {
        return 'alert.notifyPolicyNameRequired';
    }
    if (!editForm.channelIds?.length && !editForm.receiverIds?.length) {
        return 'alert.notifyPolicyChannelOrReceiverRequired';
    }
    return null;
};

const onSave = async () => {
    const errorKey = validateForm();
    if (errorKey) {
        Msg.error(errorKey);
        return;
    }
    await alertNotifyPolicyApi.save.request({ ...editForm, name: editForm.name.trim(), matchLabels: editForm.matchLabels?.trim() || '{}' });
    Msg.saveSuccess();
    editVisible.value = false;
    pageTableRef.value?.search();
};

const onToggleStatus = async (row: AlertNotifyPolicyVO) => {
    const newStatus = row.status === AlertRuleStatusEnable ? AlertRuleStatusDisable : AlertRuleStatusEnable;
    const i18nKey = newStatus === 1 ? 'alert.confirmEnable' : 'alert.confirmDisable';
    await useI18nConfirm(i18nKey);
    await alertNotifyPolicyApi.changeStatus.request({ id: row.id, status: newStatus });
    Msg.operateSuccess();
    pageTableRef.value?.search();
};

const onDelete = async (row: AlertNotifyPolicyVO) => {
    await useI18nConfirm('alert.confirmDelete');
    await alertNotifyPolicyApi.del.request({ id: row.id });
    Msg.deleteSuccess();
    pageTableRef.value?.search();
};
</script>
