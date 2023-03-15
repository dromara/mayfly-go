<template>
    <div class="role-list">
        <el-card>
            <div>
                <el-button type="primary" icon="plus" @click="edit(false)">添加</el-button>
                <el-button :disabled="chooseId == null" @click="edit(chooseData)" type="primary" icon="edit">编辑
                </el-button>
                <el-button :disabled="chooseId == null" @click="deleteAc(chooseData)" type="danger" icon="delete">删除
                </el-button>

                <div style="float: right">
                    <el-select v-model="query.type" placeholder="请选择标签" @clear="search" filterable clearable>
                        <el-option label="" value="item"> </el-option>
                    </el-select>
                    <el-input class="ml5" placeholder="请输入凭证名称" style="width: 150px" v-model="query.name" @clear="search"
                        plain clearable></el-input>
                    <el-button class="ml5" @click="search" type="success" icon="search"></el-button>
                </div>
            </div>

            <el-table :data="authcerts" @current-change="choose" ref="table" style="width: 100%">
                <el-table-column label="选择" width="55px">
                    <template #default="scope">
                        <el-radio v-model="chooseId" :label="scope.row.id">
                            <i></i>
                        </el-radio>
                    </template>
                </el-table-column>
                <el-table-column prop="name" label="名称" min-width="60px" show-overflow-tooltip></el-table-column>
                <el-table-column prop="authMethod" label="认证方式" min-width="50px">
                    <template #default="scope">
                        <el-tag v-if="scope.row.authMethod == 1" type="success" size="small">密码</el-tag>
                        <el-tag v-if="scope.row.authMethod == 2" size="small">密钥</el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="remark" label="备注" min-width="100px" show-overflow-tooltip>
                </el-table-column>
                <el-table-column prop="creator" label="创建人" min-width="60px"></el-table-column>
                <el-table-column prop="createTime" label="创建时间" min-width="100px">
                    <template #default="scope">
                        {{ dateFormat(scope.row.createTime) }}
                    </template>
                </el-table-column>
                <el-table-column prop="modifier" label="修改者" min-width="60px" show-overflow-tooltip></el-table-column>
                <el-table-column prop="updateTime" label="更新时间" min-width="100px">
                    <template #default="scope">
                        {{ dateFormat(scope.row.updateTime) }}
                    </template>
                </el-table-column>
            </el-table>
            <el-row style="margin-top: 20px" type="flex" justify="end">
                <el-pagination style="text-align: right" @current-change="handlePageChange" :total="total"
                    layout="prev, pager, next, total, jumper" v-model:current-page="query.pageNum"
                    :page-size="query.pageSize"></el-pagination>
            </el-row>
        </el-card>

        <auth-cert-edit :title="editor.title" v-model:visible="editor.visible" :data="editor.authcert"
            @val-change="editChange" />
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, onMounted } from 'vue';
import AuthCertEdit from './AuthCertEdit.vue';
import { authCertApi } from '../api';
import { ElMessage, ElMessageBox } from 'element-plus';
import { dateFormat } from '@/common/utils/date';

const state = reactive({
    query: {
        pageNum: 1,
        pageSize: 10,
        name: null,
        type: null,
    },
    total: 0,
    authcerts: [],
    chooseId: null,
    chooseData: null,
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
    chooseId,
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

const handlePageChange = (curPage: number) => {
    state.query.pageNum = curPage;
    search();
};

const choose = (item: any) => {
    if (!item) {
        return;
    }
    state.chooseId = item.id;
    state.chooseData = item;
};

const editChange = () => {
    ElMessage.success('保存成功');
    state.chooseId = null;
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
        state.chooseId = null;
        search();
    } catch (err) { }

}
</script>
<style lang="scss"></style>
