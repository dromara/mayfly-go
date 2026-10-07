<template>
    <el-drawer v-model="visible" size="58%" :close-on-click-modal="false" class="coll-meta">
        <template #header>
            <div class="flex min-w-0 items-center gap-2">
                <span class="truncate font-mono text-14px font-semibold">{{ ns }}</span>
                <span class="text-[13px] text-gray-500">{{ $t('mongo.collMeta') }}</span>
                <div class="ml-auto flex items-center gap-2">
                    <el-button v-auth="perms.ddlSave" type="primary" size="small" icon="plus" :disabled="!target" @click="openCreate">
                        {{ $t('mongo.createIndex') }}
                    </el-button>
                    <el-tooltip :content="$t('common.refresh')" placement="top">
                        <el-button size="small" icon="refresh" :loading="state.loading" :disabled="!target" @click="load" />
                    </el-tooltip>
                </div>
            </div>
        </template>

        <div v-if="!target" class="text-[13px] text-gray-400">{{ $t('mongo.selectCollectionTip') }}</div>
        <div v-else class="flex flex-col gap-4">
            <el-alert v-if="state.error" :title="state.error" type="error" show-icon :closable="false" />

            <!-- 统计：取不到时说明原因而不是摆一排 0，0 会被读成「集合真的是空的」 -->
            <div>
                <div class="section-title">{{ $t('mongo.collState') }}</div>
                <el-alert v-if="!stats && state.meta?.statsError" :title="state.meta.statsError" type="info" :closable="false" show-icon />
                <el-descriptions v-else-if="stats" :column="3" size="small" border>
                    <el-descriptions-item :label="$t('mongo.docCount')">{{ stats.count }}</el-descriptions-item>
                    <el-descriptions-item :label="$t('mongo.avgObjSize')">{{ formatByteSize(stats.avgObjSize) }}</el-descriptions-item>
                    <el-descriptions-item :label="$t('mongo.storageSize')">{{ formatByteSize(stats.storageSize) }}</el-descriptions-item>
                    <el-descriptions-item :label="$t('mongo.indexSize')">{{ formatByteSize(stats.totalIndexSize) }}</el-descriptions-item>
                    <el-descriptions-item :label="$t('mongo.totalSize')">{{ formatByteSize(stats.totalSize) }}</el-descriptions-item>
                    <el-descriptions-item :label="$t('mongo.indexCount')">{{ stats.nindexes }}</el-descriptions-item>
                    <el-descriptions-item v-if="stats.freeStorageSize > 0" :label="$t('mongo.freeStorageSize')">
                        {{ formatByteSize(stats.freeStorageSize) }}
                    </el-descriptions-item>
                </el-descriptions>
            </div>

            <!-- 索引：键的顺序即索引优先级，方向必须是可读的 ↑/↓/命名类型 -->
            <div>
                <div class="section-title">{{ $t('mongo.indexList') }}</div>
                <el-table v-loading="state.loading" :data="indexes" size="small" max-height="420">
                    <el-table-column min-width="170" :label="$t('common.name')" prop="name" show-overflow-tooltip />
                    <el-table-column min-width="200" :label="$t('mongo.indexKeys')">
                        <template #default="{ row }">
                            <span class="font-mono">{{ keysText(row.keys) }}</span>
                        </template>
                    </el-table-column>
                    <el-table-column min-width="150" :label="$t('mongo.indexOptions')">
                        <template #default="{ row }">
                            <el-tag v-if="row.unique" size="small" type="warning">unique</el-tag>
                            <el-tooltip v-for="option in extraOptions(row.spec)" :key="option.key" :content="`${option.key}: ${option.value}`" placement="top">
                                <el-tag size="small" class="ml-1">{{ option.key }}</el-tag>
                            </el-tooltip>
                        </template>
                    </el-table-column>
                    <el-table-column min-width="90" align="right" :label="$t('mongo.indexSize')">
                        <template #default="{ row }">{{ row.sizeBytes ? formatByteSize(row.sizeBytes) : '-' }}</template>
                    </el-table-column>
                    <el-table-column min-width="180" fixed="right" :label="$t('common.operation')">
                        <template #default="{ row }">
                            <el-button link type="primary" @click="onCopySpec(row)">{{ $t('common.copy') }}</el-button>
                            <el-button v-auth="perms.ddlSave" link type="primary" @click="onReuseSpec(row)">
                                {{ $t('mongo.reuseAsTemplate') }}
                            </el-button>
                            <el-tooltip v-if="row.name === ID_INDEX_NAME" :content="$t('mongo.idIndexUndroppable')" placement="top">
                                <span class="ml-2 text-[13px] text-gray-400">{{ $t('common.delete') }}</span>
                            </el-tooltip>
                            <el-button v-else v-auth="perms.ddlDel" link type="danger" :loading="state.writing" @click="onDropIndex(row)">
                                {{ $t('common.delete') }}
                            </el-button>
                        </template>
                    </el-table-column>
                    <template #empty>
                        <span class="text-[13px] text-gray-400">{{ $t('mongo.noIndex') }}</span>
                    </template>
                </el-table>
            </div>
        </div>

        <!-- 索引定义直接交给 Mongo：选项太多（ttl/partial/sparse/collation/wildcard），逐项做表单必然漏 -->
        <el-dialog v-model="createVisible" width="620px" :title="$t('mongo.createIndex')" :close-on-click-modal="false" append-to-body>
            <div class="flex flex-col gap-2">
                <span class="text-[13px] text-gray-500">{{ $t('mongo.indexSpecTip') }}</span>
                <monaco-editor v-model="specText" language="json" height="320px" />
            </div>
            <template #footer>
                <el-button @click="createVisible = false">{{ $t('common.cancel') }}</el-button>
                <el-button v-auth="perms.ddlSave" type="primary" :loading="state.writing" @click="onCreateIndexes">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>
    </el-drawer>
