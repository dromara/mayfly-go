<!-- es 编辑索引 -->

<template>
    <el-drawer
        :append-to-body="false"
        :title="t('es.addIndex')"
        v-model="visible"
        size="50%"
        :destroy-on-close="false"
        :close-on-click-modal="false"
        :close-on-press-escape="false"
        class="es-edit-index h-full"
    >
        <el-auto-resizer>
            <template #default="{ height, width }">
                <auto-form ref="formRef" v-model="formData" :items="items">
                    <template #copyIdxName>
                        <el-space>
                            <el-select v-model="formData.copyIdxName" style="width: 200px" filterable>
                                <el-option v-for="idx in idxNames" :key="idx" :value="idx" :label="idx" />
                            </el-select>
                            <el-button @click="onCopyMappings" link type="primary">{{ t('es.copyMappings') }}</el-button>
                            <el-button @click="onSampleMappings" link type="warning">{{ t('es.sampleMappings') }}</el-button>
                        </el-space>
                    </template>

                    <template #mappings>
                        <monaco-editor v-model="formData.mappings" language="json" :height="height - 130 + 'px'" width="100%" :options="{ tabSize: 2 }" />
                    </template>
                </auto-form>
            </template>
        </el-auto-resizer>
        <template #footer>
            <el-button @click="visible = false">{{ t('common.cancel') }}</el-button>
            <el-button type="primary" @click="confirm" :loading="loading">{{ t('common.confirm') }}</el-button>
        </template>
    </el-drawer>
</template>

<script setup lang="ts">
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import { Msg } from '@/hooks/useI18n';
import { esApi } from '@/views/ops/es/api';
import { ref, useTemplateRef, watch, type PropType } from 'vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const defaultSettings = {
    number_of_shards: 5,
    number_of_replicas: 1,
    blocks: {
        read_only: 'false',
    },
    max_result_window: '1000000',
    refresh_interval: '30s',
};

const emptyMappings = {
    mappings: {
        properties: {},
    },
    settings: defaultSettings,
};

// 点击加载mapping示例
const sampleMappings = {
    mappings: {
        properties: {
            title: {
                type: 'text',
                analyzer: 'ik_max_word',
                search_analyzer: 'ik_smart',
                fields: {
                    standard: {
                        type: 'text',
                        analyzer: 'standard',
                    },
                    keyword: {
                        type: 'keyword',
                        ignore_above: 250,
                    },
                },
            },
            mediaName: {
                type: 'text',
                fields: {
                    keyword: {
                        type: 'keyword',
                        ignore_above: 256,
                    },
                },
            },
        },
    },
    settings: defaultSettings,
};

const formData = ref({
    idxName: '',
    copyIdxName: '',
    mappings: '',
});

const props = defineProps({
    instId: { type: Number, required: true },
    idxNames: { type: Array as PropType<string[]>, default: () => [] },
});
const loading = ref(false);
const formRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('formRef');

/** 表单声明（AutoFormItem[]；复制索引操作行与 mappings 编辑器走 custom 插槽） */
const items: AutoFormItem[] = [
    { prop: 'idxName', label: 'es.indexName', required: true, props: { maxlength: 200, showWordLimit: true } },
    { prop: 'copyIdxName', type: 'custom' },
    { prop: 'mappings', label: 'mappings', type: 'custom', required: true },
];

const visible = defineModel<boolean>('visible');

watch(visible, async (x) => {
    if (x) {
        formData.value.idxName = '';
        formData.value.copyIdxName = '';
        formData.value.mappings = JSON.stringify(emptyMappings, null, 2);
        loading.value = false;
    }
});

const emit = defineEmits(['success']);

const confirm = async () => {
    await formRef.value?.validate();
    loading.value = true;
    if (!formData.value.idxName) {
        Msg.warning('es.requireIndexName');
        return;
    }
    await esApi.proxyReq('put', props.instId, `/${formData.value.idxName}`, JSON.parse(formData.value.mappings));
    Msg.saveSuccess();
    emit('success');
    loading.value = false;
    visible.value = false;
};

const onSampleMappings = () => {
    formData.value.mappings = JSON.stringify(sampleMappings, null, 2);
};
const onCopyMappings = async () => {
    let mp = await esApi.proxyReq<Record<string, { mappings: { properties?: Record<string, unknown> } }>>(
        'get',
        props.instId,
        `/${formData.value.copyIdxName}/_mappings`
    );
    let properties = mp[formData.value.copyIdxName].mappings.properties;
    formData.value.mappings = JSON.stringify(
        {
            mappings: {
                properties,
            },
            settings: defaultSettings,
        },
        null,
        2
    );
};
</script>
<style scoped lang="scss"></style>
