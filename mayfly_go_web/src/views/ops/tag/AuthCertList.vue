<template>
    <div>
        <page-table
            ref="pageTableRef"
            :page-api="resourceAuthCertApi.listByQuery"
            :search-items="state.searchItems"
            v-model:query-form="query"
            :columns="state.columns"
        >
            <template #tableHeader>
                <el-button v-auth="'authcert:save'" type="primary" icon="plus" @click="edit(false)">添加</el-button>
            </template>

            <template #action="{ data }">
                <el-button v-auth="'authcert:save'" @click="edit(data)" type="primary" link>编辑</el-button>

                <el-button v-auth="'authcert:del'" @click="deleteAc(data)" type="danger" link>删除</el-button>
            </template>
        </page-table>

        <ResourceAuthCertEdit
            v-model:visible="editor.visible"
            :auth-cert="editor.authcert"
            @confirm="confirmSave"
            :disable-type="state.disableAuthCertType"
            :disable-ciphertext-type="state.disableAuthCertCiphertextType"
            :resource-edit="false"
        />
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, onMounted, ref, Ref } from 'vue';
import { resourceAuthCertApi } from './api';
import { ElMessage, ElMessageBox } from 'element-plus';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { SearchItem } from '@/components/SearchForm';
import { AuthCertCiphertextTypeEnum, AuthCertTypeEnum } from './enums';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import ResourceAuthCertEdit from '../component/ResourceAuthCertEdit.vue';

const pageTableRef: Ref<any> = ref(null);
const state = reactive({
    query: {
        pageNum: 1,
        pageSize: 0,
        name: null,
    },
    searchItems: [
        SearchItem.input('name', '凭证名称'),
        SearchItem.select('type', '凭证类型').withEnum(AuthCertTypeEnum),
        SearchItem.select('ciphertextType', '密文类型').withEnum(AuthCertCiphertextTypeEnum),
    ],
    columns: [
        TableColumn.new('name', '名称'),
        TableColumn.new('type', '凭证类型').typeTag(AuthCertTypeEnum),
        TableColumn.new('username', '用户名'),
        TableColumn.new('ciphertextType', '密文类型').typeTag(AuthCertCiphertextTypeEnum),
        TableColumn.new('resourceType', '资源类型').typeTag(TagResourceTypeEnum),
        TableColumn.new('resourceCode', '资源编号'),
        TableColumn.new('remark', '备注'),
        TableColumn.new('creator', '创建人'),
        TableColumn.new('createTime', '创建时间').isTime(),
        TableColumn.new('modifier', '修改者'),
        TableColumn.new('updateTime', '修改时间').isTime(),
        TableColumn.new('action', '操作').isSlot().fixedRight().setMinWidth(120).alignCenter(),
    ],
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
    disableAuthCertType: [] as any,
    disableAuthCertCiphertextType: [] as any,
});

const { query, editor } = toRefs(state);

onMounted(() => {});

const search = async () => {
    pageTableRef.value.search();
};

const edit = (data: any) => {
    state.disableAuthCertType = [];
    state.disableAuthCertCiphertextType = [];
    if (data) {
        state.editor.authcert = data;
        //  如果数据为公共授权凭证，则不允许修改凭证类型
        if (data.type == AuthCertTypeEnum.Public.value) {
            state.disableAuthCertType = [AuthCertTypeEnum.Private.value, AuthCertTypeEnum.PrivateDefault.value, AuthCertTypeEnum.Privileged.value];
            state.disableAuthCertCiphertextType = [AuthCertCiphertextTypeEnum.Public.value];
        } else {
            // 如果非公共凭证，也无法修改为公共凭证
            state.disableAuthCertType = [AuthCertTypeEnum.Public.value];
        }
    } else {
        state.editor.authcert = {
            type: AuthCertTypeEnum.Public.value,
            ciphertextType: AuthCertCiphertextTypeEnum.Password.value,
            extra: {},
        };
    }

    state.editor.visible = true;
};

const confirmSave = async (authCert: any) => {
    await resourceAuthCertApi.save.request(authCert);
    ElMessage.success('保存成功');
    state.editor.visible = false;
    search();
};

const deleteAc = async (data: any) => {
    try {
        await ElMessageBox.confirm(`确定删除该【${data.name}授权凭证?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await resourceAuthCertApi.delete.request({ id: data.id });
        ElMessage.success('删除成功');
        search();
    } catch (err) {
        //
    }
};
</script>
<style lang="scss"></style>