</template>

<script lang="ts" setup>
import { computed, defineAsyncComponent, ref, watch } from 'vue';

import { formatByteSize } from '@/common/utils/format';
import { copyToClipboard } from '@/common/utils/string';
import { Msg } from '@/hooks/useI18n';
import { confirmByName } from '../confirm';
import { perms } from '../perms';
import { indexSpecTemplate } from '../docview/json';
import { useCollectionMeta } from '../resource/composables/useCollectionMeta';
import type { CollectionParam, MongoIndexInfo, MongoIndexKey } from '../types';

const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

const props = defineProps<{ target: CollectionParam | null }>();

const visible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits<{
    /** 索引发生增删：父级据此刷新头部读数（nindexes 会变） */
    changed: [];
}>();

const { state, load, createIndexes, dropIndex } = useCollectionMeta();

const createVisible = ref(false);
const specText = ref(indexSpecTemplate());

/** 主键索引由 Mongo 维护，删掉它等于让集合失去唯一定位能力，服务端也会拒绝 */
const ID_INDEX_NAME = '_id_';

/**
 * 这些键不在选项列重复展示：
 * - v/key/name 是索引定义的内部字段与身份，不是「选项」；
 * - unique 已有专门的可读标签，再出一个同名标签会让同一事实被读成两件事。
 */
const SPEC_BASE_KEYS = ['v', 'key', 'name', 'background', 'unique'];

const ns = computed(() => (props.target ? `${props.target.database}.${props.target.collection}` : ''));

const stats = computed(() => state.meta?.stats);

const indexes = computed<MongoIndexInfo[]>(() => state.meta?.indexes ?? []);

watch(visible, async (open) => {
    if (open && props.target) {
        await load(props.target);
    }
});

// 面板开着时切了集合，读数必须跟着换：留着上一个集合的统计与索引会让人按错集合删索引
watch(
    () => props.target,
    async (target) => {
        if (visible.value && target) {
            await load(target);
        }
    }
);

function keysText(keys: MongoIndexKey[]): string {
    return (keys ?? [])
        .map((key) => {
            if (key.direction === 'asc') return `${key.field} ↑`;
            if (key.direction === 'desc') return `${key.field} ↓`;
            // 2dsphere / text / hashed 这类命名方向原样显示，翻译成箭头会丢掉索引类型
            return `${key.field}: ${key.direction}`;
        })
        .join(', ');
}

/** 列表上只放选项名，值进 tooltip：一列里塞满 `partialFilterExpression: {...}` 没人读得完 */
function extraOptions(spec: Record<string, unknown>) {
    return Object.entries(spec ?? {})
        .filter(([key]) => !SPEC_BASE_KEYS.includes(key))
        .map(([key, value]) => ({ key, value: typeof value === 'object' ? JSON.stringify(value) : String(value) }));
}

function openCreate() {
    specText.value = indexSpecTemplate();
    createVisible.value = true;
}

/** 以现有定义为基础：改 TTL、加复合键时不必从空白重写 */
function onReuseSpec(row: MongoIndexInfo) {
    specText.value = JSON.stringify([row.spec], null, 4);
    createVisible.value = true;
}

async function onCopySpec(row: MongoIndexInfo) {
    await copyToClipboard(JSON.stringify(row.spec, null, 4));
}

async function onCreateIndexes() {
    if (!props.target) {
        return;
    }
    if (await createIndexes(props.target, specText.value)) {
        Msg.success('mongo.indexCreated');
        createVisible.value = false;
        emit('changed');
    }
}

async function onDropIndex(row: MongoIndexInfo) {
    if (!props.target) {
        return;
    }
    if (!(await confirmByName(row.name))) {
        return;
    }
    if (await dropIndex(props.target, row.name, row.name)) {
        Msg.deleteSuccess();
        emit('changed');
    }
}
</script>

<style lang="scss" scoped>
.coll-meta {
    .section-title {
        margin-bottom: 8px;
        font-size: 13px;
        font-weight: 600;
        color: var(--el-text-color-primary);
    }
}
</style>
