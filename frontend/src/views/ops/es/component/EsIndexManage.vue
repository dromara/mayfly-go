<template>
    <div class="es-index-manage-box">
        <div class="es-idx-toolbar flex-shrink-0">
            <el-space>
                <el-button type="primary" icon="Plus" size="small" @click="onAddIndex">{{ t('es.addIndex') }}</el-button>
                <el-button icon="Refresh" size="small" @click="fetchIndices">{{ t('common.refresh') }}</el-button>
                <el-button type="primary" size="small" @click="templateVisible = true">{{ t('es.templates') }}</el-button>
                <el-checkbox v-model="showSysIndex" size="small" @change="fetchIndices">{{ t('es.contextmenu.index.showSys') }}</el-checkbox>
            </el-space>
        </div>
        <div class="es-idx-table">
            <el-auto-resizer>
                <template #default="{ height, width }">
                    <el-table-v2
                        :columns="tableColumns"
                        :data="filteredIndices"
                        :width="width"
                        :height="height"
                        :row-height="40"
                        v-loading="loading"
                        :sort-state="sortState"
                        @column-sort="onColumnSort"
                        fixed
                    />
                </template>
            </el-auto-resizer>
        </div>

        <!-- 查看/编辑 Mapping 对话框 -->
        <el-drawer
            v-model="mappingDrawer.visible"
            :title="`${t('es.indexMapping')} - ${mappingDrawer.idxName}`"
            size="55%"
            :append-to-body="false"
            :destroy-on-close="false"
        >
            <el-auto-resizer>
                <template #default="{ height, width }">
                    <monaco-editor
                        v-model="mappingDrawer.content"
                        language="json"
                        :height="height - 60 + 'px'"
                        :width="width + 'px'"
                        :options="{ tabSize: 2, readOnly: !mappingDrawer.editable }"
                    />
                </template>
            </el-auto-resizer>
            <template #footer>
                <el-space>
                    <el-button @click="mappingDrawer.editable = !mappingDrawer.editable">
                        {{ mappingDrawer.editable ? t('common.cancel') : t('common.edit') }}
                    </el-button>
                    <el-button v-if="mappingDrawer.editable" type="primary" @click="onSaveMapping" :loading="mappingDrawer.saving">{{
                        t('common.save')
                    }}</el-button>
                </el-space>
            </template>
        </el-drawer>

        <!-- 添加索引对话框 -->
        <EsAddIndex :instId="props.instId" :idxNames="idxNames" v-model:visible="addIndexVisible" @success="fetchIndices" />

        <!-- 索引迁移对话框 -->
        <EsReindex
            :instId="reindexState.instId"
            :idxName="reindexState.idxName"
            :idxNames="reindexState.idxNames"
            v-model:visible="reindexState.visible"
            @success="fetchIndices"
        />

        <!-- 索引详情 -->
        <EsIndexDetail ref="esIndexDetailRef" />

        <!-- 索引模板管理 -->
        <EsIndexTemplate :instId="props.instId" :version="esVersion" v-model="templateVisible" />

        <!-- 添加别名对话框 -->
        <el-dialog v-model="aliasDialog.visible" :title="t('es.addAlias')" width="400" :append-to-body="false">
            <el-form @submit.prevent="onSubmitAddAlias">
                <el-form-item :label="t('es.aliases')">
                    <el-input v-model="aliasDialog.name" autocomplete="off" />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="aliasDialog.visible = false">{{ t('common.cancel') }}</el-button>
                <el-button type="primary" @click="onSubmitAddAlias" :loading="aliasDialog.loading">{{ t('common.confirm') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, h, onMounted, reactive, ref } from 'vue';
import { ElButton, ElDropdown, ElDropdownItem, ElDropdownMenu, ElTag } from 'element-plus';
import { useI18n } from 'vue-i18n';
import SvgIcon from '@/components/svg-icon/index.vue';
import { esApi } from '@/views/ops/es/api';
import { copyToClipboard } from '@/common/utils/string';
import { Msg, useI18nConfirm, useI18nDeleteConfirm } from '@/hooks/useI18n';

const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));
const EsAddIndex = defineAsyncComponent(() => import('./EsAddIndex.vue'));
const EsReindex = defineAsyncComponent(() => import('./EsReindex.vue'));
const EsIndexDetail = defineAsyncComponent(() => import('./EsIndexDetail.vue'));
const EsIndexTemplate = defineAsyncComponent(() => import('./EsIndexTemplate.vue'));

