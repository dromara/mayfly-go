/**
 * 数据表单的提交语义：新增还是更新既有行。
 *
 * 判定必须来自「弹窗用途」（create/edit），不能来自「表单当前是否有值」——
 * 用户一点「+」再填字段，表单就有值了，用值判会把它当成编辑，生成
 * `UPDATE ... WHERE id = NULL`（影响 0 行却提示成功），是本模块的历史缺陷。
 *
 * edit 模式仍需 oldValue 作为差异基线；没有基线（例如旧实现残留的空壳）时
 * 退回 insert，宁可新建也不要盲更新。
 */
export type FormMode = 'create' | 'edit';

export function resolveFormAction(
    mode: FormMode | undefined,
    oldValue: Record<string, unknown> | null
): 'insert' | 'update' {
    if (mode === 'create' || !oldValue) {
        return 'insert';
    }
    return 'update';
}

/** 参与必填校验的列定义最小契约（只需这三个字段，便于纯函数测试） */
export interface RequiredColumnLike {
    columnName: string;
    nullable?: boolean;
    isPrimaryKey?: boolean;
    autoIncrement?: boolean;
}

/**
 * 该列是否必填：只用于「显示星号 + 提交时校验」，**不交给 el-form-item 的 required**。
 *
 * 一旦把 required 交给 el-form-item，EP 会生成校验规则，并在动态创建字段时立刻跑一次校验，
 * 用户还没看到界面就全红（本模块「打开即红」的根因）。
 */
export function isRequiredColumn(column: RequiredColumnLike): boolean {
    return !column.nullable && !column.isPrimaryKey && !column.autoIncrement;
}

/**
 * 返回第一个「必填但为空」的列名；全部通过返回 null。
 *
 * 这是唯一的校验入口（由提交动作调用），因此可以纯函数化并被单测钉住：
 * 空串 / 纯空白 / undefined / null 都算未填。
 */
export function firstMissingRequired(
    columns: RequiredColumnLike[],
    data: Record<string, unknown>
): string | null {
    for (const column of columns) {
        if (!isRequiredColumn(column)) continue;
        const value = data[column.columnName];
        if (value === undefined || value === null || String(value).trim() === '') {
            return column.columnName;
        }
    }
    return null;
}
