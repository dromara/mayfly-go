/**
 * 表编辑的「变更求值 + DDL 组装」唯一入口。
 *
 * 表编辑组件（DbTableOp.vue）只负责表单状态与交互，变更比较与 SQL 拼装收敛在这里，
 * 因为这两件事最容易出错、也最需要被单测钉住：
 * - 漏判某类变更 → 用户改完「保存成功」，库却什么都没变（静默假成功）；
 * - 多判 → 生成无谓的 ALTER，重写整列甚至丢掉未写出的属性。
 */
import type { ChangeDiff, ColumnDefinition, DbDialect, IndexDefinition, TableEditContext } from '../dialect/types';

export interface TableEditChanges {
    columns: ChangeDiff<ColumnDefinition>;
    indexes: ChangeDiff<IndexDefinition>;
    tableInfoChanged: boolean;
}

/** 变更比较时的取值归一：空值一律视为空串，数字与字符串按文本比较（表单 numeric 输入会得到 number） */
function normalized(value: unknown): string {
    return value === undefined || value === null ? '' : String(value);
}

/** 新对象是否真的改过某个字段（以旧对象的字段集合为准，两侧结构相同） */
function isChanged(oldItem: Record<string, unknown>, newItem: Record<string, unknown>): boolean {
    return Object.keys(oldItem).some((field) => normalized(oldItem[field]) !== normalized(newItem[field]));
}

/**
 * 按唯一键比较新旧数组，产出增/删/改。
 *
 * 改名识别：新对象除自身键外还会以 `oldName` 登记，故「只改列名」的列不会被判成
 * 「删一列 + 加一列」（那会丢数据）。
 *
 * @param key 唯一键字段名（列用 name、索引用 indexName）
 */
export function filterChangedData<T extends object>(oldArr: T[], nowArr: T[], key: string): ChangeDiff<T> {
    const diff: ChangeDiff<T> = { del: [], add: [], upd: [], changed: false };
    const pick = (item: object, field: string) => (item as Record<string, unknown>)[field];

    const oldMap = new Map<string, T>();
    oldArr.forEach((item) => oldMap.set(normalized(pick(item, key)), item));

    const newMap = new Map<string, T>();
    nowArr.forEach((item) => {
        newMap.set(normalized(pick(item, key)), item);
        const oldName = normalized(pick(item, 'oldName'));
        if (oldName) {
            newMap.set(oldName, item);
        }
    });

    nowArr.forEach((item) => {
        const name = normalized(pick(item, key));
        const oldName = normalized(pick(item, 'oldName'));
        const isNew = !oldMap.has(name) && (!oldName || !oldMap.has(oldName));
        if (isNew) {
            diff.add.push(item);
        }
    });

    oldArr.forEach((oldItem) => {
        const next = newMap.get(normalized(pick(oldItem, key)));
        if (!next) {
            diff.del.push(oldItem);
            return;
        }
        // 同一新列只产出一条变更：把某列改名成另一已存在列的名字时，newMap 的新名键会被改名项覆盖，
        // 于是「原既有列」与「被改名列」都会命中同一个改名项；不去重会拼出两条完全相同的
        // CHANGE COLUMN（第二条必然因列已被改名而报 Unknown column），把重名的真实错误盖掉
        if (diff.upd.includes(next)) {
            return;
        }
        if (isChanged(oldItem as Record<string, unknown>, next as Record<string, unknown>)) {
            diff.upd.push(next);
        }
    });

    diff.changed = diff.add.length > 0 || diff.del.length > 0 || diff.upd.length > 0;
    return diff;
}

/**
 * 主键列必须 NOT NULL（所有方言一致）。
 *
 * 勾选「主键」时若该列仍可空，直接 `ADD PRIMARY KEY` 会被库拒绝（MySQL 1171、
 * PG 在含空值时同样失败），用户看到的是「保存成功、执行报错」的错位体验。
 * 这里在求值前把主键列归一为 NOT NULL：既保证 DDL 合法，又让同一次保存顺带把可空改掉。
 */
function withPrimaryKeyNotNull(fields: ColumnDefinition[]): ColumnDefinition[] {
    return fields.map((field) => (field.isPrimaryKey && field.nullable ? { ...field, nullable: false } : field));
}

/** 求值整表的变更（列 / 索引 / 表名与注释） */
export function evaluateTableEdit(data: TableEditContext): TableEditChanges {
    return {
        columns: filterChangedData(data.fields.oldFields, data.fields.res, 'name'),
        indexes: filterChangedData(data.indexes.oldIndexes, data.indexes.res, 'indexName'),
        tableInfoChanged: data.tableName !== data.oldTableName || data.tableComment !== data.oldTableComment,
    };
}

/** 统一语句收尾：去掉重复分号后逐条以 `;` 结尾，避免弹出 `;;` 这类空语句 */
function joinStatements(parts: string[]): string {
    return parts
        .map((part) => part.trim())
        .filter(Boolean)
        .map((part) => `${part.replace(/;+$/, '')};`)
        .join('\n');
}

/**
 * 生成表编辑的 SQL：新建走建表+建索引，编辑走「表信息 / 列 / 索引」三段变更。
 * 无任何变更时返回空串，由调用方提示「无更改」。
 *
 * 顺序不可调换：改名必须最先执行——列与索引语句引用的都是**新表名**，
 * 若排在改名之前，会对着尚不存在的表名发 ALTER 而整批失败。
 * （注释也随改名语句一起走旧表名，故表信息段整体前置最稳妥）
 */
export function buildTableEditSql(dialect: DbDialect, data: TableEditContext, isEdit: boolean): string {
    if (!isEdit) {
        const createTable = dialect.getCreateTableSql(data);
        const createIndex = data.indexes.res.length > 0 ? dialect.getCreateIndexSql(data) : '';
        return joinStatements([createTable, createIndex]);
    }

    // 主键列先归一为 NOT NULL，再交给方言渲染（见 withPrimaryKeyNotNull 注释）
    const normalized: TableEditContext = { ...data, fields: { ...data.fields, res: withPrimaryKeyNotNull(data.fields.res) } };
    const changes = evaluateTableEdit(normalized);
    return joinStatements([
        changes.tableInfoChanged ? dialect.getModifyTableInfoSql(normalized) : '',
        changes.columns.changed ? dialect.getModifyColumnSql(normalized, normalized.tableName, changes.columns) : '',
        changes.indexes.changed ? dialect.getModifyIndexSql(normalized, normalized.tableName, changes.indexes) : '',
    ]);
}
