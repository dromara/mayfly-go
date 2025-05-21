<!-- es 编辑索引 -->

<template>
    <el-drawer
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
                <el-form :model="formData" ref="formRef">
                    <el-form-item :label="t('es.indexName')" required prop="idxName">
                        <el-input v-model.trim="formData.idxName" maxlength="200" show-word-limit />
                    </el-form-item>
                    <el-space>
                        <el-form-item>
                            <el-select v-model="formData.copyIdxName">
                                <el-option v-for="idx in idxNames" :key="idx" :value="idx" :label="idx" />
                            </el-select>
                        </el-form-item>
                        <el-form-item>
                            <el-button @click="onCopyMappings" link type="primary">{{ t('es.copyMappings') }}</el-button>
                        </el-form-item>

                        <el-form-item>
                            <el-button @click="onSampleMappings" link type="warning">{{ t('es.sampleMappings') }}</el-button>
                        </el-form-item>
                    </el-space>
                    <el-form-item required prop="mappings" label="mappings" label-position="top">
                        <monaco-editor v-model="formData.mappings" language="json" :height="height - 130 + 'px'" width="100%" :options="{ tabSize: 2 }" />
                    </el-form-item>
                </el-form>
            </template>
        </el-auto-resizer>
        <template #footer>
            <el-button @click="visible = false">{{ t('common.cancel') }}</el-button>
            <el-button type="primary" @click="confirm" :loading="loading">{{ t('common.confirm') }}</el-button>
        </template>
    </el-drawer>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { ref, watch } from 'vue';
import { esApi } from '@/views/ops/es/api';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { ElMessage } from 'element-plus';

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

interface Props {
    instId: any;
    idxNames: string[];
}
const props = defineProps<Props>();
const loading = ref(false);
const formRef = ref();

const visible = defineModel<boolean>('visible');

watch(visible, async (x: any) => {
    if (x) {
        formData.value.idxName = '';
        formData.value.copyIdxName = '';
        formData.value.mappings = JSON.stringify(emptyMappings, null, 2);
        loading.value = false;
    }
});

const emit = defineEmits(['success']);

const confirm = async () => {
    await formRef.value.validate();
    loading.value = true;
    if (!formData.value.idxName) {
        ElMessage.warning(t('es.requireIndexName'));
        return;
    }
    await esApi.proxyReq('put', props.instId, `/${formData.value.idxName}`, JSON.parse(formData.value.mappings));
    ElMessage.success(t('common.saveSuccess'));
    emit('success');
    loading.value = false;
    visible.value = false;
};

const onSampleMappings = () => {
    formData.value.mappings = JSON.stringify(sampleMappings, null, 2);
};
const onCopyMappings = async () => {
    let mp = await esApi.proxyReq('get', props.instId, `/${formData.value.copyIdxName}/_mappings`);
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
