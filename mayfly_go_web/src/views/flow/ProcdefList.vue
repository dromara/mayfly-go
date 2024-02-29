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
                <el-button v-auth="perms.save" type="primary" icon="plus" @click="editFlowDef(false)">添加</el-button>
                <el-button v-auth="perms.del" :disabled="state.selectionData.length < 1" @click="deleteProcdef()" type="danger" icon="delete">删除</el-button>
            </template>

            <template #tasks="{ data }">
                <el-link @click="showProcdefTasks(data)" icon="view" type="primary" :underline="false"> </el-link>
            </template>

            <template #action="{ data }">
                <el-button link v-if="actionBtns[perms.save]" @click="editFlowDef(data)" type="primary">编辑</el-button>
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
import { ElMessage, ElMessageBox } from 'element-plus';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import { SearchItem } from '@/components/SearchForm';
import ProcdefEdit from './ProcdefEdit.vue';
import ProcdefTasks from './components/ProcdefTasks.vue';
import { ProcdefStatus } from './enums';

const perms = {
    save: 'flow:procdef:save',
    del: 'flow:procdef:del',
};

const searchItems = [SearchItem.input('name', '名称'), SearchItem.input('defKey', 'key')];
const columns = [
    TableColumn.new('name', '名称'),
    TableColumn.new('defKey', 'key'),
    TableColumn.new('status', '状态').typeTag(ProcdefStatus),
    TableColumn.new('remark', '备注'),
    TableColumn.new('tasks', '审批节点').isSlot().alignCenter().setMinWidth(60),
    TableColumn.new('creator', '创建账号'),
    TableColumn.new('createTime', '创建时间').isTime(),
];

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.save, perms.del]);
const actionColumn = TableColumn.new('action', '操作').isSlot().fixedRight().setMinWidth(160).noShowOverflowTooltip().alignCenter();

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
        title: '新建流程定义',
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
    state.flowTasksDialog.title = procdef.name + '-审批节点';
    state.flowTasksDialog.visible = true;
};

const editFlowDef = (data: any) => {
    if (!data) {
        state.flowDefEditor.data = null;
        state.flowDefEditor.title = '新建流程定义';
    } else {
        state.flowDefEditor.data = data;
        state.flowDefEditor.title = '编辑流程定义';
    }
    state.flowDefEditor.visible = true;
};

const valChange = () => {
    state.flowDefEditor.visible = false;
    search();
};

const deleteProcdef = async () => {
    try {
        await ElMessageBox.confirm(`确定删除【${state.selectionData.map((x: any) => x.name).join(', ')}】的流程定义?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await procdefApi.del.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
        ElMessage.success('删除成功');
        search();
    } catch (err) {
        //
    }
};
</script>
<style lang="scss"></style>
