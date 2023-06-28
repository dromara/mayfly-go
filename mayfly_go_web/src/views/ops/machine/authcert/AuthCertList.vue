<template>
    <div>
        <page-table :query="state.queryConfig" v-model:query-form="query" :show-choose-column="true"
            v-model:choose-data="state.chooseData" :data="authcerts" :columns="state.columns" :total="total"
            v-model:page-size="query.pageSize" v-model:page-num="query.pageNum" @pageChange="search()">

            <template #queryRight>
                <el-button type="primary" icon="plus" @click="edit(false)">添加</el-button>
                <el-button :disabled="!chooseData" @click="edit(chooseData)" type="primary" icon="edit">编辑
                </el-button>
                <el-button :disabled="!chooseData" @click="deleteAc(chooseData)" type="danger" icon="delete">删除
                </el-button>

            </template>

            <template #authMethod="{ data }">
                <el-tag v-if="data.authMethod == 1" type="success" size="small">密码</el-tag>
                <el-tag v-if="data.authMethod == 2" size="small">密钥</el-tag>
            </template>
        </page-table>

        <auth-cert-edit :title="editor.title" v-model:visible="editor.visible" :data="editor.authcert"
            @val-change="editChange" />
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, onMounted } from 'vue';
import AuthCertEdit from './AuthCertEdit.vue';
import { authCertApi } from '../api';
import { ElMessage, ElMessageBox } from 'element-plus';
import PageTable from '@/components/pagetable/PageTable.vue'
import { TableColumn, TableQuery } from '@/components/pagetable';

const state = reactive({
    query: {
        pageNum: 1,
        pageSize: 10,
        name: null,
    },
    queryConfig: [
        TableQuery.text("name", "凭证名称"),
    ],
    columns: [
        TableColumn.new("name", "名称"),
        TableColumn.new("authMethod", "认证方式").setSlot("authMethod"),
        TableColumn.new("remark", "备注"),
        TableColumn.new("creator", "创建人"),
        TableColumn.new("createTime", "创建时间").isTime(),
        TableColumn.new("creator", "修改者"),
        TableColumn.new("createTime", "修改时间").isTime(),
    ],
    total: 0,
    authcerts: [],
    chooseData: null as any,
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

const {
    query,
    total,
    authcerts,
    chooseData,
    editor,
} = toRefs(state)

onMounted(() => {
    search();
});

const search = async () => {
    let res = await authCertApi.list.request(state.query);
    state.authcerts = res.list;
    state.total = res.total;
};

const editChange = () => {
    ElMessage.success('保存成功');
    state.chooseData = null;
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
        await ElMessageBox.confirm(`确定删除该授权凭证?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await authCertApi.delete.request({ id: data.id });
        ElMessage.success('删除成功');
        state.chooseData = null;
        search();
    } catch (err) { }

}
</script>
<style lang="scss"></style>
