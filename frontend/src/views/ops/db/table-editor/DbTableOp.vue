<template>
    <!-- append-to-body: 抽屉默认渲染在页面 DOM 内, 会被 layout 铬层(tagsview z-index)与祖先堆叠上下文压住
         (页签栏浮在抽屉之上); 挂到 body 彻底脱离页面祖先链, popup z-index 保证盖住一切页面内容 -->
    <el-drawer v-model="visible" :title="title" size="75%" append-to-body>
        <div class="table-editor" ref="editorRef">
            <!-- 表名/注释行 -->
            <el-row :gutter="16" class="mb-4">
                <el-col :span="dialectCapabilities.supportsTableComment ? 12 : 24">
                    <el-form-item :label="$t('db.tableName')" label-width="80px">
                        <el-input v-model="formData.tableName" size="small" />
                    </el-form-item>
                </el-col>
                <!-- 表注释由方言能力门控（sqlite 不支持），不支持时表名独占整行 -->
                <el-col v-if="dialectCapabilities.supportsTableComment" :span="12">
                    <el-form-item :label="$t('db.comment')" label-width="80px">
                        <el-input v-model="formData.tableComment" size="small" />
                    </el-form-item>
                </el-col>
            </el-row>

            <!-- Tabs -->
            <el-tabs v-model="activeName">
                <el-tab-pane :label="$t('db.column')" name="1">
                    <el-table ref="tableRef" :data="formData.fields.res" :height="tableHeight" style="width: 100%">
                        <el-table-column
                            :prop="item.prop"
                            :label="$t(item.label)"
                            v-for="item in visibleFieldColNames"
                            :key="item.prop"
                            :min-width="item.width || 100"
                        >
                            <template #default="scope">
                                <el-input v-if="item.prop === 'name'" size="small" v-model="scope.row.name" />

                                <el-select v-else-if="item.prop === 'type'" filterable size="small" v-model="scope.row.type" allow-create>
                                    <el-option
                                        v-for="pgsqlType in getDbDialect(dbType!).getInfo().columnTypes"
                                        :key="pgsqlType.dataType"
                                        :value="pgsqlType.udtName"
                                        :label="pgsqlType.dataType"
                                    >
                                        <span v-if="pgsqlType.dataType === pgsqlType.udtName"
                                            >{{ pgsqlType.dataType }}{{ pgsqlType.desc && '：' + pgsqlType.desc }}</span
                                        >
                                        <span v-else>{{ pgsqlType.dataType }}，{{ $t('db.alias') }}: {{ pgsqlType.udtName }} {{ pgsqlType.desc }}</span>
                                    </el-option>
                                </el-select>

                                <el-input v-else-if="item.prop === 'value'" size="small" v-model="scope.row.value" />

                                <el-input v-else-if="item.prop === 'length'" type="number" size="small" v-model.number="scope.row.length" />

                                <el-input v-else-if="item.prop === 'numScale'" type="number" size="small" v-model.number="scope.row.numScale" />

                                <el-checkbox
                                    v-else-if="item.prop === 'nullable'"
                                    size="small"
                                    v-model="scope.row.nullable"
                                    :disabled="scope.row.isPrimaryKey"
                                    :title="scope.row.isPrimaryKey ? $t('db.primaryKeyNotNull') : ''"
                                />

                                <el-checkbox
                                    v-else-if="item.prop === 'isPrimaryKey'"
                                    size="small"
                                    v-model="scope.row.isPrimaryKey"
                                    :disabled="!dialectCapabilities.supportsPrimaryKeyEdit"
                                    :title="dialectCapabilities.supportsPrimaryKeyEdit ? '' : $t('db.primaryKeyEditUnsupported')"
                                    @change="onPrimaryKeyChange(scope.row)"
                                />

                                <el-checkbox
                                    v-else-if="item.prop === 'autoIncrement'"
                                    size="small"
                                    v-model="scope.row.autoIncrement"
                                    :disabled="disableEditIncr"
                                />

                                <el-input v-else-if="item.prop === 'comment'" size="small" v-model="scope.row.comment" />

                                <el-popconfirm v-else-if="item.prop === 'action'" :title="$t('common.deleteConfirm')" @confirm="deleteRow(scope.$index)">
                                    <template #reference>
                                        <el-link type="danger" plain size="small" underline="never">{{ $t('common.delete') }}</el-link>
                                    </template>
                                </el-popconfirm>
                            </template>
                        </el-table-column>
                    </el-table>
                    <el-row class="mt-4">
                        <el-button @click="addDefaultRows()" link type="warning" icon="plus">{{ $t('db.addDefaultColumn') }}</el-button>
                        <el-button @click="addRow()" link type="primary" icon="plus">{{ $t('db.addColumn') }}</el-button>
                    </el-row>
                </el-tab-pane>
                <el-tab-pane :label="$t('db.index')" name="2">
                    <el-table :data="formData.indexes.res" :height="tableHeight" style="width: 100%">
                        <el-table-column :prop="item.prop" :label="$t(item.label)" v-for="item in visibleIndexColNames" :key="item.prop">
                            <template #default="scope">
                                <el-input v-if="item.prop === 'indexName'" size="small" disabled v-model="scope.row.indexName"></el-input>

                                <el-select
                                    v-if="item.prop === 'columnNames'"
                                    v-model="scope.row.columnNames"
                                    multiple
                                    collapse-tags
                                    collapse-tags-tooltip
                                    filterable
                                    size="small"
                                    @change="indexChanges(scope.row)"
                                >
                                    <el-option v-for="cl in formData.indexes.columns" :key="cl.name" :label="cl.name" :value="cl.name">
                                        {{ cl.name + ' - ' + (cl.comment || '') }}
                                    </el-option>
                                </el-select>

                                <el-checkbox v-if="item.prop === 'unique'" size="small" v-model="scope.row.unique" @change="indexChanges(scope.row)">
                                </el-checkbox>

                                <el-input v-if="item.prop === 'indexType'" disabled size="small" v-model="scope.row.indexType" />

                                <el-input v-if="item.prop === 'indexComment'" size="small" v-model="scope.row.indexComment"> </el-input>

                                <el-popconfirm v-else-if="item.prop === 'action'" :title="$t('common.deleteConfirm')" @confirm="deleteIndex(scope.$index)">
                                    <template #reference>
                                        <el-link type="danger" plain size="small" underline="never">{{ $t('common.delete') }}</el-link>
                                    </template>
                                </el-popconfirm>
                            </template>
                        </el-table-column>
                    </el-table>

                    <el-row class="mt-4">
                        <el-button @click="addIndex()" link type="primary" icon="plus">{{ $t('db.addIndex') }}</el-button>
                    </el-row>
                </el-tab-pane>
                <el-tab-pane :label="$t('db.teTemplate')" name="3">
                    <div style="padding: 16px 0">
                        <el-button type="primary" size="small" @click="openTemplateSelector">
                            <el-icon><MagicStick /></el-icon> {{ $t('db.teApplyTemplate') }}
                        </el-button>
                        <el-alert
                            v-if="formData.fields.res.length === 0"
                            :title="$t('db.teTemplateEmptyHint')"
                            type="info"
                            :closable="false"
                            show-icon
                            style="margin-top: 12px"
                        />
                        <el-descriptions v-else :column="3" border size="small" style="margin-top: 12px" label-width="90">
                            <el-descriptions-item :label="$t('db.teTemplateAppliedCols')">{{ formData.fields.res.length }}</el-descriptions-item>
                            <el-descriptions-item :label="$t('db.teTemplateAppliedIdx')">{{ formData.indexes.res.length }}</el-descriptions-item>
                        </el-descriptions>
                    </div>
                </el-tab-pane>
                <el-tab-pane :label="$t('db.teValidation')" name="4">
                    <ValidationResult :errors="validationErrors" :warnings="validationWarnings" />
                </el-tab-pane>
                <el-tab-pane :label="$t('db.teDdlTab')" name="5">
                    <DdlPreviewPanel :ddl="previewDdl" />
                </el-tab-pane>
                <el-tab-pane v-if="props.data?.edit" :label="$t('db.teSchemaDiff')" name="6">
                    <SchemaDiffView v-if="schemaDiff" :diff="schemaDiff" />
                    <el-empty v-else :description="$t('db.teSchemaDiffEmpty')" :image-size="60" />
                </el-tab-pane>
            </el-tabs>
        </div>

        <template #footer>
            <div style="display: flex; justify-content: flex-end; gap: 12px">
                <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                <el-button :loading="state.btnloading" @click="submit()" type="primary">{{ $t('common.save') }}</el-button>
            </div>
        </template>
    </el-drawer>

    <TemplateSelector
        v-if="visible"
        ref="templateSelectorRef"
        :table-name="formData.tableName || ''"
        :dialect="dbDialect?.getInfo()?.formatSqlDialect || 'mysql'"
        @apply="handleApplyTemplate"
    />
