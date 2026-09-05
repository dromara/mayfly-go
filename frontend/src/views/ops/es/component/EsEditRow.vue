<template>
    <el-drawer
        :append-to-body="false"
        :title="`${model.isAdd ? t('common.add') : t('common.edit')} ${model.idxName}`"
        v-model="visible"
        :destroy-on-close="false"
        :close-on-click-modal="false"
        :close-on-press-escape="false"
        size="50%"
    >
        <el-auto-resizer>
            <template #default="{ height, width }">
                <auto-form v-model="editForm" :items="editItems" label-width="auto">
                    <template #doc>
                        <monaco-editor v-model="model.doc" language="json" :height="height - 40 + 'px'" :options="{ wordWrap: 'on', tabSize: 2 }" />
                    </template>
                </auto-form>
            </template>
        </el-auto-resizer>
        <template #footer>
            <el-button size="small" @click="visible = false">{{ t('common.cancel') }}</el-button>
            <el-button size="small" v-auth="perms.saveData" @click="onSaveDoc" :loading="loading" type="primary">{{ t('common.confirm') }}</el-button>
        </template>
    </el-drawer>
</template>

<script setup lang="ts">
import { Msg } from '@/hooks/useI18n';
import { esApi } from '@/views/ops/es/api';
import type { AutoFormItem } from '@/components/auto-form';
import { computed, defineAsyncComponent, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

const { t } = useI18n();

const perms = {
    saveData: 'es:data:save',
};

const visible = defineModel<boolean>('visible');
const loading = ref(false);
const _id = ref('');

/** 文档编辑表单声明（_id 独立 ref 经 computed 代理；doc 编辑器为 custom 插槽） */
const editForm = computed({
    get: () => ({ _id: _id.value }),
    set: (v: { _id: string }) => (_id.value = v._id),
});
const editItems = computed<AutoFormItem[]>(() => [
    { prop: '_id', label: '_id', props: { autocomplete: 'off' }, placeholder: 'es.specifyIdAdd', disabled: () => model.value._id != '' },
    { prop: 'doc', type: 'custom' },
]);

interface Params {
    isAdd: boolean;
    instId: number;
    doc: string;
    idxName: string;
    _id: string;
}
const model = defineModel<Params>({ required: true });
const emit = defineEmits(['success']);

const getZeroValueByProperties = async () => {
    // 根据mapping字段赋值
    const mp = await esApi.proxyReq<Record<string, { mappings: { properties: Record<string, { type?: string }> } }>>(
        'get',
        model.value.instId,
        `/${model.value.idxName}/_mappings`
    );
    const properties = mp[model.value.idxName].mappings.properties;
    const data: Record<string, unknown> = {};

    for (const key in properties) {
        const item = properties[key];
        switch (item.type) {
            case 'object':
            case 'nested':
            case 'flattened':
                data[key] = {};
                break;
            case 'long':
            case 'short':
            case 'byte':
            case 'double':
            case 'float':
            case 'half_float':
            case 'scaled_float':
                data[key] = 0;
                break;
            case 'boolean':
                data[key] = false;
                break;
            default:
                data[key] = '';
                break;
        }
    }

    return data;
};

watch(visible, async (newValue) => {
    if (!newValue) {
        model.value._id = '';
        model.value.doc = '';
        _id.value = '';
        loading.value = false;
    } else {
        if (model.value._id) {
            _id.value = model.value._id;
        }
        if (!model.value.doc) {
            model.value.doc = JSON.stringify(await getZeroValueByProperties(), null, 2);
        }
    }
});

const onSaveDoc = async () => {
    loading.value = true;
    let doc = model.value.doc;
    let data;
    try {
        data = JSON.parse(doc);
    } catch (error) {
        Msg.error('es.docJsonError');
        loading.value = false;
        return;
    }

    // 如果数据中带有_id，则删除_id
    if (data._id) {
        delete data._id;
    }
    // 2 秒后关闭loading，避免接口报错后不关闭loading
    setTimeout(async () => {
        loading.value = false;
    }, 2000);

    await esApi.proxyReq('post', model.value.instId, `/${model.value.idxName}/_doc/${_id.value}`, data);
    Msg.saveSuccess();

    setTimeout(() => {
        visible.value = false;
        emit('success');
    }, 500);
};
</script>

<style scoped lang="scss"></style>
