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
            <template #tableHeader>
                <el-button v-auth="perms.save" type="primary" icon="plus" @click="onEditFlowDef(false)">{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.del" :disabled="state.selectionData.length < 1" @click="onDeleteProcdef()" type="danger" icon="delete">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #codePaths="{ data }">
                <TagCodePath :path="data.tags" />
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

const { t } = useI18n();

const perms = {
    save: 'flow:procdef:save',
    del: 'flow:procdef:del',
};

const searchItems = [SearchItem.input('name', 'common.name'), SearchItem.input('defKey', 'key')];
const columns = [
    TableColumn.new('name', 'common.name'),
    TableColumn.new('defKey', 'Key'),
    TableColumn.new('status', 'common.status').typeTag(ProcdefStatus),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('codePaths', 'tag.relateTag').isSlot().setMinWidth('250px'),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
];

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

onMounted(() => {
    if (Object.keys(actionBtns).length > 0) {
        columns.push(actionColumn);
    }
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
        await useI18nDeleteConfirm(state.selectionData.map((x: Procdef) => x.name).join(', '));
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
    state.flowDesignEditor.title = t('flow.procDesign');
    state.flowDesignEditor.visible = true;
};

const onSaveFlowDesign = async (data: unknown) => {
    await procdefApi.saveFlowDef.request({ id: state.flowDesignEditor.procdefId, flow: data });
    Msg.saveSuccess();
    state.flowDesignEditor.visible = false;
};
</script>
<style lang="scss"></style>