</template>

<script lang="ts" setup>
import { Msg } from '@/hooks/useI18n';
import { computed, nextTick, onBeforeUnmount, reactive, ref, Ref, toRef, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { MagicStick } from '@element-plus/icons-vue';
import { buildColumnDefinitions } from '../core/columnDefinition';
import { getDbDialect, getDialectCapabilities } from '../dialect/index';
import type { DbDialect, IndexDefinition, ColumnDefinition, TableEditContext } from '../dialect/index';
import { buildTableEditSql } from './tableChange';
import SqlExecBox from '../sql-editor/SqlExecBox';
import type { DbTableIndex, TableOpData } from '../types';
import type { TableColumnDefinition, TableDefinition, TableIndexDefinition, SchemaSnapshot } from '../types/schema';
import { validationService, schemaDiffService } from './services';
import TemplateSelector from './TemplateSelector.vue';
import ValidationResult from './ValidationResult.vue';
import DdlPreviewPanel from './DdlPreviewPanel.vue';
import SchemaDiffView from './SchemaDiffView.vue';

const { t } = useI18n();

const props = defineProps({
    title: {
        type: String,
    },
    data: {
        type: Object as () => TableOpData | null,
    },
    dbId: {
        type: Number,
    },
    db: {
        type: String,
    },
    dbType: {
        type: String,
    },
    version: {
        type: String,
    },
});

//定义事件
const emit = defineEmits<{
    /** 「生成 SQL 并执行」成功，回传表名，父级据此关闭弹窗并进入该表的编辑态 */
    'submit-sql': [table: { tableName: string }];
}>();

const visible = defineModel<boolean>('visible', { default: false });

const dbDialect: Ref<DbDialect> = computed(() => getDbDialect(props.dbType!, props.version));

/** 方言能力：注释类字段与自增勾选的唯一判据，新增方言只需在自身声明能力，无需改动本文件 */
const dialectCapabilities = computed(() => getDialectCapabilities(dbDialect.value));

type ColName = {
    prop: string;
    label: string;
    width?: number;
};

/**
 * 表格高度 = 抽屉实际可用高度 − 表名行/页签头/底部按钮行。
 *
 * 为什么必须用测量而不是纯 CSS：高度要传到表格，必须穿过 el-tabs 自己渲染的
 * 内容区容器；对它设 flex/height 就是在改 EP 内部布局。既然不改 EP，就由脚本
 * 把实测值直接给表格 :height，EP 侧一行样式都不碰（本文件样式段只剩 padding）。
 *
 * 为什么不用 calc(100vh - N)：抽屉里还有表名行、页签头、底部按钮行与抽屉
 * header/footer，固定扣减必然差几十像素，多出来的部分把 el-drawer 的 body
 * 顶成滚动容器，于是出现「表格内一条 + 抽屉外一条」，外层那条还会带走页签头。
 */
const tableHeight = ref('400px');

const editorRef = useTemplateRef<HTMLElement>('editorRef');

/** 当前可见的页签面板（Element Plus 非激活页签是 display:none） */
const activePane = (editor: HTMLElement) => Array.from(editor.querySelectorAll<HTMLElement>('.el-tab-pane')).find((p) => p.offsetParent !== null);

/**
 * 量算并写回表格高度。
 *
 * 取抽屉 body 内容区下边界，减去页签头下边界（含 margin）与当前页签底部动作行高度，
 * 再留 8px 余量：宁可少几像素（表格下方留白），也绝不 多一像素——多一像素就会把
 * 抽屉 body 顶出滚动条，页签头又会被那条滚动带走。
 */
function computeTableHeight() {
    const editor = editorRef.value;
    if (!editor) return;
    const body = editor.closest('.el-drawer__body') as HTMLElement | null;
    const header = editor.querySelector<HTMLElement>('.el-tabs__header');
    if (!body || !header) return;

    const bodyStyle = getComputedStyle(body);
    const contentBottom = body.getBoundingClientRect().bottom - (parseFloat(bodyStyle.paddingBottom) || 0);
    const headerBottom = header.getBoundingClientRect().bottom + (parseFloat(getComputedStyle(header).marginBottom) || 0);

    // 底部「添加列/添加索引」行只有列/索引页签才有，其余页签不占这块
    const actionRow = activePane(editor)?.querySelector<HTMLElement>(':scope > .el-row');
    const reserved = actionRow ? actionRow.offsetHeight + (parseFloat(getComputedStyle(actionRow).marginTop) || 0) : 0;

    tableHeight.value = `${Math.max(160, Math.floor(contentBottom - headerBottom - reserved - 8))}px`;
    nextTick(() => tableRef.value?.doLayout?.());
}

/** 抽屉尺寸变化（窗口缩放、侧栏收起）时重算；抽屉关闭即解绑 */
let bodyResizeObserver: ResizeObserver | null = null;
function watchDrawerSize() {
    const body = editorRef.value?.closest('.el-drawer__body');
    if (!body || bodyResizeObserver) return;
    bodyResizeObserver = new ResizeObserver(() => computeTableHeight());
    bodyResizeObserver.observe(body);
}

onBeforeUnmount(() => {
    bodyResizeObserver?.disconnect();
    bodyResizeObserver = null;
});

// 抽屉打开后量算（此时才有真实尺寸），关闭时解绑观察
watch(visible, async (open) => {
    if (!open) {
        bodyResizeObserver?.disconnect();
        bodyResizeObserver = null;
        return;
    }
    await nextTick();
    computeTableHeight();
    watchDrawerSize();
});

const tableRef = useTemplateRef<{ $el: HTMLElement; doLayout: () => void }>('tableRef');

const state = reactive({
    btnloading: false,
    activeName: '1',
    fieldColNames: [
        {
            prop: 'name',
            label: 'db.columnName',
            width: 200,
        },
        {
            prop: 'type',
            label: 'common.type',
            width: 120,
        },
        {
            prop: 'length',
            label: 'db.length',
            width: 120,
        },
        {
            prop: 'numScale',
            label: 'db.numScale',
            width: 120,
        },
        {
            prop: 'value',
            label: 'db.defaultValue',
            width: 120,
        },

        {
            prop: 'nullable',
            label: 'db.nullable',
            width: 60,
        },
        {
            prop: 'isPrimaryKey',
            label: 'db.primaryKey',
            width: 60,
        },
        {
            prop: 'autoIncrement',
            label: 'db.autoIncrement',
            width: 60,
        },
        {
            prop: 'comment',
            label: 'db.comment',
        },
        {
            prop: 'action',
            label: 'common.operation',
            width: 70,
        },
    ] as ColName[],
    indexColNames: [
        {
            prop: 'indexName',
            label: 'common.name',
        },
        {
            prop: 'columnNames',
            label: 'db.columnName',
        },
        {
            prop: 'unique',
            label: 'db.unique',
        },
        {
            prop: 'indexType',
            label: 'common.type',
        },
        {
            prop: 'indexComment',
            label: 'db.comment',
        },
        {
            prop: 'action',
            label: 'common.operation',
        },
    ] as ColName[],
});

const activeName = toRef(state, 'activeName');

// 切换页签后底部动作行是否存在会变（列/索引有「添加…」行，其余没有），重新量算表格高度
watch(activeName, async () => {
    await nextTick();
    computeTableHeight();
});

/** 列编辑表头：列注释按方言能力剔除（sqlite 不支持列注释） */
const visibleFieldColNames = computed(() => state.fieldColNames.filter((c) => c.prop !== 'comment' || dialectCapabilities.value.supportsColumnComment));

/** 索引编辑表头：索引注释为少数派特性（仅 mysql/mssql 支持），其余方言不渲染该列 */
const visibleIndexColNames = computed(() => state.indexColNames.filter((c) => c.prop !== 'indexComment' || dialectCapabilities.value.supportsIndexComment));

/** 表单数据（直接管理，不再依赖 AutoFormDrawer） */
interface TableFormData {
    tableName: string;
    tableComment: string;
    oldTableName: string;
    oldTableComment: string;
    db?: string;
    /** 现状主键的约束名（pg/mssql 撤销主键要按名字 DROP CONSTRAINT） */
    primaryKeyName?: string;
    fields: { res: ColumnDefinition[]; oldFields: ColumnDefinition[] };
    indexes: { res: IndexDefinition[]; oldIndexes: IndexDefinition[]; columns: { name: string; comment: string }[] };
}

const formData = ref<TableFormData>({
    tableName: '',
    tableComment: '',
    oldTableName: '',
    oldTableComment: '',
    fields: { res: [], oldFields: [] },
    indexes: { res: [], oldIndexes: [], columns: [] },
});

/** 抽屉打开时初始化表单数据 */
watch(visible, async (val) => {
    if (!val || !props.data) {
        return;
    }

    // 优先使用预处理的 formData（由 onEditTable 在抽屉打开前构建，避免阻塞打开动画）
    if (props.data.formData) {
        formData.value = props.data.formData;
    } else {
        // 回退：无预处理数据时，在此处转换（新建表等场景）
        const { row, indexes, columns } = props.data;
        const tableName = (row.tableName as string) || '';
        const tableComment = (row.tableComment as string) || '';

        const fieldsRes: ColumnDefinition[] = buildColumnDefinitions(columns && Array.isArray(columns) ? columns : []);
        const fieldsOld: ColumnDefinition[] = fieldsRes.map((field) => structuredClone(field));
        const indexColumns: { name: string; comment: string }[] = fieldsRes.map((field) => ({ name: field.name, comment: field.comment }));
        const indexesRes: IndexDefinition[] = [];
        const indexesOld: IndexDefinition[] = [];

        // 回显索引
        if (indexes && Array.isArray(indexes) && indexes.length > 0) {
            indexes
                .filter((a) => (a as DbTableIndex).indexName !== 'PRIMARY')
                .forEach((a) => {
                    const idx = a as DbTableIndex;
                    let index: IndexDefinition = {
                        indexName: idx.indexName,
                        columnNames: idx.columnName?.split(',') ?? [],
                        unique: idx.isUnique || false,
                        indexType: idx.indexType,
                        indexComment: idx.indexComment,
                    };
                    indexesRes.push(index);
                    indexesOld.push(structuredClone(index));
                });
        }

        formData.value = {
            tableName,
            tableComment,
            oldTableName: tableName,
            oldTableComment: tableComment,
            db: props.db,
            fields: { res: fieldsRes, oldFields: fieldsOld },
            indexes: { res: indexesRes, oldIndexes: indexesOld, columns: indexColumns },
        };
    }

    activeName.value = '1';

    // 抽屉已移除 destroy-on-close，内容常驻 DOM；隐藏期间 el-table 测不到容器宽度，
    // 重新显示后必须手动 doLayout，否则列宽错乱
    await nextTick();
    tableRef.value?.doLayout();
});

// 切换到索引tab时，刷新索引字段下拉选项
watch(activeName, (newValue) => {
    if (newValue === '2') {
        formData.value.indexes.columns = formData.value.fields.res.map((a: ColumnDefinition) => {
            return { name: a.name, comment: a.comment };
        });
    }
});

const cancel = () => {
    visible.value = false;
};

const addRow = () => {
    formData.value.fields.res.push({
        name: '',
        type: '',
        value: '',
        length: '',
        numScale: '',
        nullable: true,
        isPrimaryKey: false,
        autoIncrement: false,
        comment: '',
    });

    // 滚动到最后一行
    nextTick(() => {
        if (tableRef.value) {
            const lastRow = tableRef.value?.$el.querySelector('.el-table__body-wrapper tbody tr:last-child');
            if (lastRow) {
                lastRow.scrollIntoView({ behavior: 'smooth' });
            }
        }
    });
};

const addIndex = () => {
    formData.value.indexes.res.push(dbDialect.value.getDefaultIndex());
};

const addDefaultRows = () => {
    formData.value.fields.res.push(...dbDialect.value.getDefaultColumns());
};

const deleteRow = (index: number) => {
    formData.value.fields.res.splice(index, 1);
};

const deleteIndex = (index: number) => {
    formData.value.indexes.res.splice(index, 1);
};

/**
 * 勾选主键即强制取消可空：主键列必须 NOT NULL，否则 ADD PRIMARY KEY 会被库拒绝
 * （MySQL 1171 / PG 含空值失败），用户只会看到「保存成功、执行报错」的错位体验。
 */
const onPrimaryKeyChange = (row: ColumnDefinition) => {
    if (row.isPrimaryKey) {
        row.nullable = false;
    }
};

const submit = async () => {
    let sql = genSql();
    if (!sql) {
        Msg.warning('db.noChange');
        return;
    }
    SqlExecBox({
        sql: sql,
        dbId: props.dbId!,
        db: props.db!,
        formatDialect: dbDialect.value.getInfo().formatSqlDialect,
        runSuccessCallback: () => {
            emit('submit-sql', { tableName: formData.value.tableName });
        },
    });
};

/**
 * 生成待执行的表编辑 SQL。
 *
 * 变更求值与拼装由 tableChange.ts 单一持有（纯函数、可单测）：
 * 组件这里只负责把当前表单当作编辑上下文交给方言。
 */
const genSql = () => buildTableEditSql(dbDialect.value, formData.value as TableEditContext, props.data?.edit === true);

const indexChanges = (row: IndexDefinition) => {
    let name = '';
    if (row.columnNames && row.columnNames.length > 0) {
        for (const column of row.columnNames) {
            name += column.replace('_', '').toLowerCase() + '_';
        }
        name = name.substring(0, name.length - 1);
    } else {
        return;
    }

    let suffix = row.unique ? 'udx' : 'idx';
    let commentSuffix = row.unique ? t('db.uniqueIndex') : t('db.normalIndex');
    row.indexName = `${formData.value.tableName}_${name}_${suffix}`.replaceAll(' ', '');
    row.indexComment = `${formData.value.tableName} ${t('db.table')} (${name.replaceAll('_', ',')})${commentSuffix}`;
};

/**
 * 自增列可否编辑完全由方言能力声明，本组件不认识任何具体方言：
 * 按新建/编辑分别取 canEditAutoIncrementOnCreate / OnEdit
 * （postgres 系自增由 serial/identity 列类型表达，建表即不可编辑；mssql 的 identity 建表后不可变更）。
 *
 * 无自增概念的方言（clickhouse）不必在此特判——defineCapabilities() 已把
 * supportsAutoIncrement=false 派生为两个 canEdit* 均为 false，不变式只存在一处。
 */
const disableEditIncr = computed(() => {
    const capabilities = dialectCapabilities.value;
    return props.data?.edit ? !capabilities.canEditAutoIncrementOnEdit : !capabilities.canEditAutoIncrementOnCreate;
});

// ==================== 新功能集成 ====================

const templateSelectorRef = useTemplateRef<{ open: () => void }>('templateSelectorRef');

const openTemplateSelector = () => {
    templateSelectorRef.value?.open();
};

const formColumns = computed<TableColumnDefinition[]>(() => {
    return formData.value.fields.res.map((r) => ({
        name: r.name,
        type: r.type || 'varchar',
        length: String(r.length ?? ''),
        numScale: String(r.numScale ?? ''),
        nullable: !!r.nullable,
        isPrimaryKey: !!r.isPrimaryKey,
        autoIncrement: !!r.autoIncrement,
        value: r.value || '',
        comment: r.comment || '',
    }));
});

const formIndexes = computed(() => {
    return formData.value.indexes.res.map((idx) => ({
        name: idx.indexName,
        columns: (Array.isArray(idx.columnNames) ? idx.columnNames : []).map((name: string) => ({ name })),
        unique: !!idx.unique,
        type: (idx.indexType || 'BTREE') as 'BTREE' | 'HASH' | 'FULLTEXT' | 'SPATIAL' | 'NORMAL',
    }));
});

const validationErrors = computed(() => {
    const table = {
        name: formData.value.tableName || 'temp',
        columns: formColumns.value,
        indexes: formIndexes.value,
        constraints: [],
    };
    const result = validationService.validateTable(table);
    return result.errors;
});

const validationWarnings = computed(() => {
    const table = {
        name: formData.value.tableName || 'temp',
        columns: formColumns.value,
        indexes: formIndexes.value,
        constraints: [],
    };
    const result = validationService.validateTable(table);
    return result.warnings;
});

const previewDdl = computed(() => {
    return genSql() || '';
});

/**
 * 方言层模型 → 表编辑子系统领域模型（供 SchemaDiffView 与 validationService 消费）。
 * 方言层 IndexDefinition 与领域层 TableIndexDefinition 是两套模型，此处显式转换并补全索引注释，
 * 保证 diff 两侧口径一致（注释取空串而非 undefined，避免恒不相等而误报变更）。
 *
 * @param fields 方言层列定义（表单实际编辑的数据）
 * @param dialectIndexes 方言层索引定义
 */
function buildTableDefinition(fields: ColumnDefinition[], dialectIndexes: IndexDefinition[], tableName: string): TableDefinition {
    const columns: TableColumnDefinition[] = fields.map((r) => ({
        name: r.name,
        type: r.type || 'varchar',
        length: String(r.length ?? ''),
        numScale: String(r.numScale ?? ''),
        nullable: !!r.nullable,
        isPrimaryKey: !!r.isPrimaryKey,
        autoIncrement: !!r.autoIncrement,
        value: r.value || '',
        comment: r.comment || '',
    }));

    const indexes: TableIndexDefinition[] = dialectIndexes.map((idx) => ({
        name: idx.indexName,
        columns: (Array.isArray(idx.columnNames) ? idx.columnNames : []).map((name) => ({ name })),
        unique: !!idx.unique,
        type: (idx.indexType || 'BTREE') as TableIndexDefinition['type'],
        // 注释统一取空串而非 undefined：before/after 两侧都经本函数构造，取值口径一致才不会误报变更
        comment: idx.indexComment || '',
    }));

    return { name: tableName, columns, indexes, constraints: [] };
}

const schemaDiff = computed(() => {
    if (!props.data?.edit) return null;
    const data = formData.value as TableEditContext;
    const tableName = data.tableName || 'temp';

    const beforeTable = buildTableDefinition(data.fields.oldFields || [], data.indexes.oldIndexes || [], tableName);
    const afterTable = buildTableDefinition(data.fields.res, data.indexes.res, tableName);

    const before: SchemaSnapshot = { database: props.db || '', tables: [beforeTable], timestamp: 0 };
    const after: SchemaSnapshot = { database: props.db || '', tables: [afterTable], timestamp: Date.now() };

    return schemaDiffService.diff(before, after);
});

/**
 * 应用模板：表编辑子系统领域模型 → 方言层模型，与 buildTableDefinition 方向相反。
 * 两侧索引类型现已异名（TableIndexDefinition / IndexDefinition），转换方向可直接从标注读出。
 */
const handleApplyTemplate = (table: TableDefinition) => {
    const newFields: ColumnDefinition[] = table.columns.map((col) => ({
        name: col.name,
        oldName: col.name,
        type: col.type,
        value: col.value || '',
        length: col.length ?? '',
        numScale: col.numScale ?? '',
        nullable: !!col.nullable,
        isPrimaryKey: !!col.isPrimaryKey,
        autoIncrement: !!col.autoIncrement,
        comment: col.comment || '',
    }));

    const newIndexes: IndexDefinition[] = (table.indexes || []).map((idx) => ({
        indexName: idx.name,
        columnNames: (idx.columns || []).map((c) => c.name),
        unique: !!idx.unique,
        // 模板未指定索引类型时取本方言缺省值（mssql 为 NONCLUSTERED、oracle/dm 为 NORMAL）
        indexType: idx.type || dialectCapabilities.value.defaultIndexType,
        indexComment: '',
    }));

    formData.value.fields.res = newFields;
    formData.value.fields.oldFields = newFields.map((f) => JSON.parse(JSON.stringify(f)));
    formData.value.indexes.res = newIndexes;
    formData.value.indexes.oldIndexes = newIndexes.map((i) => JSON.parse(JSON.stringify(i)));

    Msg.success(t('db.teTemplateApplied'));
};
</script>

<style lang="scss" scoped>
/*
 * 这里只有组件自己的内边距——el-drawer / el-tabs / el-table 一行样式都不碰。
 * 表格高度由 computeTableHeight() 实测后写给 :height（见脚本），
 * 这样既不改 EP 内部布局，也不会把抽屉 body 顶出第二条滚动条。
 */
.table-editor {
    padding: 0 4px;
}
</style>
