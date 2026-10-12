<template>
    <div class="h-full">
        <data-table
            ref="dataTableRef"
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

            <!-- 行内操作由表格统一渲染（见 columns 里的 setActions），列宽与按钮同源 -->
        </data-table>

        <ResourceAuthCertEdit
            :title="editor.title"
            v-model:visible="editor.visible"
            :auth-cert="editor.authcert"
            @confirm="onConfirmSave"
            @cancel="editor.authcert = {} as ResourceAuthCert"
            :disable-type="state.disableAuthCertType"
            :disable-ciphertext-type="state.disableAuthCertCiphertextType"
            :resource-edit="false"
        />
    </div>
</template>

<script lang="ts" setup>
import { ResourceTypeEnum, TagResourceTypeEnum } from '@/common/commonEnum';
import EnumValue from '@/common/Enum';
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import ResourceAuthCertEdit from '../component/ResourceAuthCertEdit.vue';
import { resourceAuthCertApi } from './api';
import { AuthCertCiphertextTypeEnum, AuthCertTypeEnum } from './enums';
import type { ResourceAuthCert } from '@/types/common';

const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');
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
        TableColumn.new('resourceCode', 'ac.resourceCode').isSlot(),
        TableColumn.new('remark', 'common.remark'),
        TableColumn.new('creator', 'common.creator'),
        TableColumn.new('createTime', 'common.createTime').isTime(),
        TableColumn.new('modifier', 'common.modifier'),
        TableColumn.new('updateTime', 'common.updateTime').isTime(),
        TableColumn.new('action', 'common.operation')
            .fixedRight()
            .setActions<ResourceAuthCert>([
                { key: 'edit', label: 'common.edit', type: 'primary', perm: 'authcert:save', onClick: (row) => onEdit(row) },
                { key: 'del', label: 'common.delete', type: 'danger', perm: 'authcert:del', onClick: (row) => onDeleteAc(row) },
            ])
            .alignCenter(),
    ],
    paramsDialog: {
        visible: false,
        config: null as ResourceAuthCert | null,
        params: {} as Record<string, unknown>,
        paramsFormItem: [] as Record<string, unknown>[],
    },
    editor: {
        title: '',
        visible: false,
        authcert: {} as ResourceAuthCert,
    },
    disableAuthCertType: [] as number[],
    disableAuthCertCiphertextType: [] as number[],
});

const { query, editor } = toRefs(state);

onMounted(() => {});

const search = async () => {
    dataTableRef.value?.search();
};

const onEdit = (data: ResourceAuthCert | false) => {
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
        } as ResourceAuthCert;
    }

    state.editor.visible = true;
};

const onConfirmSave = async (authCert: ResourceAuthCert) => {
    await resourceAuthCertApi.save.request(authCert);
    Msg.saveSuccess();
    state.editor.visible = false;
    search();
};

const onDeleteAc = async (data: ResourceAuthCert) => {
    try {
        if (!(await useI18nDeleteConfirm(data.name))) {
            return;
        }
        await resourceAuthCertApi.delete.request({ id: data.id });
        Msg.deleteSuccess();
        search();
    } catch (err) {
        //
    }
};
</script>
<style lang="scss"></style>
