<template>
    <div>
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
                <el-button v-auth="perms.save" type="primary" icon="plus" @click="editFlowDef(false)">{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.del" :disabled="state.selectionData.length < 1" @click="deleteProcdef()" type="danger" icon="delete">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #tasks="{ data }">
                <el-link @click="showProcdefTasks(data)" icon="view" type="primary" :underline="false"> </el-link>
            </template>

            <template #codePaths="{ data }">
                <TagCodePath :path="data.tags?.map((tag: any) => tag.codePath)" />
            </template>

            <template #action="{ data }">
                <el-button link v-if="actionBtns[perms.save]" @click="editFlowDef(data)" type="primary">{{ $t('common.edit') }}</el-button>
            </template>
        </page-table>

        <el-dialog v-model="flowTasksDialog.visible" :title="flowTasksDialog.title">
            <procdef-tasks :tasks="flowTasksDialog.tasks" />
        </el-dialog>

        <procdef-edit v-model:visible="flowDefEditor.visible" :title="flowDefEditor.title" v-model:data="flowDefEditor.data" @val-change="valChange()" />
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, Ref } from 'vue';
import { procdefApi } from './api';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import { SearchItem } from '@/components/SearchForm';
import ProcdefEdit from './ProcdefEdit.vue';
import ProcdefTasks from './components/ProcdefTasks.vue';
import { ProcdefStatus } from './enums';
import TagCodePath from '../ops/component/TagCodePath.vue';
import { useI18nCreateTitle, useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nEditTitle } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';

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
    TableColumn.new('tasks', 'flow.approvalNode').isSlot().alignCenter().setMinWidth(60),
    TableColumn.new('codePaths', 'tag.relateTag').isSlot().setMinWidth('250px'),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
];

// 该用户拥有的的操作列按钮权限
const actionBtns: any = hasPerms([perms.save, perms.del]);
const actionColumn = TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(160).noShowOverflowTooltip().alignCenter();

const pageTableRef: Ref<any> = ref(null);
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
        data: null as any,
    },
    flowTasksDialog: {
        title: '',
        visible: false,
        tasks: '',
    },
});

const { selectionData, query, flowDefEditor, flowTasksDialog } = toRefs(state);

onMounted(() => {
    if (Object.keys(actionBtns).length > 0) {
        columns.push(actionColumn);
    }
});

const search = async () => {
    pageTableRef.value.search();
};

const showProcdefTasks = (procdef: any) => {
    state.flowTasksDialog.tasks = procdef.tasks;
    state.flowTasksDialog.title = procdef.name + ' - ' + t('flow.approvalNode');
    state.flowTasksDialog.visible = true;
};

const editFlowDef = (data: any) => {
    if (!data) {
        state.flowDefEditor.data = null;
        state.flowDefEditor.title = useI18nCreateTitle('flow.procdef');
    } else {
        state.flowDefEditor.data = data;
        state.flowDefEditor.title = useI18nEditTitle('flow.procdef');
    }
    state.flowDefEditor.visible = true;
};

const valChange = () => {
    state.flowDefEditor.visible = false;
    search();
};

const deleteProcdef = async () => {
    try {
        await useI18nDeleteConfirm(state.selectionData.map((x: any) => x.name).join(', '));
        await procdefApi.del.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
        useI18nDeleteSuccessMsg();
        search();
    } catch (err) {
        //
    }
};
</script>
<style lang="scss"></style>