const { t } = useI18n();

const props = defineProps<{
    instId: number;
}>();

const loading = ref(false);
const showSysIndex = ref(false);

/** 索引行数据 (_cat/indices 返回) */
interface EsIndexRow {
    index: string;
    health?: string;
    status?: string;
    [key: string]: unknown;
}

const indices = ref<EsIndexRow[]>([]);
const aliasesMap = reactive<Record<string, string[]>>({});
const sortState = ref<Record<string, string>>({ index: 'ascending' });

// tableColumns for el-table-v2
const tableColumns = computed(() => [
    {
        dataKey: 'index',
        key: 'index',
        title: t('es.indexName'),
        width: 220,
        sortable: true,
        cellRenderer: ({ rowData }: { rowData: EsIndexRow }) => h('a', {
            href: 'javascript:void(0)',
            style: { color: 'var(--el-color-primary)', textDecoration: 'none' },
            onClick: () => emit('viewData', rowData.index)
        }, rowData.index)
    },
    {
        dataKey: 'aliases',
        key: 'aliases',
        title: t('es.aliases'),
        width: 200,
        cellRenderer: ({ rowData }: { rowData: EsIndexRow }) => {
            const aliases = aliasesMap[rowData.index] || [];
            return h('div', { class: 'flex items-center gap-1 flex-wrap' },
                [...aliases.map((alias: string) => h(ElTag, {
                    closable: true,
                    size: 'small',
                    type: 'info',
                    onClose: () => onRemoveAlias(rowData.index, alias)
                }, () => alias)),
                h(ElButton, {
                    link: true,
                    type: 'primary',
                    size: 'small',
                    onClick: () => onAddAlias(rowData)
                }, () => h(SvgIcon, { name: 'Plus', size: 14 }))]
            );
        }
    },
    {
        dataKey: 'health',
        key: 'health',
        title: t('es.health'),
        width: 100,
        sortable: true,
        align: 'center',
        cellRenderer: ({ rowData }: { rowData: EsIndexRow }) => h(ElTag, { size: 'small', type: getHealthTagType(rowData.health) }, () => rowData.health)
    },
    {
        dataKey: 'status',
        key: 'status',
        title: t('es.status'),
        width: 100,
        sortable: true,
        align: 'center',
        cellRenderer: ({ rowData }: { rowData: EsIndexRow }) => h(ElTag, { size: 'small', type: rowData.status === 'open' ? 'success' : 'danger' }, () => rowData.status)
    },
    { dataKey: 'pri', key: 'pri', title: 'pri', width: 70, align: 'center' },
    { dataKey: 'rep', key: 'rep', title: 'rep', width: 70, align: 'center' },
    {
        dataKey: 'docs.count',
        key: 'docs.count',
        title: t('es.docs'),
        width: 120,
        sortable: true,
        align: 'right',
        cellRenderer: ({ rowData }: { rowData: EsIndexRow }) => (rowData['docs.count'] as string | number) ?? '-'
    },
    { dataKey: 'store.size', key: 'store.size', title: t('es.size'), width: 120, sortable: true, align: 'right' },
    {
        dataKey: 'operation',
        key: 'operation',
        title: t('common.operation'),
        width: 200,
        fixed: 'right',
        align: 'center',
        cellRenderer: ({ rowData }: { rowData: EsIndexRow }) => {
            const dropdownTrigger = h(ElButton, { link: true, type: 'primary', size: 'small' }, () => [t('common.more'), h(SvgIcon, { name: 'ArrowDown', size: 14 })]);
            const dropdownMenu = [
                h(ElDropdownItem, { key: 'copyName', command: 'copyName' }, () => t('es.contextmenu.index.copyName')),
                h(ElDropdownItem, { key: 'refresh', command: 'refresh' }, () => t('es.contextmenu.index.refresh')),
                h(ElDropdownItem, { key: 'flush', command: 'flush' }, () => t('es.contextmenu.index.flush')),
                h(ElDropdownItem, { key: 'clearCache', command: 'clearCache' }, () => t('es.contextmenu.index.clearCache')),
                h(ElDropdownItem, { key: 'reindex', command: 'reindex' }, () => t('es.Reindex')),
                rowData.status === 'open'
                    ? h(ElDropdownItem, { key: 'close', command: 'close' }, () => t('es.contextmenu.index.Close'))
                    : h(ElDropdownItem, { key: 'open', command: 'open' }, () => t('es.contextmenu.index.Open')),
                h(ElDropdownItem, { key: 'delete', command: 'delete', divided: true }, () => t('common.delete'))
            ];
            return h('div', { class: 'flex items-center justify-center gap-1' }, [
                h(ElButton, { link: true, type: 'primary', size: 'small', onClick: () => onViewDetail(rowData) }, () => t('es.indexDetail')),
                h(ElDropdown, {
                    trigger: 'click',
                    onCommand: (cmd: string) => onRowCommand(cmd, rowData)
                }, { default: () => dropdownTrigger, dropdown: () => h(ElDropdownMenu, {}, () => dropdownMenu) })
            ]);
        }
    }
]);

