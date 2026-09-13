<template>
    <!-- append-to-body: 抽屉默认渲染在页面 DOM 内, 会被 layout 铬层(tagsview z-index)与祖先堆叠上下文压住
         (页签栏浮在抽屉之上); 挂到 body 彻底脱离页面祖先链, popup z-index 保证盖住一切页面内容 -->
    <el-drawer v-model="visible" :title="title" size="75%" append-to-body>
        <div class="table-editor">
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

                                <el-checkbox v-else-if="item.prop === 'notNull'" size="small" v-model="scope.row.notNull" />

                                <el-checkbox v-else-if="item.prop === 'pri'" size="small" v-model="scope.row.pri" />

                                <el-checkbox
                                    v-else-if="item.prop === 'auto_increment'"
                                    size="small"
                                    v-model="scope.row.auto_increment"
                                    :disabled="disableEditIncr"
                                />

                                <el-input v-else-if="item.prop === 'remark'" size="small" v-model="scope.row.remark" />

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
                    <el-table :data="formData.indexs.res" :height="tableHeight" style="width: 100%">
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
                                    <el-option v-for="cl in formData.indexs.columns" :key="cl.name" :label="cl.name" :value="cl.name">
                                        {{ cl.name + ' - ' + (cl.remark || '') }}
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
                        <el-descriptions v-else :column="3" border size="small" style="margin-top: 12px">
                            <el-descriptions-item :label="$t('db.teTemplateAppliedCols')">{{ formData.fields.res.length }}</el-descriptions-item>
                            <el-descriptions-item :label="$t('db.teTemplateAppliedIdx')">{{ formData.indexs.res.length }}</el-descriptions-item>
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
import { computed, nextTick, reactive, ref, Ref, toRef, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { MagicStick } from '@element-plus/icons-vue';
import { DbInst } from '../db';
import { getDbDialect, getDialectCapabilities } from '../dialect/index';
import type { ChangeDiff, DbDialect, IndexDefinition, RowDefinition, TableEditContext } from '../dialect/index';
import SqlExecBox from '../sql-editor/SqlExecBox';
import type { DbTableIndex, TableOpData } from '../types';
import type { ColumnDefinition, TableDefinition, TableIndexDefinition, SchemaSnapshot } from '../types/schema';
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

const tableHeight = 'calc(100vh - 320px)';

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
            prop: 'notNull',
            label: 'db.notNull',
            width: 60,
        },
        {
            prop: 'pri',
            label: 'db.primaryKey',
            width: 60,
        },
        {
            prop: 'auto_increment',
            label: 'db.autoIncrement',
            width: 60,
        },
        {
            prop: 'remark',
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

/** 列编辑表头：列注释按方言能力剔除（sqlite 不支持列注释） */
const visibleFieldColNames = computed(() => state.fieldColNames.filter((c) => c.prop !== 'remark' || dialectCapabilities.value.supportsColumnComment));

/** 索引编辑表头：索引注释为少数派特性（仅 mysql/mssql 支持），其余方言不渲染该列 */
const visibleIndexColNames = computed(() => state.indexColNames.filter((c) => c.prop !== 'indexComment' || dialectCapabilities.value.supportsIndexComment));

/** 表单数据（直接管理，不再依赖 AutoFormDrawer） */
interface TableFormData {
    tableName: string;
    tableComment: string;
    oldTableName: string;
    oldTableComment: string;
    db?: string;
    fields: { res: RowDefinition[]; oldFields: RowDefinition[] };
    indexs: { res: IndexDefinition[]; oldIndexs: IndexDefinition[]; columns: { name: string; remark: string }[] };
}

const formData = ref<TableFormData>({
    tableName: '',
    tableComment: '',
    oldTableName: '',
    oldTableComment: '',
    fields: { res: [], oldFields: [] },
    indexs: { res: [], oldIndexs: [], columns: [] },
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
        const { row, indexs, columns } = props.data;
        const tableName = (row.tableName as string) || '';
        const tableComment = (row.tableComment as string) || '';

        const fieldsRes: RowDefinition[] = [];
        const fieldsOld: RowDefinition[] = [];
        const indexColumns: { name: string; remark: string }[] = [];
        const indexsRes: IndexDefinition[] = [];
        const indexsOld: IndexDefinition[] = [];

        // 回显列
        if (columns && Array.isArray(columns) && columns.length > 0) {
            columns.forEach((a) => {
                let defaultValue = '';
                if (a.columnDefault) {
                    defaultValue = a.columnDefault.trim().replace(/^'|'$/g, '');
                    defaultValue = defaultValue.replace("'::character varying", '');
                }
                let field: RowDefinition = {
                    name: a.columnName,
                    oldName: a.columnName,
                    type: a.dataType,
                    value: defaultValue,
                    length: a.showLength ?? '',
                    numScale: a.showScale ?? '',
                    notNull: !a.nullable,
                    pri: a.isPrimaryKey ?? false,
                    auto_increment: a.autoIncrement ?? false,
                    remark: a.columnComment ?? '',
                };
                fieldsRes.push(field);
                fieldsOld.push(structuredClone(field));
                indexColumns.push({ name: a.columnName, remark: a.columnComment ?? '' });
            });
        }

        // 回显索引
        if (indexs && Array.isArray(indexs) && indexs.length > 0) {
            indexs
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
                    indexsRes.push(index);
                    indexsOld.push(structuredClone(index));
                });
        }

        formData.value = {
            tableName,
            tableComment,
            oldTableName: tableName,
            oldTableComment: tableComment,
            db: props.db,
            fields: { res: fieldsRes, oldFields: fieldsOld },
            indexs: { res: indexsRes, oldIndexs: indexsOld, columns: indexColumns },
        };

        DbInst.initColumns(props.data?.columns ?? []);
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
        formData.value.indexs.columns = formData.value.fields.res.map((a: RowDefinition) => {
            return { name: a.name, remark: a.remark };
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
        notNull: false,
        pri: false,
        auto_increment: false,
        remark: '',
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
    formData.value.indexs.res.push(dbDialect.value.getDefaultIndex());
};

const addDefaultRows = () => {
    formData.value.fields.res.push(...dbDialect.value.getDefaultRows());
};

const deleteRow = (index: number) => {
    formData.value.fields.res.splice(index, 1);
};

const deleteIndex = (index: number) => {
    formData.value.indexs.res.splice(index, 1);
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
 * 对比两个数组，取出被修改过的对象数组
 * @param oldArr 原对象数组
 * @param nowArr 修改后的对象数组
 * @param key 标志对象唯一属性
 */
const filterChangedData = <T extends object>(
    oldArr: T[],
    nowArr: T[],
    key: string
): ChangeDiff<T> => {
    let data: ChangeDiff<T> = {
        del: [] as T[],
        add: [] as T[],
        upd: [] as T[],
        changed: false,
    };

    if (oldArr && Array.isArray(oldArr) && oldArr.length === 0 && nowArr && Array.isArray(nowArr) && nowArr.length > 0) {
        data.add = nowArr;
        data.changed = true;
        return data;
    }

    if (nowArr && Array.isArray(nowArr) && nowArr.length === 0 && oldArr && Array.isArray(oldArr) && oldArr.length > 0) {
        data.del = oldArr;
        data.changed = true;
        return data;
    }

    let oldMap: Record<string, T> = {},
        newMap: Record<string, T> = {};
    oldArr.forEach((a) => (oldMap[(a as Record<string, unknown>)[key] as string] = a));

    nowArr.forEach((a) => {
        const rec = a as Record<string, unknown>;
        let k = rec[key] as string;
        newMap[k] = a;
        let oldName = rec['oldName'] as string | undefined;
        oldName && (newMap[oldName] = a);
        if (!oldMap.hasOwnProperty(k) && (!oldName || (oldName && !oldMap.hasOwnProperty(oldName)))) {
            data.changed = true;
            data.add.push(a);
        }
    });

    oldArr.forEach((a) => {
        const rec = a as Record<string, unknown>;
        let k = rec[key] as string;
        let newData = newMap[k];
        if (!newData) {
            data.changed = true;
            data.del.push(a);
        } else {
            const newRec = newData as Record<string, unknown>;
            for (let f in rec) {
                let oldV = rec[f] as { toString(): string } | null | undefined;
                let newV = newRec[f] as { toString(): string } | null | undefined;
                if (oldV?.toString() !== newV?.toString()) {
                    data.changed = true;
                    data.upd.push(newData);
                    break;
                }
            }
        }
    });
    return data;
};

const genSql = () => {
    let data = formData.value as unknown as TableEditContext;
    if (!props.data?.edit) {
        let createTable = dbDialect.value.getCreateTableSql(data);
        let createIndex = '';
        if (data.indexs.res.length > 0) {
            createIndex = dbDialect.value.getCreateIndexSql(data);
        }
        return createTable + ';' + createIndex;
    } else {
        let changeColData = filterChangedData(data.fields.oldFields, data.fields.res, 'name');
        let colSql = changeColData.changed ? dbDialect.value.getModifyColumnSql(data, data.tableName, changeColData) : '';
        let changeIdxData = filterChangedData(data.indexs.oldIndexs, data.indexs.res, 'indexName');
        let idxSql = changeIdxData.changed ? dbDialect.value.getModifyIndexSql(data, data.tableName, changeIdxData) : '';
        let tableInfoSql =
            data.tableName !== data.oldTableName || data.tableComment !== data.oldTableComment ? dbDialect.value.getModifyTableInfoSql(data) : '';

        let sqlArr = [];
        colSql && sqlArr.push(colSql);
        idxSql && sqlArr.push(idxSql);
        tableInfoSql && sqlArr.push(tableInfoSql);

        return sqlArr.join(';');
    }
};

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

const formColumns = computed<ColumnDefinition[]>(() => {
    return formData.value.fields.res.map((r) => ({
        name: r.name,
        type: r.type || 'varchar',
        length: String(r.length ?? ''),
        numScale: String(r.numScale ?? ''),
        notNull: !!r.notNull,
        pri: !!r.pri,
        auto_increment: !!r.auto_increment,
        value: r.value || '',
        remark: r.remark || '',
    }));
});

const formIndexes = computed(() => {
    return formData.value.indexs.res.map((idx) => ({
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
 *
 * 两侧的索引类型曾经同名（都叫 IndexDefinition），本文件只能导入一侧、另一侧被遮蔽，
 * 于是这处转换既没有显式的目标类型标注，也漏掉了注释字段：索引注释从未进过快照，
 * diff 两侧恒为 undefined，注释变更永远比不出来。改名后两侧可同时导入，转换关系得以显式写出。
 *
 * @param fields 方言层列定义（表单实际编辑的数据）
 * @param indexs 方言层索引定义
 */
function buildTableDefinition(fields: RowDefinition[], indexs: IndexDefinition[], tableName: string): TableDefinition {
    const columns: ColumnDefinition[] = fields.map((r) => ({
        name: r.name,
        type: r.type || 'varchar',
        length: String(r.length ?? ''),
        numScale: String(r.numScale ?? ''),
        notNull: !!r.notNull,
        pri: !!r.pri,
        auto_increment: !!r.auto_increment,
        value: r.value || '',
        remark: r.remark || '',
    }));

    const indexes: TableIndexDefinition[] = indexs.map((idx) => ({
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
    const data = formData.value as unknown as TableEditContext;
    const tableName = data.tableName || 'temp';

    const beforeTable = buildTableDefinition(data.fields.oldFields || [], data.indexs.oldIndexs || [], tableName);
    const afterTable = buildTableDefinition(data.fields.res, data.indexs.res, tableName);

    const before: SchemaSnapshot = { database: props.db || '', tables: [beforeTable], timestamp: 0 };
    const after: SchemaSnapshot = { database: props.db || '', tables: [afterTable], timestamp: Date.now() };

    return schemaDiffService.diff(before, after);
});

/**
 * 应用模板：表编辑子系统领域模型 → 方言层模型，与 buildTableDefinition 方向相反。
 * 两侧索引类型现已异名（TableIndexDefinition / IndexDefinition），转换方向可直接从标注读出。
 */
const handleApplyTemplate = (table: TableDefinition) => {
    const newFields: RowDefinition[] = table.columns.map((col) => ({
        name: col.name,
        oldName: col.name,
        type: col.type,
        value: col.value || '',
        length: col.length ?? '',
        numScale: col.numScale ?? '',
        notNull: !!col.notNull,
        pri: !!col.pri,
        auto_increment: !!col.auto_increment,
        remark: col.remark || '',
    }));

    const newIndexs: IndexDefinition[] = (table.indexes || []).map((idx) => ({
        indexName: idx.name,
        columnNames: (idx.columns || []).map((c) => c.name),
        unique: !!idx.unique,
        // 模板未指定索引类型时取本方言缺省值（mssql 为 NONCLUSTERED、oracle/dm 为 NORMAL）
        indexType: idx.type || dialectCapabilities.value.defaultIndexType,
        indexComment: '',
    }));

    formData.value.fields.res = newFields;
    formData.value.fields.oldFields = newFields.map((f) => JSON.parse(JSON.stringify(f)));
    formData.value.indexs.res = newIndexs;
    formData.value.indexs.oldIndexs = newIndexs.map((i) => JSON.parse(JSON.stringify(i)));

    Msg.success(t('db.teTemplateApplied'));
};
</script>

<style lang="scss" scoped>
.table-editor {
    padding: 0 4px;
}
</style>
