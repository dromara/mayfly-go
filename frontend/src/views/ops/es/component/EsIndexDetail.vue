<template>
    <el-drawer
        :title="t('es.indexDetail') + ' - ' + state.idxName"
        v-model="visible"
        size="50%"
        :destroy-on-close="false"
        :close-on-click-modal="false"
        :close-on-press-escape="false"
        class="es-index-detail h-full"
        @close="onClose"
    >
        <el-auto-resizer>
            <template #default="{ height, width }">
                <el-tabs v-model="activeName">
                    <el-tab-pane name="settings">
                        <template #label>
                            <el-tooltip>
                                <template #content> {{ t('es.availableSettingFields') }}: {{ allowedKeys }}</template>
                                <el-space>{{ t('es.opSettings') }}<SvgIcon name="QuestionFilled" /></el-space>
                            </el-tooltip>
                        </template>
                        <monaco-editor v-model="state.settings" language="json" :height="height - 40 + 'px'" :options="{ tabSize: 2 }" />
                    </el-tab-pane>
                    <el-tab-pane :label="t('es.indexMapping')" name="mappings">
                        <monaco-editor v-model="state.mappings" language="json" :height="height - 40 + 'px'" :options="state.editorOptions" />
                    </el-tab-pane>
                    <el-tab-pane :label="t('es.indexStats')" name="stats">
                        <monaco-editor v-model="state.stats" language="json" :height="height - 40 + 'px'" :options="state.editorOptions" />
                    </el-tab-pane>
                    <el-tab-pane :label="t('es.aliases')" name="aliases">
                        <el-button type="primary" @click="onAddAlias" icon="plus" size="small">{{ t('es.addAlias') }}</el-button>

                        <div :style="{ paddingTop: '20px' }">
                            <el-space direction="vertical" alignment="start">
                                <el-tag v-for="tag in state.aliases" :key="tag" closable type="primary" @close="onRemoveAlias(tag)">
                                    {{ tag }}
                                </el-tag>
                            </el-space>
                        </div>
                    </el-tab-pane>
                </el-tabs>
            </template>
        </el-auto-resizer>

        <template #footer>
            <el-button size="small" @click="visible = false">{{ t('common.close') }}</el-button>
            <el-button size="small" @click="onOk" type="primary" v-if="activeName == 'settings'" :loading="state.loading">{{ t('common.confirm') }}</el-button>
        </template>
    </el-drawer>

    <el-dialog v-model="dialogFormVisible" :title="t('es.addAlias')" width="400">
        <el-form :model="state.aliasesForm">
            <el-form-item :label="t('es.aliases')">
                <el-input v-model="state.aliasesForm.name" autocomplete="off" />
            </el-form-item>
        </el-form>
        <template #footer>
            <el-button size="small" @click="dialogFormVisible = false">{{ t('common.cancel') }}</el-button>
            <el-button size="small" @click="onSubmitAddAlias" :loading="aliasLoading" type="primary">{{ t('common.confirm') }}</el-button>
        </template>
    </el-dialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { defineAsyncComponent, reactive, ref, watch } from 'vue';
import { esApi } from '@/views/ops/es/api';
import { ElMessage } from 'element-plus';
import { useI18nDeleteConfirm } from '@/hooks/useI18n';
const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

const { t } = useI18n();

const visible = ref(false);
const aliasLoading = ref(false);
const activeName = ref('settings');

const defaultData = {
    instId: 0,
    loading: false,
    aliases: [] as string[],
    idxName: '',
    health: {},
    mappings: '',
    stats: '',
    settings: '',
    editorOptions: { tabSize: 2, readOnly: true, readOnlyMessage: { value: t('es.readonlyMsg') } },
    aliasesForm: { name: '' },
};

let state = reactive(defaultData);

const dialogFormVisible = ref(false);

let allowedKeys = ['number_of_replicas', 'refresh_interval', 'blocks.read_only', 'blocks.read', 'blocks.write', 'max_result_window', 'blocks'];

const onOk = async () => {
    if (activeName.value === 'settings') {
        /**
         * 常见可修改设置：
         * 设置项                         描述
         * number_of_replicas            副本分片数，可随时修改
         * refresh_interval              刷新间隔，控制索引频率
         * blocks.read_only              设置索引为只读或可写
         * blocks.read / blocks.write    控制是否允许读/写操作
         * max_result_window             控制最大返回结果数量，默认为10000
         */
        let settings = JSON.parse(state.settings).index;
        // 只允许传可设置的字段
        for (let key in settings) {
            if (allowedKeys.indexOf(key) == -1) {
                delete settings[key];
            }
        }

        await esApi.proxyReq('put', state.instId, `/${state.idxName}/_settings`, { index: settings });
        ElMessage.success(t('common.saveSuccess'));
    }
};

watch(activeName, async (val) => {
    state.mappings = '';
    state.stats = '';
    state.aliases = [];
    // 如果没有值就请求接口获取值
    if (val === 'mappings') {
        await refreshMappings();
    } else if (val === 'stats') {
        await refreshStats();
    } else if (val === 'aliases') {
        await refreshAlias();
    }
});

const refreshMappings = async () => {
    let res = await esApi.proxyReq('get', state.instId, `/${state.idxName}/_mappings`);
    state.mappings = JSON.stringify(res[state.idxName].mappings, null, 2);
};

const refreshStats = async () => {
    let stats = await esApi.proxyReq('get', state.instId, `/${state.idxName}/_stats`);
    state.stats = JSON.stringify(stats.indices[state.idxName], null, 2);
};

const refreshAlias = async () => {
    let aliases = await esApi.proxyReq('get', state.instId, `/${state.idxName}/_alias`);
    state.aliases = Object.keys(aliases[state.idxName].aliases);
};

const refreshSettings = async () => {
    let res = await esApi.proxyReq('get', state.instId, `/${state.idxName}/_settings`);
    let st = res[state.idxName].settings;

    state.settings = JSON.stringify(st, null, 2);
};

const initBasic = async () => {
    state.health = await esApi.proxyReq('get', state.instId, `/_cluster/health/${state.idxName}`);
    await refreshSettings();
};

const onAddAlias = async () => {
    dialogFormVisible.value = true;
    state.aliasesForm.name = '';
    aliasLoading.value = false;
};
const onRemoveAlias = async (name: string) => {
    await useI18nDeleteConfirm(`${t('es.aliases')}: ${name}`);

    await esApi.proxyReq('delete', state.instId, `/${state.idxName}/_alias/${name}`);
    ElMessage.success(t('common.deleteSuccess'));
    await refreshAlias();
};

const onSubmitAddAlias = async () => {
    aliasLoading.value = true;
    await esApi.proxyReq('put', state.instId, `/${state.idxName}/_alias/${state.aliasesForm.name}`);
    ElMessage.success(t('common.saveSuccess'));
    await refreshAlias();
    dialogFormVisible.value = false;
    aliasLoading.value = false;
};

const onClose = () => {
    state = reactive(defaultData);
};

const open = (data: any) => {
    visible.value = true;
    activeName.value = 'settings';
    state = reactive(defaultData);
    state.instId = data.instId;
    state.idxName = data.idxName;
    initBasic();
};
const close = () => {
    visible.value = false;
    onClose();
};

defineExpose({
    open,
    close,
});
</script>

<style lang="scss">
.es-index-detail {
    .el-drawer__body {
        padding-top: 0;
    }
}
</style>
