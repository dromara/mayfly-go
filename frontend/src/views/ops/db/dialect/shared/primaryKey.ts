/**
 * 主键变更的求值（方言无关）。
 *
 * 主键是**表级约束**，列级 MODIFY / ALTER COLUMN 改不动它：勾选或取消「主键」必须另发
 * DROP/ADD PRIMARY KEY 一类的语句，否则就是「保存成功、主键没变」的静默假成功。
 * 各家语法不同（mysql 的 DROP PRIMARY KEY、oracle 的 DROP/ADD CONSTRAINT），
 * 故本模块只负责算清「改没改、改成什么」，渲染交给方言。
 */
import type { ColumnDefinition } from '../types';

export interface PrimaryKeyChange {
    /** 变更后的主键列（按表内列顺序；已把旧名映射成新名） */
    columns: string[];
    /** 与旧主键相比是否发生变化 */
    changed: boolean;
}

/**
 * 求值主键变更。
 *
 * 旧侧用 `oldName` 映射到新列名后再比较：只改列名时主键成员并未变化，
 * 不该触发一次 DROP/ADD PRIMARY KEY（那会在改名过程中多一次约束重建，也可能被引用方挡住）。
 */
export function diffPrimaryKey(oldFields: ColumnDefinition[], newFields: ColumnDefinition[]): PrimaryKeyChange {
    const renameTo = new Map<string, string>();
    newFields.forEach((field) => {
        if (field.oldName && field.oldName !== field.name) {
            renameTo.set(field.oldName, field.name);
        }
    });

    const oldColumns = oldFields.filter((field) => field.isPrimaryKey).map((field) => renameTo.get(field.name) ?? field.name);
    const columns = newFields.filter((field) => field.isPrimaryKey).map((field) => field.name);

    return { columns, changed: oldColumns.join('\u0000') !== columns.join('\u0000') };
}
