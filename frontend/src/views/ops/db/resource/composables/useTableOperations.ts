/**
 * 表/对象操作 composable：从 DbDataOp.vue 提取的树节点操作回调。
 *
 * 包含：编辑表、删除表、生成DDL、重命名表、复制表、查看对象属性/DDL 等。
 * 这些回调均围绕「树节点右键菜单 → 弹框 → 执行 SQL → 刷新树」的统一模式，
 * 与标签页管理无直接关联，故独立为 composable。
 */
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import SqlExecBox from '@/views/ops/db/sql-editor/SqlExecBox';
import { formatSql } from '@/views/ops/db/sql-editor/utils/formatSql';
import { ElCheckbox, ElMessageBox } from 'element-plus';
import { h, ref, type Ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { dbApi } from '../../api';
import { DbInst } from '../../db';
import { getDbDialect, type ColumnDefinition, type IndexDefinition, type TableInfoEditContext } from '../../dialect/index';
import type { TreeNodeCallbackData, TableOpData } from '../../types';

/** 表创建/编辑弹框状态 */
export interface TableCreateDialogState {
    visible: boolean;
    title: string;
    activeName: string;
    dbId: number;
    version: string;
    db: string;
    dbType: string;
    data: TableOpData | null;
    parentKey: string;
}

/** DDL 查看弹框状态 */
export interface DdlDialogState {
    visible: boolean;
    ddl: string;
}

/** 对象属性弹框状态（序列等无行数据的对象） */
export interface PropsDialogState {
    visible: boolean;
    name: string;
    attrs: Record<string, unknown>;
    args: null | { id: number; db: string; type: string; schema?: string; kind: string; name: string };
}

export interface UseTableOperationsParams {
    nowDbInst: Ref<DbInst>;
    tableCreateDialog: Ref<TableCreateDialogState>;
    ddlDialog: Ref<DdlDialogState>;
    propsDialog: Ref<PropsDialogState>;
    chooseTableName: Ref<string>;
    reloadNode: (nodeKey: string) => void;
}

export function useTableOperations(params: UseTableOperationsParams) {
    const { t } = useI18n();
    const { nowDbInst, tableCreateDialog, ddlDialog, propsDialog, chooseTableName, reloadNode } = params;

    /** 编辑表：加载列/索引元数据并打开编辑抽屉 */
    const onEditTable = async (data: TreeNodeCallbackData) => {
        let { db, id, tableName, tableComment, type, parentKey, key, version } = data.params;
        if (tableName) {
            tableCreateDialog.value.title = useI18nEditTitle('db.table');
            let [indexes, columns] = await Promise.all([
                dbApi.tableIndex.request({ id, db, tableName }),
                dbApi.columnMetadata.request({ id, db, tableName }),
            ]);

            // 预处理：在抽屉打开前完成数据转换，避免 watch(visible) 阻塞打开动画
            const fieldsRes: ColumnDefinition[] = [];
            const fieldsOld: ColumnDefinition[] = [];
            const indexColumns: { name: string; comment: string }[] = [];
            const indexesRes: IndexDefinition[] = [];
            const indexesOld: IndexDefinition[] = [];

            if (columns && Array.isArray(columns) && columns.length > 0) {
                columns.forEach((a) => {
                    let defaultValue = '';
                    if (a.columnDefault) {
                        defaultValue = a.columnDefault.trim().replace(/^'|'$/g, '');
                        defaultValue = defaultValue.replace("'::character varying", '');
                    }
                    let field: ColumnDefinition = {
                        name: a.columnName,
                        oldName: a.columnName,
                        type: a.dataType,
                        value: defaultValue,
                        length: a.showLength ?? '',
                        numScale: a.showScale ?? '',
                        nullable: a.nullable ?? false,
                        isPrimaryKey: a.isPrimaryKey ?? false,
                        autoIncrement: a.autoIncrement ?? false,
                        comment: a.columnComment ?? '',
                    };
                    fieldsRes.push(field);
                    fieldsOld.push(structuredClone(field));
                    indexColumns.push({ name: a.columnName, comment: a.columnComment ?? '' });
                });
            }

            if (indexes && Array.isArray(indexes) && indexes.length > 0) {
                indexes
                    .filter((a) => (a as any).indexName !== 'PRIMARY')
                    .forEach((a) => {
                        const idx = a as any;
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

            DbInst.initColumns(columns ?? []);

            const row = { tableName, tableComment };
            tableCreateDialog.value.data = {
                edit: true,
                row,
                indexes,
                columns,
                formData: {
                    tableName,
                    tableComment: tableComment ?? '',
                    oldTableName: tableName,
                    oldTableComment: tableComment ?? '',
                    db,
                    fields: { res: fieldsRes, oldFields: fieldsOld },
                    indexes: { res: indexesRes, oldIndexes: indexesOld, columns: indexColumns },
                },
            };
            tableCreateDialog.value.parentKey = parentKey ?? '';
        } else {
            tableCreateDialog.value.title = useI18nCreateTitle('db.table');
            tableCreateDialog.value.data = { edit: false, row: {} };
            tableCreateDialog.value.parentKey = key ?? '';
        }

        tableCreateDialog.value.activeName = '1';
        tableCreateDialog.value.dbId = id;
        tableCreateDialog.value.version = version ?? '';
        tableCreateDialog.value.db = db;
        tableCreateDialog.value.dbType = type;
        tableCreateDialog.value.visible = true;
    };

    /** 删除表 */
    const onDeleteTable = async (data: TreeNodeCallbackData) => {
        let { db, id, tableName, parentKey, type } = data.params;
        await useI18nDeleteConfirm(tableName);

        const sql = getDbDialect(type).getDropTableSql(db, tableName ?? '');

        dbApi.sqlExec.request({ id, db, sql }).then((res) => {
            let success = true;
            for (let re of res) {
                if (re.errorMsg) {
                    success = false;
                    Msg.error(`${re.sql} -> ${re.errorMsg}`);
                }
            }
            if (success) {
                Msg.deleteSuccess();
                setTimeout(() => {
                    parentKey && reloadNode(parentKey);
                }, 1000);
            }
        });
    };

    /** 生成表 DDL 并展示 */
    const onGenDdl = async (data: TreeNodeCallbackData) => {
        let { db, id, tableName, type } = data.params;
        chooseTableName.value = tableName ?? '';
        let res = await dbApi.tableDdl.request({ id, db, tableName });
        ddlDialog.value.ddl = await formatSql(res, getDbDialect(type).getInfo().formatSqlDialect);
        ddlDialog.value.visible = true;
    };

    /** 查看扩展对象（视图/序列等）DDL */
    const onGenObjectDdl = async (args: { id: number; db: string; type: string; schema?: string; kind: string; name: string }) => {
        const ddl = await dbApi.metaObjectDdl.request({ id: args.id, db: args.db, kind: args.kind, schema: args.schema, name: args.name });
        chooseTableName.value = args.name ?? '';
        ddlDialog.value.ddl = await formatSql(ddl, getDbDialect(args.type).getInfo().formatSqlDialect);
        ddlDialog.value.visible = true;
    };

    /** 展示序列等对象属性面板 */
    const onShowObjectProps = (args: {
        id: number;
        db: string;
        type: string;
        schema?: string;
        kind: string;
        name: string;
        attrs: Record<string, unknown>;
    }) => {
        propsDialog.value.name = args.name;
        propsDialog.value.attrs = args.attrs ?? {};
        propsDialog.value.args = args;
        propsDialog.value.visible = true;
    };

    /** 属性面板内「查看DDL」 */
    const onPropsViewDdl = () => {
        const a = propsDialog.value.args;
        if (!a) {
            return;
        }
        propsDialog.value.visible = false;
        void onGenObjectDdl({ ...a });
    };

    /** 重命名表 */
    const onRenameTable = async (data: TreeNodeCallbackData) => {
        let { db, id, tableName, tableComment, parentKey } = data.params;
        let tableData: TableInfoEditContext = {
            db,
            oldTableName: tableName ?? '',
            tableName: tableName ?? '',
            oldTableComment: tableComment ?? '',
            tableComment: tableComment ?? '',
        };

        let value = ref(tableName ?? '');
        const promptValue = await ElMessageBox.prompt('', t('db.renamePrompt', { db, tableName }), {
            inputValue: value.value,
            confirmButtonText: t('common.confirm'),
            cancelButtonText: t('common.cancel'),
        });

        tableData.tableName = promptValue.value;
        let sql = nowDbInst.value.getDialect().getModifyTableInfoSql(tableData);
        if (!sql) {
            Msg.warning('db.noChange');
            return;
        }

        SqlExecBox({
            sql: sql,
            dbId: id,
            db: db,
            formatDialect: nowDbInst.value.getDialect().getInfo().formatSqlDialect,
            runSuccessCallback: () => {
                setTimeout(() => {
                    parentKey && reloadNode(parentKey);
                }, 1000);
            },
        });
    };

    /** 复制表 */
    const onCopyTable = async (data: TreeNodeCallbackData) => {
        let { db, id, tableName, parentKey } = data.params;

        let checked = ref(false);

        await ElMessageBox({
            title: `${t('db.copyTable')}【${tableName}】`,
            type: 'warning',
            message: () =>
                h(ElCheckbox, {
                    label: t('db.isCopyTableData'),
                    modelValue: checked.value,
                    'onUpdate:modelValue': (val: boolean | string | number) => {
                        if (typeof val === 'boolean') {
                            checked.value = val;
                        }
                    },
                }),
            callback: (action: string) => {
                if (action === 'confirm') {
                    dbApi.copyTable.request({ id, db, tableName, copyData: checked.value }).then(() => {
                        Msg.operateSuccess();
                        setTimeout(() => {
                            parentKey && reloadNode(parentKey);
                        }, 1000);
                    });
                }
            },
        });
    };

    /** 表编辑弹框提交回调 */
    const onSubmitEditTableSql = () => {
        tableCreateDialog.value.visible = false;
        tableCreateDialog.value.data = { edit: false, row: {} };
        reloadNode(tableCreateDialog.value.parentKey);
    };

    return {
        onEditTable,
        onDeleteTable,
        onGenDdl,
        onGenObjectDdl,
        onShowObjectProps,
        onPropsViewDdl,
        onRenameTable,
        onCopyTable,
        onSubmitEditTableSql,
    };
}
