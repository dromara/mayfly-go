<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
            :page-api="resourceAuthCertApi.listByQuery"
            :search-items="state.searchItems"
            v-model:query-form="query"
            :columns="state.columns"
        >
            <template #tableHeader>
                <el-button v-auth="'authcert:save'" type="primary" icon="plus" @click="onEdit(false)">{{ $t('common.create') }}</el-button>
            </template>

            <template #resourceCode="{ data }">
                <SvgIcon
                    :name="EnumValue.getEnumByValue(TagResourceTypeEnum, data.resourceType)?.extra.icon"
                    :color="EnumValue.getEnumByValue(TagResourceTypeEnum, data.resourceType)?.extra.iconColor"
                />
                {{ data.resourceCode }}
            </template>

            <template #action="{ data }">
                <el-button v-auth="'authcert:save'" @click="onEdit(data)" type="primary" link>{{ $t('common.edit') }}</el-button>

                <el-button v-auth="'authcert:del'" @click="onDeleteAc(data)" type="danger" link>{{ $t('common.delete') }}</el-button>
            </template>
        </page-table>

        <ResourceAuthCertEdit
            :title="editor.title"
            v-model:visible="editor.visible"
            :auth-cert="editor.authcert"
            @confirm="onConfirmSave"
            @cancel="editor.authcert = {}"
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
import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import ResourceAuthCertEdit from '../component/ResourceAuthCertEdit.vue';
import EnumValue from '@/common/Enum';
import { useI18nCreateTitle, useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nEditTitle, useI18nSaveSuccessMsg } from '@/hooks/useI18n';

const pageTableRef: Ref<any> = ref(null);
const state = reactive({
    query: {
        pageNum: 1,
        pageSize: 0,
        name: null,
    },
    searchItems: [
        SearchItem.input('resourceCode', 'ac.resourceCode'),
        SearchItem.input('name', 'ac.credentialName'),
        SearchItem.select('resourceType', 'ac.resourceType').withEnum(ResourceTypeEnum),
        SearchItem.select('type', 'ac.credentialType').withEnum(AuthCertTypeEnum),
        SearchItem.select('ciphertextType', 'ac.ciphertextType').withEnum(AuthCertCiphertextTypeEnum),
    ],
    columns: [
        TableColumn.new('name', 'common.name'),
        TableColumn.new('type', 'ac.credentialType').typeTag(AuthCertTypeEnum),
        TableColumn.new('username', 'common.username'),
        TableColumn.new('ciphertextType', 'ac.ciphertextType').typeTag(AuthCertCiphertextTypeEnum),
        TableColumn.new('resourceCode', 'ac.resourceCode').isSlot().setAddWidth(30),
        TableColumn.new('remark', 'common.remark'),
        TableColumn.new('creator', 'common.creator'),
        TableColumn.new('createTime', 'common.createTime').isTime(),
        TableColumn.new('modifier', 'common.modifier'),
        TableColumn.new('updateTime', 'common.updateTime').isTime(),
        TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(120).alignCenter(),
    ],
    paramsDialog: {
        visible: false,
        config: null as any,
        params: {},
        paramsFormItem: [] as any,
    },
    editor: {
        title: '',
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

const onEdit = (data: any) => {
    state.disableAuthCertType = [];
    state.disableAuthCertCiphertextType = [];
    if (data) {
        state.editor.title = useI18nEditTitle('ac.ac');
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
        state.editor.title = useI18nCreateTitle('ac.ac');
        state.editor.authcert = {
            type: AuthCertTypeEnum.Public.value,
            ciphertextType: AuthCertCiphertextTypeEnum.Password.value,
            extra: {},
        };
    }

    state.editor.visible = true;
};

const onConfirmSave = async (authCert: any) => {
    await resourceAuthCertApi.save.request(authCert);
    useI18nSaveSuccessMsg();
    state.editor.visible = false;
    search();
};

const onDeleteAc = async (data: any) => {
    try {
        await useI18nDeleteConfirm(data.name);
        await resourceAuthCertApi.delete.request({ id: data.id });
        useI18nDeleteSuccessMsg();
        search();
    } catch (err) {
        //
    }
};
</script>
<style lang="scss"></style>
