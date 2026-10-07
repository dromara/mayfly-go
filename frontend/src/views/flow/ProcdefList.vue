<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="procdefApi.list"
            :search-items="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
        >
            <template #empty>
                <div class="table-empty">
                    <p>{{ $t('flow.procdefEmpty') }}</p>
                    <p class="empty-tip">{{ $t('flow.procdefEmptyTip') }}</p>
                </div>
            </template>

            <template #tableHeader>
                <el-button v-auth="perms.save" type="primary" icon="plus" @click="onEditFlowDef(false)">{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.del" :disabled="state.selectionData.length < 1" @click="onDeleteProcdef()" type="danger" icon="delete">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #codePaths="{ data }">
                <TagCodePath :path="data.tags" />
            </template>

            <template #triggerPolicy="{ data }">
                <!-- 徽标只说明「哪个场景、最严到什么级别」，具体内容放 hover：
                     管理员要能在列表上确认策略，又不必逐条打开抽屉 -->
                <el-tooltip v-if="policySummaryOf(data).length" placement="top" :show-after="250">
                    <template #content>
                        <div v-for="(line, index) in policyLinesOf(data)" :key="index">{{ line }}</div>
                    </template>
                    <div class="policy-summary">
                        <el-tag v-for="item in policySummaryOf(data)" :key="item.bizType" size="small" :type="SEVERITY_TAG_TYPES[item.severity]" effect="light">
                            {{ scenarioText(item) }}
                        </el-tag>
                    </div>
                </el-tooltip>
                <span v-else class="policy-empty">{{ $t('flow.policy.noRule') }}</span>
            </template>

            <template #action="{ data }">
                <el-button link v-if="actionBtns[perms.save]" @click="onEditFlowDef(data)" type="primary">{{ $t('common.edit') }}</el-button>

                <el-button link v-if="actionBtns[perms.save]" @click="onShowFlowDesign(data)" type="primary">{{ $t('flow.flowDesign') }}</el-button>
            </template>
        </page-table>

        <procdef-edit v-model:visible="flowDefEditor.visible" :title="flowDefEditor.title" v-model:data="flowDefEditor.data" @val-change="handleValChange()" />
        <FlowDesignDrawer
            :disabled="flowDesignEditor.disabled"
            v-model:visible="flowDesignEditor.visible"
            :data="flowDesignEditor.data"
            @save="onSaveFlowDesign"
        />
    </div>
</template>

<script lang="ts" setup>
import { hasPerms } from '@/components/auth/auth';
import { createPolicy, describePolicy, policySummary, SEVERITY_LABEL_KEYS, SEVERITY_TAG_TYPES, usePolicySchema } from '@/components/policy-builder';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import TagCodePath from '../ops/component/TagCodePath.vue';
import ProcdefEdit from './ProcdefEdit.vue';
import { procdefApi } from './api';
import FlowDesignDrawer from './components/flowdesign/FlowDesignDrawer.vue';
import { ProcdefStatus } from './enums';
import type { Procdef } from './types';
import type { Severity } from '@/components/policy-builder';

const { t } = useI18n();

const perms = {
    save: 'flow:procdef:save',
    del: 'flow:procdef:del',
};

const searchItems = [SearchItem.input('name', 'common.name'), SearchItem.input('defKey', 'key')];
const columns = [
    TableColumn.new('name', 'common.name'),
    TableColumn.new('defKey', 'flow.defKey'),
    TableColumn.new('status', 'common.status').typeTag(ProcdefStatus),
    TableColumn.new('triggerPolicy', 'flow.triggeringCondition').isSlot().setMinWidth('210px'),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('codePaths', 'tag.relateTag').isSlot().setMinWidth('250px'),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
];

// 默认收起备注与创建者：实测 8 列合计 1413px 已超 1200px 视口下的容器 1138px，一进页面就带横向滚动条，
// 而决策需要的「触发策略」「生效资源」两列会被挤到屏外；需要时在右上角「表格配置」里按需勾选
const hiddenByDefault = ['remark', 'creator'];
columns.forEach((column) => {
    if (hiddenByDefault.includes(column.prop)) column.show = 0;
});

const { policySchema, load: loadPolicySchema } = usePolicySchema(() => procdefApi.policySchema.request());

const policySummaryOf = (data: Procdef) => policySummary(data.triggerPolicy);

// 悬停内容是策略的自然语句摘要，与编辑页「规则解读」同一套派生，两处不会说法不一。
// 按场景分组并带上前缀：一条流程可以同时管 DBMS/Redis/机器，混成一列时无法判断某条规则属于谁
const policyLinesOf = (data: Procdef) => {
    const lines: string[] = [];
    for (const item of policySummaryOf(data)) {
        if (item.bizType !== '*') lines.push(scenarioText(item));
        lines.push(...describePolicy(data.triggerPolicy ?? createPolicy(), policySchema.value, t, item.bizType));
    }
    return lines;
};

// 场景名与级别拼成一句话（如「DBMS-执行SQL · 需审批」），文案缺失时回退原始 key 不阻塞新场景
const scenarioText = (item: { bizType: string; severity: Severity }) => {
    const bizKey = item.bizType === '*' ? 'flow.policy.anyScenario' : `flow.bizTypeName.${item.bizType}`;
    const bizLabel = t(bizKey);
    return `${bizLabel === bizKey ? item.bizType : bizLabel} · ${t(SEVERITY_LABEL_KEYS[item.severity])}`;
};

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.save, perms.del]) as Record<string, boolean>;
const actionColumn = TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(160).noShowOverflowTooltip().alignCenter();

