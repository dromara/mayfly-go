/**
 * 值补全的上下文判定。
 *
 * 单独成模块的理由：判定「光标处是在填某个列的值」靠的是光标前的文本形态，
 * 这是整条补全链里最容易误判的一步（判错会把列名候选整批挤掉）。把它做成纯函数，
 * 既能被用例逐条钉住，也不参与贡献者链的异步取数。
 */

/** 值补全定位到的目标列 */
export interface ValueTarget {
    /** 列名（已去掉引用符） */
    column: string;
    /** 列的限定符（表别名或表名），未限定则为空 */
    qualifier: string;
    /** 光标前已输入的值片段，用于过滤候选 */
    partial: string;
    /** 值片段前是否已有开引号：有则插入裸值，否则插入会多出一对引号 */
    quoted: boolean;
}

/**
 * `列 = / <> / > / < / IN ( / LIKE ` 之后紧跟值片段的形态。
 *
 * 只认这几类比较运算符：不认 `BETWEEN` 的第二操作数与函数调用参数等形态，
 * 宁可少给候选也不要误判成值位置而把列名/函数候选整批挤掉。
 *
 * 片段分两支：开引号内可以含空格（`'new york`），无引号时遇到空格即止——
 * 否则 `WHERE a = 1 AND b` 会被读成「在填 a 的值、片段是 '1 AND b'」，那是在给错列的候选
 */
const VALUE_CONTEXT =
    /(?:(?<qualifier>[\w$]+)\s*\.\s*)?(?<column>`[^`]+`|"[^"]+"|\[[^\]]+\]|[\w$]+)\s*(?:=|<>|!=|>|<|>=|<=|\bIN\s*\(|\bLIKE\s+)\s*(?:'(?<quotedPartial>[^']*)|(?<barePartial>\S*))$/i;

/** 去掉标识符引用符 */
function unquote(name: string): string {
    return name.replace(/^[`"[]/, '').replace(/[`"\]]$/, '');
}

/**
 * 从光标前的文本判定是否处于某个列的值位置。
 *
 * @param prefix 光标之前的语句文本（不含光标之后的内容）
 */
export function resolveValueTarget(prefix: string): ValueTarget | null {
    const matched = VALUE_CONTEXT.exec(prefix);
    if (!matched?.groups) {
        return null;
    }
    const { qualifier, column, quotedPartial, barePartial } = matched.groups;
    if (!column) {
        return null;
    }
    // 开引号分支命中时 quotedPartial 存在（哪怕是空串），不能用真值判断
    const quoted = quotedPartial !== undefined;
    return {
        column: unquote(column),
        qualifier: qualifier ? unquote(qualifier) : '',
        partial: (quoted ? quotedPartial : barePartial) ?? '',
        quoted,
    };
}

/** 候选值是否匹配已输入的片段（大小写不敏感的前缀匹配） */
export function matchesValuePartial(value: string, partial: string): boolean {
    if (!partial) {
        return true;
    }
    return value.toLowerCase().startsWith(partial.toLowerCase());
}

/** 该值是否需要引号包裹（数字值裸写，其余按字符串字面量处理） */
export function isNumericLiteral(value: string): boolean {
    return value !== '' && Number.isFinite(Number(value));
}

/**
 * 生成插入文本。
 *
 * 单引号内的值必须把内部单引号双写，否则 `O'Brien` 会截断语句；
 * 已有开引号时不再补引号（用户已经打了引号，补上就是 ''值''）
 */
export function valueInsertText(value: string, quoted: boolean): string {
    if (quoted) {
        return value.replace(/'/g, "''");
    }
    if (isNumericLiteral(value)) {
        return value;
    }
    return `'${value.replace(/'/g, "''")}'`;
}
