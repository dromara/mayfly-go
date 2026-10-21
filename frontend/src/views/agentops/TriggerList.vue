<template>
    <div class="h-full">
        <data-table
            ref="dataTableRef"
            :page-api="agentTriggerApi.list"
            :search-items="searchItems"
            v-model:query-form="params"
            :show-selection="true"
            v-model:selection-data="state.selectionData"
            :columns="columns"
        >
            <template #tableHeader>
                <el-button v-auth="perms.save" type="primary" icon="plus" plain @click="openFormDialog(false)">{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.del" :disabled="selectionData.length < 1" type="danger" icon="delete" @click="deleteTrigger()">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <!-- 运行配置：关联任务（复用任务全套配置）或内联档位。
                 必须显式带「关联任务」前缀：该列不是「任务名列」，没有前缀时看不出这一行走的是
                 复用任务还是内联档位（两个分支的标签都是普通文字）。
                 名字由后端 VO 带回来（taskName）：前端拿任务列表自行映射会与任务列表的可见面不同源，
                 「看得到触发器、看不到那个任务」的行就又会退化成裸 id -->
            <template #runConfig="{ data }">
                <el-tag v-if="data.taskId" type="warning" effect="plain">
                    {{ $t('agentops.bindTask') }}：{{ data.taskName || $t('agentops.bindTaskMissing', { id: data.taskId }) }}
                </el-tag>
                <el-tag v-else type="info" effect="plain">{{ enumLabel(ProfileEnum, data.profileId, t) }}</el-tag>
            </template>
        </data-table>

        <TriggerEdit v-model:visible="triggerEdit.visible" v-model:data="triggerEdit.data" :title="triggerEdit.title" @submit-success="search" />
    </div>
</template>

<script lang="ts" setup>
import { enumLabel } from '@/common/Enum';
import { baseUrl } from '@/common/request';
import { buildWebhookUrl } from './webhookUrl';
import { copyToClipboard } from '@/common/utils/string';
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { defineAsyncComponent, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import { agentTriggerApi } from './api';
import { AgentTaskStatusEnum, ProfileEnum, TriggerSourceEnum } from './enums';
import { triggerSourceOptionsApi } from './useTriggerSources';
import { AgentopsPerms as P } from './perms';
import type { AgentTrigger } from './types';

const TriggerEdit = defineAsyncComponent(() => import('./TriggerEdit.vue'));

const { t } = useI18n();

// 权限码取自集中声明（与后端路由、迁移资源三方同源）
const perms = {
    save: P.triggerSave,
    del: P.triggerDel,
};

const searchItems = [
    SearchItem.input('name', 'common.name'),
    // 来源筛选项取后端注册表（与表单同源）：新增来源无需改前端即可被筛出
    SearchItem.select('sourceType', 'agentops.sourceType').withOptionsApi(triggerSourceOptionsApi()),
    SearchItem.select('enabled', 'common.status').withEnum(AgentTaskStatusEnum),
];

const columns = ref([
    TableColumn.new('name', 'common.name'),
    TableColumn.new('sourceType', 'agentops.sourceType').typeTag(TriggerSourceEnum),
    TableColumn.new('enabled', 'common.status').typeTag(AgentTaskStatusEnum),
    TableColumn.new('runConfig', 'agentops.runConfig').isSlot(),
    TableColumn.new('ownerUsername', 'agentops.owner'),
    TableColumn.new('cooldownSeconds', 'agentops.cooldown').alignCenter(),
    TableColumn.new('lastFireTime', 'agentops.lastFireTime').isTime(),
    TableColumn.new('action', 'common.operation')
        .setActions<AgentTrigger>([
            {
                key: 'webhookUrl',
                label: 'agentops.webhookUrl',
                type: 'primary',
                // 推送地址内含鉴权令牌，与「保存触发器」同一权限点把守（详情接口也只回传给它）
                perm: perms.save,
                // 仅 webhook 来源有推送地址可复制（列表行不再携带令牌，故不再按令牌判断）
                disabled: (row) => row.sourceType !== TriggerSourceEnum.Webhook.value,
                onClick: (row) => copyWebhookUrl(row),
            },
            { key: 'edit', label: 'common.edit', type: 'primary', perm: perms.save, onClick: (row) => openFormDialog(row) },
        ])
        .fixedRight()
        .noShowOverflowTooltip()
        .alignCenter(),
]);

const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');

const state = reactive({
    params: {
        pageNum: 1,
        pageSize: 0,
        name: null,
        sourceType: null,
        enabled: null,
    },
    selectionData: [] as AgentTrigger[],
    triggerEdit: {
        visible: false,
        data: null as AgentTrigger | null,
        title: '',
    },
});

const { selectionData, params, triggerEdit } = toRefs(state);

// 枚举文案统一走 common/Enum 的 enumLabel（颜色走 enumTagType），不再在各页各写一份

const openFormDialog = (data: AgentTrigger | false) => {
    if (data) {
        state.triggerEdit.data = data;
        state.triggerEdit.title = useI18nEditTitle('menu.aiopsTrigger');
    } else {
        state.triggerEdit.data = null;
        state.triggerEdit.title = useI18nCreateTitle('menu.aiopsTrigger');
    }
    state.triggerEdit.visible = true;
};

/**
 * 复制该触发器的 webhook 推送地址：origin + API 基址 + 令牌路径，供外部监控（Prometheus/Zabbix/自定义）配置。
 *
 * 令牌走详情接口按 id 取（与保存同一权限点）：列表返回体不携带令牌——它是免鉴权 webhook 路由的唯一凭据，
 * 而列表 GET 按平台惯例不带权限码（任意登录用户可达）
 */
const copyWebhookUrl = async (row: AgentTrigger) => {
    const detail = await agentTriggerApi.detail.request({ id: row.id });
    if (!detail?.webhookToken) {
        Msg.warning('agentops.webhookTokenMissing');
        return;
    }
    // 地址拼装收在 buildWebhookUrl（baseApiUrl 默认是含 origin 的绝对地址，
    // 旧写法无条件前置 origin 会拼出 http://hosthttp://host/api/… 这种外部监控无法使用的地址）
    void copyToClipboard(buildWebhookUrl(baseUrl, detail.webhookToken, window.location.origin));
};

const deleteTrigger = async () => {
    try {
        if (!(await useI18nDeleteConfirm(state.selectionData.map((x) => x.name).join('、')))) {
            return;
        }
        await agentTriggerApi.del.request({ ids: state.selectionData.map((x) => x.id).join(',') });
        Msg.deleteSuccess();
        search();
    } catch (err) {
        // 取消删除
    }
};

const search = () => {
    dataTableRef.value?.search();
};
</script>