const pageTableRef = useTemplateRef<{ search: () => void }>('pageTableRef');
const state = reactive({
    /**
     * 选中的数据
     */
    selectionData: [],
    /**
     * 查询条件
     */
    query: {
        name: '',
        pageNum: 1,
        pageSize: 0,
    },
    flowDefEditor: {
        title: '',
        visible: false,
        data: null as Procdef | null,
    },
    flowDesignEditor: {
        title: '',
        disabled: false,
        visible: false,
        procdefId: 0,
        data: null as Record<string, unknown> | null,
    },
});

const { selectionData, query, flowDefEditor, flowDesignEditor } = toRefs(state);

onMounted(async () => {
    if (Object.keys(actionBtns).length > 0) {
        columns.push(actionColumn);
    }
    await loadPolicySchema();
});

const search = async () => {
    pageTableRef.value?.search();
};

const onEditFlowDef = (data: Procdef | false) => {
    if (!data) {
        state.flowDefEditor.data = null;
        state.flowDefEditor.title = useI18nCreateTitle('flow.procdef');
    } else {
        state.flowDefEditor.data = data;
        state.flowDefEditor.title = useI18nEditTitle('flow.procdef');
    }
    state.flowDefEditor.visible = true;
};

const handleValChange = () => {
    state.flowDefEditor.visible = false;
    search();
};

const onDeleteProcdef = async () => {
    try {
        if (!(await useI18nDeleteConfirm(state.selectionData.map((x: Procdef) => x.name).join(', ')))) {
            return;
        }
        await procdefApi.del.request({ id: state.selectionData.map((x: Procdef) => x.id).join(',') });
        Msg.deleteSuccess();
        search();
    } catch (err) {
        //
    }
};

const onShowFlowDesign = async (data: Procdef) => {
    state.flowDesignEditor.procdefId = data.id;
    state.flowDesignEditor.data = await procdefApi.flowDef.request({ id: data.id });
    state.flowDesignEditor.visible = true;
};

const onSaveFlowDesign = async (data: unknown) => {
    await procdefApi.saveFlowDef.request({ id: state.flowDesignEditor.procdefId, flow: data });
    Msg.saveSuccess();
    state.flowDesignEditor.visible = false;
};
</script>
<style lang="scss" scoped>
.policy-summary {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
}
</style>
