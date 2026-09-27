<template>
    <div class="flex h-full min-h-0 flex-col gap-2">
        <!-- 工具条（仅表格布局）：左侧写入入口，中间关键字过滤，右侧批量选择 + 视角操作。
             窄面板下允许换行而不是裁切按钮（详情面板可能被拖到 400px 左右） -->
        <div v-if="!isValueLayout" class="flex flex-wrap items-center gap-2">
            <!-- 选择态只禁用主按钮，不替换它：按钮换位会让整行宽度重排，看着像界面在抖 -->
            <el-button v-if="caps.create" v-auth="PERM_DATA_SAVE" class="shrink-0" type="primary" icon="plus" :disabled="batchMode" @click="onCreate">
                {{ $t('redis.addMember') }}
            </el-button>

            <el-input
                v-if="caps.keyword"
                v-model="filterValue"
                class="keyword-input min-w-[110px] flex-1"
                clearable
                :placeholder="$t('redis.filterPlaceholder')"
                @clear="onSearch"
                @keyup.enter="onSearch"
            >
                <template #prefix>
                    <SvgIcon name="Search" :size="15" />
                </template>
            </el-input>

            <!-- 成员总数由详情头的统计药丸承担，这里不再重复一份读数，给关键字过滤留出宽度 -->
            <div class="ml-auto flex shrink-0 items-center gap-2">
                <el-button
                    v-if="caps.batchDelete"
                    v-auth="PERM_DATA_DEL"
                    class="shrink-0"
                    size="small"
                    :type="batchMode ? 'primary' : 'default'"
                    @click="onToggleBatch"
                >
                    {{ $t('redis.batchSelect') }}
                </el-button>
                <el-dropdown v-if="visibleOps.length" trigger="click" @command="onOp">
                    <!-- 触发器必须是一个普通元素：再套一层 el-tooltip 会让 ElOnlyChild 合入的
                         菜单开关属性落在 tooltip 的 fragment 上而丢弃，菜单永远点不开；
                         入口同时改成带文字的下拉按钮，窄面板下纯图标看不出能干什么 -->
                    <el-button class="shrink-0" size="small">
                        {{ $t('redis.viewOps') }}
                        <SvgIcon name="ArrowDown" :size="11" class="ml-1 opacity-70" />
                    </el-button>
                    <template #dropdown>
                        <el-dropdown-menu>
                            <el-dropdown-item v-for="item in visibleOps" :key="item.name" :command="item.name">
                                <span class="flex items-center gap-1.5">
                                    <SvgIcon :name="opIcon(descriptor?.view ?? '', item.name)" :size="14" />
                                    {{ $t(item.label) }}
                                </span>
                            </el-dropdown-item>
                        </el-dropdown-menu>
                    </template>
                </el-dropdown>
            </div>
        </div>

        <!-- 单值布局：整个 key 就是一个值（string、HyperLogLog 等） -->
        <div v-if="isValueLayout" class="value-pane flex min-h-0 flex-1 flex-col">
            <FormatViewer ref="viewerRef" class="min-h-0 flex-1" :content="valueContent" :readonly="!isValueEditable" @change="dirty = $event" />
            <div class="value-footer flex items-center gap-3">
                <span v-if="!isValueEditable" class="text-xs text-muted-foreground">{{ $t('redis.valueNotEditable') }}</span>
                <AutoForm
                    v-if="isValueEditable && sideFields.length"
                    v-model="sideForm"
                    :schema="sideFields"
                    class="value-side-form"
                    label-width="auto"
                    :cols="sideFields.length"
                />
                <span v-if="isValueEditable && dirty" class="dirty-flag shrink-0">
                    <i class="dirty-dot"></i>
                    {{ $t('redis.unsavedChanges') }}
                </span>
                <el-button
                    v-if="isValueEditable"
                    v-auth="PERM_DATA_SAVE"
                    class="ml-auto shrink-0"
                    type="primary"
                    icon="check"
                    :loading="saving"
                    :disabled="!dirty"
                    @click="onSaveValue"
                >
                    {{ $t('common.save') }}
                </el-button>
                <el-button v-else v-auth="PERM_DATA_SAVE" class="ml-auto shrink-0" type="primary" icon="plus" @click="onCreate">
                    {{ $t('redis.addMember') }}
                </el-button>
            </div>
        </div>

        <!-- 新增态：key 还不存在，先写入首个成员 -->
        <div v-else-if="store.creating.value" class="card flex flex-1 items-center justify-center">
            <el-empty :description="$t('redis.keyNotCreated')">
                <el-button v-auth="PERM_DATA_SAVE" type="primary" icon="plus" @click="onCreate">{{ $t('redis.writeFirstMember') }}</el-button>
            </el-empty>
        </div>

        <template v-else>
            <el-table
                ref="memberTableRef"
                v-loading="store.loading.value"
                :data="store.members.value"
                class="member-table flex-1 min-h-0"
                height="100%"
                @selection-change="onSelectionChange"
                @row-click="onRowClick"
            >
                <el-table-column v-if="batchMode && caps.batchDelete" type="selection" width="45" />
                <el-table-column
                    v-for="column in columns"
                    :key="column.field"
                    :prop="column.field"
                    :label="$t(column.label)"
                    :min-width="column.width || 150"
                    :sortable="column.sortable && !store.hasMore.value"
                    show-overflow-tooltip
                    resizable
                >
                    <template #default="{ row }">
                        <el-tag v-if="column.value === 'tag'" size="small" effect="plain">{{ cellText(row, column) }}</el-tag>
                        <span v-else-if="column.value === 'code'" class="font-mono text-xs">{{ cellText(row, column) }}</span>
                        <span v-else>{{ cellText(row, column) }}</span>
                    </template>
                </el-table-column>

                <!-- 行操作：复制值任何视角都可用，编辑/删除按描述符能力开关显示 -->
                <el-table-column v-if="columns.length" :label="$t('common.operation')" width="128" fixed="right" align="center">
                    <template #default="{ row }">
                        <div class="row-actions">
                            <el-tooltip :content="$t('redis.copyValue')" placement="top">
                                <el-button class="row-btn" text size="small" :aria-label="$t('redis.copyValue')" @click="onCopyMember(row)">
                                    <SvgIcon name="DocumentCopy" :size="15" />
                                </el-button>
                            </el-tooltip>
                            <el-tooltip v-if="caps.update" :content="$t('redis.editMember')" placement="top">
                                <el-button v-auth="PERM_DATA_SAVE" class="row-btn" text size="small" :aria-label="$t('redis.editMember')" @click="onEdit(row)">
                                    <SvgIcon name="edit" :size="15" />
                                </el-button>
                            </el-tooltip>
                            <el-popconfirm v-if="caps.delete" :title="$t('redis.deleteConfirm')" @confirm="onDelete(row)">
                                <template #reference>
                                    <el-button v-auth="PERM_DATA_DEL" class="row-btn row-btn--danger" text size="small" :aria-label="$t('common.delete')">
                                        <SvgIcon name="delete" :size="15" />
                                    </el-button>
                                </template>
                            </el-popconfirm>
                        </div>
                    </template>
                </el-table-column>

                <template #empty>
                    <el-empty :description="$t('redis.noMembers')" :image-size="80" />
                </template>
            </el-table>

            <!-- 底部动作条：常态是「加载更多」，选择态原位换成删除，工具条与表格都不重排 -->
            <div v-if="batchMode || store.hasMore.value" class="list-bar">
                <el-button v-if="!batchMode" class="load-more" text :loading="store.loading.value" @click="store.loadMembers()">
                    <SvgIcon name="ArrowDown" :size="14" />
                    <span class="ml-1">{{ $t('redis.loadMore') }}</span>
                </el-button>
                <template v-else>
                    <span class="text-xs text-muted-foreground">{{ $t('redis.selectedMembers', { count: selection.length }) }}</span>
                    <el-button
                        v-auth="PERM_DATA_DEL"
                        class="ml-auto shrink-0"
                        type="danger"
                        size="small"
                        icon="delete"
                        :disabled="!selection.length"
                        @click="onBatchDelete"
                    >
                        {{ $t('redis.batchDeleteSelected', { count: selection.length }) }}
                    </el-button>
                </template>
            </div>
        </template>

        <AutoFormDialog
            v-model:visible="dialog.visible"
            :title="dialog.title"
            :schema="dialog.schema"
            :data="dialog.data"
            :confirm-api="memberConfirm"
            width="620px"
            @confirm="submit"
        />

        <el-dialog v-model="opResult.visible" :title="opResult.title" width="620px" :close-on-click-modal="false">
            <pre class="op-result max-h-[60vh] overflow-auto rounded-md bg-[var(--el-fill-color-light)] p-3 text-xs">{{ opResult.content }}</pre>
            <template #footer>
                <el-button @click="opResult.visible = false">{{ $t('common.close') }}</el-button>
                <el-button type="primary" icon="CopyDocument" @click="copyResult">{{ $t('common.copy') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { AutoForm, AutoFormDialog, type JsonField } from '@/components/auto-form';
import { copyToClipboard } from '@/common/utils/string';
import { hasPerm } from '@/components/auth/auth';
import { Msg } from '@/hooks/useI18n';
import { PERM_DATA_DEL, PERM_DATA_SAVE } from './permission';
import { computed, ref, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import type { RedisKeyMember, RedisViewColumn } from '../types';
import FormatViewer from '../FormatViewer.vue';
import { columnValue, LAYOUT_VALUE, prefillForm, toArgs, ttlCellText } from './descriptor';
import { opIcon } from './appearance';
import { useKeyFormDialog } from './useKeyFormDialog';
import type { KeyViewStore } from './useKeyView';

const props = defineProps<{
    store: KeyViewStore;
    /** 新增 key 时随首个成员一起落库的过期时间（秒） */
    ttl?: number;
}>();

const { t } = useI18n();

const filterValue = ref('');
const selection = ref<RedisKeyMember[]>([]);
/** 批量删除态：默认关闭，避免平时表格多一列勾选框占位与误点 */
const batchMode = ref(false);
/** 单值编辑器是否有未保存修改 */
const dirty = ref(false);

const memberTableRef = useTemplateRef<{ toggleRowSelection: (row: RedisKeyMember, selected?: boolean) => void }>('memberTableRef');

/** 选择态开关：再点一次即退出并清空勾选，因此不需要「取消」按钮 */
const onToggleBatch = () => {
    if (batchMode.value) {
        selection.value = [];
        batchMode.value = false;
        return;
    }
    batchMode.value = true;
};

/** 选择态下点整行即勾选，不用去瞄那 16px 的复选框；勾选框自身的点击交给表格原生处理 */
const onRowClick = (row: RedisKeyMember, column?: { type?: string }) => {
    if (!batchMode.value || column?.type === 'selection') {
        return;
    }
    memberTableRef.value?.toggleRowSelection(row);
};
const saving = ref(false);
const sideForm = ref<Record<string, unknown>>({});
const viewerRef = useTemplateRef<{ getContent: () => string }>('viewerRef');

const descriptor = computed(() => props.store.descriptor.value);
const caps = computed(() => props.store.caps.value);
const columns = computed<RedisViewColumn[]>(() => descriptor.value?.columns ?? []);
const isValueLayout = computed(() => descriptor.value?.layout === LAYOUT_VALUE);
// 新增态没有 key 可操作；写操作还要按保存权限过滤，与成员按钮同源，避免点了才被后端拒
const visibleOps = computed(() => {
    if (props.store.creating.value || !caps.value.ops) {
        return [];
    }
    return (descriptor.value?.ops ?? []).filter((item) => !item.write || hasPerm(PERM_DATA_SAVE));
});

/**
 * 单值布局的正文列 = 描述符的第一列，其余表单字段作为正文旁边的辅助控件，
 * 因此视角新增辅助入参（如二进制开关）时本组件不需要改动
 */
const contentColumn = computed(() => columns.value[0]);
/** 行内「复制值」取的内容列：有独立值列时复制值，否则退回首列（如单列集合） */
const copyColumn = computed(() => columns.value.find((column) => column.field === 'value') ?? columns.value[0]);
/**
 * 正文是否可直接编辑：视角表单里存在与正文列同名的字段才可编辑。
 * string 的表单有 value 字段；HyperLogLog 唯一一行是统计出的基数，只能只读展示 + 走「添加成员」，
 * 否则会把统计结果当成元素再写回去
 */
const isValueEditable = computed(() => (descriptor.value?.form?.fields ?? []).some((field) => field.prop === contentColumn.value?.field));
const sideFields = computed<JsonField[]>(() => {
    const content = contentColumn.value?.field;
    return (descriptor.value?.form?.fields ?? []).filter((field) => field.prop !== content);
});
const valueContent = computed(() => {
    const row = props.store.members.value[0];
    const column = contentColumn.value;
    if (!row || !column) {
        return '';
    }
    return columnValue(row, column);
});

const { dialog, dialogKind, opResult, openMember, openOp, submit } = useKeyFormDialog(props.store);

/** 成员写入交给宿主默认提交（内置成功提示与关闭），视角操作交给 @confirm */
const memberConfirm = computed(() => (dialogKind() === 'member' ? submit : undefined));

function cellText(row: RedisKeyMember, column: RedisViewColumn): string {
    const text = columnValue(row, column);
    if (!text) {
        return text;
    }
    if (column.value === 'time') {
        return new Date(Number(text)).toLocaleString();
    }
    return column.value === 'ttl' ? ttlCellText(text, t('redis.permanent')) : text;
}

const onCreate = () => openMember('create', undefined, props.ttl);

const onEdit = (row: RedisKeyMember) => openMember('update', row);

const onDelete = async (row: RedisKeyMember) => {
    await props.store.deleteMembers([row]);
    Msg.deleteSuccess();
};

const onBatchDelete = async () => {
    const rows = selection.value;
    if (!rows.length) {
        return;
    }
    await props.store.deleteMembers(rows);
    Msg.deleteSuccess();
    onToggleBatch();
};

const onSelectionChange = (rows: RedisKeyMember[]) => {
    selection.value = rows;
};

const onSearch = () => props.store.search(filterValue.value.trim());

const onCopyMember = async (row: RedisKeyMember) => {
    const column = copyColumn.value;
    if (!column) {
        return;
    }
    await copyToClipboard(columnValue(row, column));
};

const onOp = (opName: string) => openOp(opName);

/** 单值布局的保存：正文取编辑器内容，辅助控件的值按原 prop 一并提交 */
const onSaveValue = async () => {
    const content = contentColumn.value;
    if (!content) {
        return;
    }

    const args: Record<string, string> = { ...toArgs(sideForm.value), [content.field]: viewerRef.value?.getContent() ?? '' };
    saving.value = true;
    try {
        await props.store.saveMember(props.store.creating.value ? 'create' : 'update', args, null, props.ttl);
        // 表格视角的提示由表单宿主负责，单值面板没有宿主，成功反馈在这里补上
        Msg.saveSuccess();
        dirty.value = false;
    } finally {
        saving.value = false;
    }
};

const copyResult = async () => {
    // 复制提示由 copyToClipboard 统一发出，叠两条会出现两个一模一样的 toast
    await copyToClipboard(opResult.content);
};

// 辅助控件随行数据回填：二进制内容读出来是 base64 且带 binary 标记，
// 开关必须跟着打开，否则保存会把 base64 文本当原文写回，直接损坏数据
function syncSideForm() {
    const content = contentColumn.value?.field;
    const prefilled = prefillForm(descriptor.value?.form, props.store.members.value[0]);
    if (content) {
        delete prefilled[content];
    }
    sideForm.value = prefilled;
}

// 只有换视角 / 换 key 才重置关键词：成员行刷新（尤其是一次过滤后的结果变化）不能重置，
// 否则输入框里的关键词会自己消失，界面正显示着过滤结果却看不出过滤条件是什么
watch(
    () => [descriptor.value?.view, props.store.meta.value?.key],
    () => {
        filterValue.value = '';
    }
);

watch(
    () => [descriptor.value?.view, props.store.members.value[0]],
    () => {
        syncSideForm();
        // 换行/重新加载都以服务端为准，未保存标记不能残留
        dirty.value = false;
    },
    { immediate: true }
);
</script>

<style lang="scss" scoped>
@use './toolbar.scss' as *;

.member-table {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: $redis-radius;

    :deep(.el-table__inner-wrapper::before) {
        display: none;
    }

    :deep(th.el-table__cell) {
        background-color: var(--el-fill-color-lighter);
        color: var(--el-text-color-secondary);
        font-weight: 500;
    }

    :deep(td.el-table__cell) {
        padding: 7px 0;
    }

    .row-btn {
        // 行内三个图标按钮必须在一行 128px 内排下：组件库默认的小尺寸按钮内边距 +
        // 12px 相邻间距会算出 ~135px，导致图标折行、行高撑开
        height: 24px;
        margin: 0 !important;
        padding: 0 5px;
        color: var(--el-text-color-secondary);
        transition: color $redis-duration $redis-ease;
    }

    .row-btn:hover {
        color: var(--el-color-primary);
    }

    .row-btn--danger:hover {
        color: var(--el-color-danger);
    }
}

// 行操作容器：固定列宽下靠 flex 排开，不依赖组件库的相邻按钮外边距
.row-actions {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 2px;
}

// 值列用等宽字体，长值省略号 + tooltip，行高更从容
.member-table :deep(.cell) {
    line-height: 20px;
}

.value-pane {
    gap: 0;
}

.value-footer {
    padding-top: 10px;
    margin-top: 10px;
    border-top: 1px solid var(--el-border-color-lighter);
}

.value-side-form {
    flex: 1;
    min-width: 0;
}

// 未保存提示：小圆点 + 文字，只在正文与服务端不一致时出现
.dirty-flag {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    color: var(--el-color-warning);
    font-size: 12px;

    .dirty-dot {
        width: 6px;
        height: 6px;
        border-radius: 50%;
        background-color: currentColor;
    }
}

.op-result {
    white-space: pre-wrap;
    word-break: break-all;
}
</style>