const addIndexVisible = ref(false);
const templateVisible = ref(false);
const esVersion = ref('');

const emit = defineEmits(['viewData']);

const esIndexDetailRef = ref();

const aliasDialog = reactive({
    visible: false,
    idxName: '',
    name: '',
    loading: false,
});

const reindexState = reactive({
    instId: 0 as number,
    idxName: '',
    visible: false,
    idxNames: [] as string[],
});

const mappingDrawer = reactive({
    visible: false,
    idxName: '',
    content: '',
    editable: false,
    saving: false,
});

const idxNames = computed(() => indices.value.map((idx) => idx.index).filter((n) => !n.startsWith('.')));

const filteredIndices = computed(() => {
    const data = [...indices.value];
    const entries = Object.entries(sortState.value);
    if (entries.length === 0) return data;
    const [key, order] = entries[0];
    const dir = order === 'ascending' ? 1 : -1;
    return data.sort((a, b) => {
        const va = a[key] ?? '';
        const vb = b[key] ?? '';
        if (typeof va === 'number' && typeof vb === 'number') return (va - vb) * dir;
        return String(va).localeCompare(String(vb)) * dir;
    });
});

onMounted(() => {
    fetchIndices();
    fetchVersion();
});

const fetchVersion = async () => {
    try {
        const res = await esApi.proxyReq<{ version?: { number?: string } }>('get', props.instId, '/');
        esVersion.value = res?.version?.number || '';
    } catch {
        // non-critical
    }
};

const fetchIndices = async () => {
    loading.value = true;
    try {
        const res = await esApi.proxyReq<EsIndexRow[]>('get', props.instId, `/_cat/indices/?h=index,health,status,uuid,pri,rep,docs.count,docs.deleted,store.size,sc,cd`);
        const list = res || [];
        indices.value = showSysIndex.value ? list : list.filter((idx) => !idx.index.startsWith('.'));
        // Fetch aliases for all indices
        await fetchAliases();
    } finally {
        loading.value = false;
    }
};

const fetchAliases = async () => {
    try {
        const res = await esApi.proxyReq<Record<string, { aliases?: Record<string, unknown> }>>('get', props.instId, '/_alias');
        // Clear and rebuild
        for (const key of Object.keys(aliasesMap)) {
            delete aliasesMap[key];
        }
        for (const idxName of Object.keys(res || {})) {
            const aliases = Object.keys(res[idxName]?.aliases || {});
            if (aliases.length > 0) {
                aliasesMap[idxName] = aliases;
            }
        }
    } catch {
        // Alias fetch failure is non-critical
    }
};

const onColumnSort = ({ key, order }: { key: string; order: string }) => {
    sortState.value = { [key]: order };
};

const onAddIndex = () => {
    addIndexVisible.value = true;
};

const getHealthTagType = (health?: string) => {
    return health == 'green' ? 'success' : health == 'yellow' ? 'warning' : 'danger';
};

// ---- Index operations ----

const onRowCommand = async (cmd: string, row: EsIndexRow) => {
    switch (cmd) {
        case 'copyName':
            await copyToClipboard(row.index);
            break;
        case 'refresh':
            await esApi.proxyReq('post', props.instId, `/${row.index}/_refresh`);
            Msg.operateSuccess();
            break;
        case 'mapping':
            await onViewMapping(row);
            break;
        case 'flush':
            await onFlushIndex(row);
            break;
        case 'clearCache':
            await onClearCache(row);
            break;
        case 'reindex':
            onReindex(row);
            break;
        case 'close':
            await onCloseIndex(row);
            break;
        case 'open':
            await onOpenIndex(row);
            break;
        case 'delete':
            await onDeleteIndex(row);
            break;
    }
};

