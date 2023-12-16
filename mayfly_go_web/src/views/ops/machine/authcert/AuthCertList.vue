<template>
    <div>
        <page-table
            ref="pageTableRef"
            :page-api="authCertApi.list"
            :search-items="state.searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="state.columns"
        >
            <template #tableHeader>
                <el-button type="primary" icon="plus" @click="edit(false)">添加</el-button>
                <el-button :disabled="selectionData.length < 1" @click="deleteAc(selectionData)" type="danger" icon="delete">删除 </el-button>
            </template>

            <template #action="{ data }">
                <el-button @click="edit(data)" type="primary" link>编辑 </el-button>
            </template>
        </page-table>

        <auth-cert-edit :title="editor.title" v-model:visible="editor.visible" :data="editor.authcert" @val-change="editChange" />
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, onMounted, ref, Ref } from 'vue';
import AuthCertEdit from './AuthCertEdit.vue';
import { authCertApi } from '../api';
import { ElMessage, ElMessageBox } from 'element-plus';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { AuthMethodEnum } from '../enums';
import { SearchItem } from '@/components/SearchForm';

const pageTableRef: Ref<any> = ref(null);
const state = reactive({
    query: {
        pageNum: 1,
        pageSize: 0,
        name: null,
    },
    searchItems: [SearchItem.input('name', '凭证名称')],
    columns: [
        TableColumn.new('name', '名称'),
        TableColumn.new('authMethod', '认证方式').typeTag(AuthMethodEnum),
        TableColumn.new('remark', '备注'),
        TableColumn.new('creator', '创建人'),
        TableColumn.new('createTime', '创建时间').isTime(),
        TableColumn.new('creator', '修改者'),
        TableColumn.new('createTime', '修改时间').isTime(),
        TableColumn.new('action', '操作').isSlot().fixedRight().setMinWidth(65).alignCenter(),
    ],
    selectionData: [],
    paramsDialog: {
        visible: false,
        config: null as any,
        params: {},
        paramsFormItem: [] as any,
    },
    editor: {
        title: '授权凭证保存',
        visible: false,
        authcert: {},
    },
});

const { query, selectionData, editor } = toRefs(state);

onMounted(() => {});

const search = async () => {
    pageTableRef.value.search();
};

const editChange = () => {
    ElMessage.success('保存成功');
    search();
};

const edit = (data: any) => {
    if (data) {
        state.editor.authcert = data;
    } else {
        state.editor.authcert = false;
    }

    state.editor.visible = true;
};

const deleteAc = async (data: any) => {
    try {
        await ElMessageBox.confirm(`确定删除该【${data.map((x: any) => x.name).join(', ')}授权凭证?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await authCertApi.delete.request({ id: data.map((x: any) => x.id).join(',') });
        ElMessage.success('删除成功');
        search();
    } catch (err) {
        //
    }
};
</script>
<style lang="scss"></style>
