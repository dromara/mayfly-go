<template>
    <el-drawer
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
                <el-form :model="formData" ref="formRef">
                    <el-form-item :label="t('es.ReindexTargetIdx')" required prop="targetIdxName">
                        <el-select clearable filterable v-model="formData.targetIdxName" :style="{ width: '100%' }">
                            <el-option v-for="idx in idxNames" :key="idx" :value="idx" :label="idx" />
                        </el-select>
                    </el-form-item>
                    <el-form-item :label="t('es.ReindexIsSync')">
                        <el-space>
                            <el-switch v-model="formData.sync" />
                            <el-text type="info" size="small">{{ t('es.ReindexSyncDescription') }}</el-text>
                        </el-space>
                    </el-form-item>
                    <el-form-item>
                        <el-text type="info" size="small">{{ t('es.ReindexDescription') }}</el-text>
                    </el-form-item>
                </el-form>
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
import { useI18n } from 'vue-i18n';
import { ref } from 'vue';
import { esApi } from '@/views/ops/es/api';
import { ElMessage } from 'element-plus';

const { t } = useI18n();
const visible = defineModel<boolean>('visible');

const formRef = ref();

interface Props {
    instId: any;
    idxName: string;
    idxNames: string[];
}
const props = defineProps<Props>();
const tabActiveName = ref('basic');

const formData = ref({
    targetIdxName: '',
    sync: false,
});

const confirm = async () => {
    if (tabActiveName.value === 'basic') {
        await doBasicReindex();
    }
};

const doBasicReindex = async () => {
    await formRef.value.validate();
    let wfc = '';
    if (!formData.value.sync) {
        wfc = '?wait_for_completion=false';
    }
    let data = { source: { index: props.idxName }, dest: { index: formData.value.targetIdxName } };

    let res = await esApi.proxyReq('POST', props.instId, `/_reindex${wfc}`, data);
    // FIXME 如果是异步，返回异步任务id，添加到任务列表中，可以在任务列表中查看状态
    ElMessage.success(t('common.operateSuccess'));
};
</script>

<style scoped lang="scss"></style>