const onViewDetail = (row: EsIndexRow) => {
    esIndexDetailRef.value?.open({ idxName: row.index, instId: props.instId });
};

const onReindex = async (row: EsIndexRow) => {
    reindexState.instId = props.instId;
    reindexState.idxName = row.index;
    reindexState.idxNames = idxNames.value.filter((n) => n !== row.index);
    reindexState.visible = true;
};

const onViewMapping = async (row: EsIndexRow) => {
    const res = await esApi.proxyReq<Record<string, { mappings?: Record<string, unknown> }>>('get', props.instId, `/${row.index}/_mappings`);
    mappingDrawer.idxName = row.index;
    mappingDrawer.content = JSON.stringify(res[row.index]?.mappings || {}, null, 2);
    mappingDrawer.editable = false;
    mappingDrawer.saving = false;
    mappingDrawer.visible = true;
};

const onSaveMapping = async () => {
    mappingDrawer.saving = true;
    try {
        await esApi.proxyReq('put', props.instId, `/${mappingDrawer.idxName}/_mappings`, JSON.parse(mappingDrawer.content));
        Msg.saveSuccess();
        mappingDrawer.editable = false;
    } finally {
        mappingDrawer.saving = false;
    }
};

const onCloseIndex = async (row: EsIndexRow) => {
    await useI18nConfirm('es.closeIndexConfirm', { name: row.index });
    await esApi.proxyReq('post', props.instId, `/${row.index}/_close`);
    row.status = 'close';
    Msg.operateSuccess();
};

const onOpenIndex = async (row: EsIndexRow) => {
    await useI18nConfirm('es.openIndexConfirm', { name: row.index });
    await esApi.proxyReq('post', props.instId, `/${row.index}/_open`);
    row.status = 'open';
    Msg.operateSuccess();
};

const onFlushIndex = async (row: EsIndexRow) => {
    await esApi.proxyReq('post', props.instId, `/${row.index}/_flush`);
    Msg.operateSuccess();
};

const onClearCache = async (row: EsIndexRow) => {
    await useI18nConfirm('es.clearCacheConfirm', { name: row.index });
    await esApi.proxyReq('post', props.instId, `/${row.index}/_cache/clear`);
    Msg.operateSuccess();
};

const onDeleteIndex = async (row: EsIndexRow) => {
    await useI18nDeleteConfirm(row.index);
    await esApi.proxyReq('delete', props.instId, row.index);
    Msg.deleteSuccess();
    await fetchIndices();
};

// ---- Alias operations ----

const onAddAlias = (row: EsIndexRow) => {
    aliasDialog.idxName = row.index;
    aliasDialog.name = '';
    aliasDialog.loading = false;
    aliasDialog.visible = true;
};

const onSubmitAddAlias = async () => {
    if (!aliasDialog.name) return;
    aliasDialog.loading = true;
    try {
        await esApi.proxyReq('put', props.instId, `/${aliasDialog.idxName}/_alias/${aliasDialog.name}`);
        Msg.saveSuccess();
        // Update local aliases
        if (!aliasesMap[aliasDialog.idxName]) {
            aliasesMap[aliasDialog.idxName] = [];
        }
        aliasesMap[aliasDialog.idxName].push(aliasDialog.name);
        aliasDialog.visible = false;
    } finally {
        aliasDialog.loading = false;
    }
};

const onRemoveAlias = async (idxName: string, alias: string) => {
    await useI18nDeleteConfirm(`${t('es.aliases')}: ${alias}`);
    await esApi.proxyReq('delete', props.instId, `/${idxName}/_alias/${alias}`);
    Msg.deleteSuccess();
    // Update local aliases
    if (aliasesMap[idxName]) {
        aliasesMap[idxName] = aliasesMap[idxName].filter((a: string) => a !== alias);
    }
};
</script>

<style scoped lang="scss">
.es-index-manage-box {
    height: 100%;
    display: flex;
    flex-direction: column;
}

.es-idx-toolbar {
    padding: 6px 8px;
    border-bottom: 1px solid var(--el-border-color-light);
}

.es-idx-table {
    flex: 1;
    min-height: 0;
}
</style>
