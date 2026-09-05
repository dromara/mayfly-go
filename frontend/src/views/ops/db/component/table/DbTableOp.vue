<template>
    <auto-form-drawer ref="drawerRef" v-model:visible="visible" :title="title" :items="tableItems" :data="editData" size="75%" @opened="onOpened" @cancel="reset">
        <template #fieldsTabs>
            <el-tabs v-model="activeName">
                <el-tab-pane :label="$t('db.column')" name="1">
                    <el-table ref="tableRef" :data="form.fields.res" :height="tableHeight">
                        <el-table-column
                            :prop="item.prop"
                            :label="$t(item.label)"
                            v-for="item in state.fieldColNames"
                            :key="item.prop"
                            :width="item.width"
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
                                    :disabled="disableEditIncr()"
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
                    <el-table :data="form.indexs.res" :height="tableHeight">
                        <el-table-column :prop="item.prop" :label="$t(item.label)" v-for="item in state.indexColNames" :key="item.prop">
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
                                    <el-option v-for="cl in form.indexs.columns" :key="cl.name" :label="cl.name" :value="cl.name">
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
            </el-tabs>
        </template>

        <template #footer>
            <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
            <el-button :loading="state.btnloading" @click="submit()" type="primary">{{ $t('common.save') }}</el-button>
        </template>
    </auto-form-drawer>
</template>

<script lang="ts" setup>
import { Msg } from '@/hooks/useI18n';
import { computed, nextTick, reactive, ref, Ref, toRef, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { DbInst } from '../../db';
import { DbDialect, DbType, getDbDialect, IndexDefinition, RowDefinition } from '../../dialect/index';
import SqlExecBox from '../sqleditor/SqlExecBox';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import type { TableOpData } from '../../types';

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
const emit = defineEmits(['cancel', 'val-change', 'submit-sql']);

const visible = defineModel<boolean>('visible', { default: false });

let dbDialect: Ref<DbDialect> = computed(() => getDbDialect(props.dbType!, props.version));

type ColName = {
    prop: string;
    label: string;
    width?: number;
};

/** 表索引原始数据 (tableIndex API 返回的动态结构) */
interface TableOpIndex {
    indexName: string;
    columnName?: string;
    indexType: string;
    isUnique?: boolean;
    indexComment?: string;
    [key: string]: unknown;
}

const tableHeight = 'calc(100vh - 320px)';

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => Promise<unknown> }>('drawerRef');
const tableRef = useTemplateRef<{ $el: HTMLElement }>('tableRef');

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

/** 表编辑表单声明（列/索引编辑 tabs 为 custom 插槽） */
const tableItems: AutoFormItem[] = [
    { prop: 'tableName', label: 'db.tableName', span: 12, props: { size: 'small' } },
    { prop: 'tableComment', label: 'db.comment', span: 12, props: { size: 'small' } },
    { prop: 'fieldsTabs', type: 'custom' },
];

/** 传给 AutoFormDrawer 的回填数据：由 props.data 的行列元数据转换为表结构编辑形态（colNames 等纯渲染结构不进入表单） */
const editData = computed<AutoFormData | null>(() => {
    const data = props.data;
    if (!data) {
        return null;
    }
    const { row, indexs, columns } = data;
    // 回显表名表注释
    const tableName = row.tableName as string;
    const tableComment = row.tableComment as string;

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
                // 解决高斯的默认值问题
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
            fieldsOld.push(JSON.parse(JSON.stringify(field)));
            // 索引字段下拉选项
            indexColumns.push({ name: a.columnName, remark: a.columnComment ?? '' });
        });
    }

    // 回显索引
    if (indexs && Array.isArray(indexs) && indexs.length > 0) {
        // 索引过滤掉主键
        indexs
            .filter((a) => (a as TableOpIndex).indexName !== 'PRIMARY')
            .forEach((a) => {
                const idx = a as TableOpIndex;
                let index: IndexDefinition = {
                    indexName: idx.indexName,
                    columnNames: idx.columnName?.split(',') ?? [],
                    unique: idx.isUnique || false,
                    indexType: idx.indexType,
                    indexComment: idx.indexComment,
                };
                indexsRes.push(index);
                indexsOld.push(JSON.parse(JSON.stringify(index)));
            });
    }

    return {
        tableName,
        tableComment,
        oldTableName: tableName,
        oldTableComment: tableComment,
        db: props.db,
        fields: { res: fieldsRes, oldFields: fieldsOld },
        indexs: { res: indexsRes, oldIndexs: indexsOld, columns: indexColumns },
    } as unknown as AutoFormData;
});

/** 抽屉打开后暂存的内部表单引用（列/索引表格数据与 genSql 均基于它） */
const form = ref<AutoFormData>({ fields: { res: [], oldFields: [] }, indexs: { res: [], oldIndexs: [], columns: [] } });

