<template>
    <div>
        <page-table
            ref="pageTableRef"
            :page-api="resourceAuthCertApi.listByQuery"
            :search-items="state.searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="state.columns"
        >
            <template #tableHeader>
                <el-button v-auth="'authcert:save'" type="primary" icon="plus" @click="edit(false)">添加</el-button>
                <el-button v-auth="'authcert:del'" :disabled="disabledDelBtn" @click="deleteAc(selectionData)" type="danger" icon="delete">删除 </el-button>
            </template>

            <template #action="{ data }">
                <el-button v-auth="'authcert:save'" v-if="data.type == AuthCertTypeEnum.Public.value" @click="edit(data)" type="primary" link>编辑 </el-button>
            </template>
        </page-table>

        <ResourceAuthCertEdit
            v-model:visible="editor.visible"
            :auth-cert="editor.authcert"
            @confirm="confirmSave"
            :resource-edit="false"
            :disable-ciphertext-type="[AuthCertCiphertextTypeEnum.Public.value]"
            :disable-type="[AuthCertTypeEnum.Private.value, AuthCertTypeEnum.PrivateDefault.value, AuthCertTypeEnum.Privileged.value]"
        />
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, onMounted, ref, Ref, computed } from 'vue';
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
        TableColumn.new('username', '用户名'),
        TableColumn.new('ciphertextType', '密文类型').typeTag(AuthCertCiphertextTypeEnum),
        TableColumn.new('type', '凭证类型').typeTag(AuthCertTypeEnum),
        TableColumn.new('resourceType', '资源类型').typeTag(TagResourceTypeEnum),
        TableColumn.new('resourceCode', '资源编号'),
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

const disabledDelBtn = computed(() => {
    return state.selectionData.length < 1 || state.selectionData.find((item: any) => item.type != AuthCertTypeEnum.Public.value);
});

const edit = (data: any) => {
    if (data) {
        state.editor.authcert = data;
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
        await ElMessageBox.confirm(`确定删除该【${data.map((x: any) => x.name).join(', ')}授权凭证?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await resourceAuthCertApi.delete.request({ id: data.map((x: any) => x.id).join(',') });
        ElMessage.success('删除成功');
        search();
    } catch (err) {
        //
    }
};
</script>
<style lang="scss"></style>
