<template>
    <el-drawer
        :append-to-body="false"
        :title="t('es.Reindex')"
        v-model="visible"
        size="40%"
        :destroy-on-close="false"
        :close-on-click-modal="false"
        :close-on-press-escape="false"
        class="es-reindex h-full"
    >
        <el-tabs v-model="tabActiveName">
            <el-tab-pane name="basic" label="basic">
                <auto-form ref="formRef" v-model="formData" :items="items" />
            </el-tab-pane>
            <el-tab-pane name="otherInst" :label="t('es.ReindexToOtherInst')"> developing... </el-tab-pane>
            <el-tab-pane name="task" :label="t('es.ReindexSyncTask')"> developing... </el-tab-pane>
        </el-tabs>

        <template #footer>
            <el-button @click="visible = false">{{ t('common.cancel') }}</el-button>
            <el-button type="primary" @click="confirm">{{ t('common.confirm') }}</el-button>
        </template>
    </el-drawer>
</template>

<script setup lang="ts">
import { Msg } from '@/hooks/useI18n';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import { esApi } from '@/views/ops/es/api';
import { computed, ref, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
const visible = defineModel<boolean>('visible');

const formRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('formRef');

interface Props {
    instId: number;
    idxName: string;
    idxNames: string[];
}
const props = defineProps<Props>();
const tabActiveName = ref('basic');

const formData = ref({
    targetIdxName: '',
    sync: false,
});

/** 表单声明（AutoFormItem[]；同步说明文案由 tooltip + description 承载） */
const items = computed<AutoFormItem[]>(() => [
    {
        prop: 'targetIdxName',
        label: 'es.ReindexTargetIdx',
        type: 'select',
        required: true,
        options: props.idxNames.map((idx) => ({ label: idx, value: idx })),
    },
    { prop: 'sync', label: 'es.ReindexIsSync', type: 'switch', tooltip: 'es.ReindexSyncDescription', description: 'es.ReindexDescription' },
]);

const confirm = async () => {
    if (tabActiveName.value === 'basic') {
        await doBasicReindex();
    }
};

const doBasicReindex = async () => {
    await formRef.value?.validate();
    let wfc = '';
    if (!formData.value.sync) {
        wfc = '?wait_for_completion=false';
    }
    let data = { source: { index: props.idxName }, dest: { index: formData.value.targetIdxName } };

    let res = await esApi.proxyReq('POST', props.instId, `/_reindex${wfc}`, data);
    // FIXME 如果是异步，返回异步任务id，添加到任务列表中，可以在任务列表中查看状态
    Msg.operateSuccess();
};
</script>

<style scoped lang="scss"></style>