const onOpened = (formData: AutoFormData) => {
    form.value = formData;
    DbInst.initColumns(props.data?.columns ?? []);
    activeName.value = '1';
};

watch(visible, async (val) => {
    dbDialect.value = getDbDialect(props.dbType!);
});

// 切换到索引tab时，刷新索引字段下拉选项
watch(activeName, (newValue) => {
    if (newValue === '2') {
        form.value.indexs.columns = form.value.fields.res.map((a: RowDefinition) => {
            return { name: a.name, remark: a.remark };
        });
    }
});

const cancel = () => {
    visible.value = false;
};

const addRow = () => {
    form.value.fields.res.push({
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
    form.value.indexs.res.push(dbDialect.value.getDefaultIndex());
};

const addDefaultRows = () => {
    form.value.fields.res.push(...dbDialect.value.getDefaultRows());
};

const deleteRow = (index: number) => {
    form.value.fields.res.splice(index, 1);
};

const deleteIndex = (index: number) => {
    form.value.indexs.res.splice(index, 1);
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
        dbType: dbDialect.value.getInfo().formatSqlDialect,
        runSuccessCallback: () => {
            emit('submit-sql', { tableName: form.value.tableName });
            // cancel();
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
): { del: T[]; add: T[]; upd: T[]; changed: boolean } => {
    let data = {
        del: [] as T[], // 删除的数据
        add: [] as T[], // 新增的数据
        upd: [] as T[], // 修改的数据
        changed: false,
    };

    // 旧数据为空
    if (oldArr && Array.isArray(oldArr) && oldArr.length === 0 && nowArr && Array.isArray(nowArr) && nowArr.length > 0) {
        data.add = nowArr;
        data.changed = true;
        return data;
    }

    // 新数据为空
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
        // 取oldName，因为修改了name，但是oldName不会变
        let oldName = rec['oldName'] as string | undefined;
        oldName && (newMap[oldName] = a);
        if (!oldMap.hasOwnProperty(k) && (!oldName || (oldName && !oldMap.hasOwnProperty(oldName)))) {
            // 新增
            data.changed = true;
            data.add.push(a);
        }
    });

    oldArr.forEach((a) => {
        const rec = a as Record<string, unknown>;
        let k = rec[key] as string;
        let newData = newMap[k];
        if (!newData) {
            // 删除
            data.changed = true;
            data.del.push(a);
        } else {
            const newRec = newData as Record<string, unknown>;
            // 判断每个字段是否相等，否则为修改
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
    let data = form.value as unknown as {
        tableName: string;
        tableComment: string;
        oldTableName: string;
        oldTableComment: string;
        fields: { res: RowDefinition[]; oldFields: RowDefinition[] };
        indexs: { res: IndexDefinition[]; oldIndexs: IndexDefinition[] };
    };
    // 创建表
    if (!props.data?.edit) {
        let createTable = dbDialect.value.getCreateTableSql(data);
        let createIndex = '';
        if (data.indexs.res.length > 0) {
            createIndex = dbDialect.value.getCreateIndexSql(data);
        }
        return createTable + ';' + createIndex;
    } else {
        // 修改列
        let changeColData = filterChangedData(data.fields.oldFields, data.fields.res, 'name');
        let colSql = changeColData.changed ? dbDialect.value.getModifyColumnSql(data, data.tableName, changeColData) : '';
        // 修改索引
        let changeIdxData = filterChangedData(data.indexs.oldIndexs, data.indexs.res, 'indexName');
        let idxSql = changeIdxData.changed ? dbDialect.value.getModifyIndexSql(data, data.tableName, changeIdxData) : '';
        // 修改表名,表注释
        let tableInfoSql =
            data.tableName !== data.oldTableName || data.tableComment !== data.oldTableComment ? dbDialect.value.getModifyTableInfoSql(data) : '';

        let sqlArr = [];
        colSql && sqlArr.push(colSql);
        idxSql && sqlArr.push(idxSql);
        tableInfoSql && sqlArr.push(tableInfoSql);

        return sqlArr.join(';');
    }
};

const reset = () => {
    activeName.value = '1';
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
    // 以表名为前缀
    row.indexName = `${form.value.tableName}_${name}_${suffix}`.replaceAll(' ', '');
    row.indexComment = `${form.value.tableName} ${t('db.table')} (${name.replaceAll('_', ',')})${commentSuffix}`;
};

const disableEditIncr = () => {
    if (DbType.postgresql === props.dbType) {
        return true;
    }

    // 如果是mssql则不能修改自增
    if (props.data?.edit) {
        if (DbType.mssql === props.dbType) {
            return true;
        }
    }

    return false;
};
</script>
