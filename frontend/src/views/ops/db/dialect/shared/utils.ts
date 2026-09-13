/**
 * 方言共享工具函数
 *
 * 抽取各 dialect 中重复的工具逻辑，避免代码重复。
 */

/**
 * 引号转义（用于 SQL 注释中的单引号）
 * 如：comment xx is '注''释' → 最终注释文本为：注'释
 */
export function QuoteEscape(str: string): string {
    if (!str) {
        return '';
    }
    return str.replace(/'/g, "''");
}

/**
 * 子串匹配：检查 text 是否包含 arr 中任意元素（大小写敏感）
 * 原实现分散在 PostgreSQL/Oracle/DM 三个方言中，逻辑完全相同
 */
export function matchType(text: string, arr: string[]): boolean {
    if (!text || !arr || arr.length === 0) {
        return false;
    }
    return arr.some((item) => text.indexOf(item) > -1);
}

/**
 * 从 db 字符串中提取 schema（支持 "db/schema" 格式）
 * 原逻辑在 PostgreSQL/Oracle/MSSQL/DM 的多个方法中重复出现
 */
export function extractSchema(db: string): string {
    const parts = db.split('/');
    return parts.length > 1 ? parts[parts.length - 1] : parts[0];
}

/**
 * 构建带 schema 的表名引用
 */
export function buildSchemaTable(quoteIdentifier: (name: string) => string, db: string, tableName: string): string {
    const schema = extractSchema(db);
    return `${quoteIdentifier(schema)}.${quoteIdentifier(tableName)}`;
}

/**
 * 匹配数值类型关键字，返回匹配结果（无匹配时为 null）
 *
 * 下沉自 DbInst.isNumber：方言层（纯 SQL 策略）不应反向依赖应用层的 DbInst，
 * 否则会形成 dialect → db → completion → dialect 的模块循环并触发 TDZ。
 * 注意返回的是 RegExpMatchArray 而非 boolean，调用方按真值判断即可。
 *
 * @param columnType 字段类型
 */
/**
 * 标准 `LIMIT n` 追加：适用于 mysql/sqlite/clickhouse/postgres/dm 等支持该语法的方言。
 * 不支持的方言（mssql 的 TOP、oracle 的 ROWNUM）在自身 getPreviewSql 内给出实现。
 */
export function appendLimitSql(sql: string, limit: number): string {
    return `${sql} LIMIT ${limit}`;
}

export function matchNumericType(columnType: string): RegExpMatchArray | null {
    return columnType ? columnType.match(/(int|uint|double|float|number|numeric|decimal|byte|bit)/gi) : null;
}

/**
 * 通用数据类型判断（适用于大多数关系型数据库）
 * 原实现分散在 MySQL/PostgreSQL/MSSQL/SQLite/ClickHouse/DM 六个方言中
 */
export function getDefaultDataType(columnType: string): 'string' | 'number' | 'date' | 'time' | 'datetime' {
    const lower = columnType.toLowerCase();
    // 数值类型
    if (
        /^(tinyint|smallint|mediumint|int|integer|bigint|float|double|decimal|numeric|real|number|money|bit|serial|bigserial|smallserial|largeserial|uint|int8|int16|int32|int64|float32|float64)$/i.test(
            lower
        )
    ) {
        return 'number';
    }
    // 日期时间
    if (/datetime|timestamp/i.test(lower)) {
        return 'datetime';
    }
    // 日期
    if (/^date$/i.test(lower) || lower === 'date32') {
        return 'date';
    }
    // 时间
    if (/^time/i.test(lower)) {
        return 'time';
    }
    return 'string';
}

/**
 * 通用值包装（适用于大多数关系型数据库）
 * 原实现分散在 MySQL/PostgreSQL/SQLite/DM 四个方言中
 */
export function wrapValueDefault(columnType: string, value: unknown): string | number | boolean | null {
    if (value == null) {
        return 'NULL';
    }
    const lower = columnType.toLowerCase();
    // 数值类型直接返回
    if (
        /^(tinyint|smallint|mediumint|int|integer|bigint|float|double|decimal|numeric|real|number|money|bit|serial|bigserial|smallserial|largeserial|uint|int8|int16|int32|int64|float32|float64|bool|boolean)$/i.test(
            lower
        )
    ) {
        return value as number;
    }
    // 布尔类型
    if (/^bool/i.test(lower)) {
        return value as boolean;
    }
    // 其余加单引号
    const strVal = String(value).replace(/[\r\n]/g, '\\n');
    return `'${strVal}'`;
}

/**
 * MSSQL 值包装（N'字符串' 前缀）
 */
export function wrapValueMssql(columnType: string, value: unknown): string | number | boolean | null {
    if (value == null) {
        return 'NULL';
    }
    const lower = columnType.toLowerCase();
    if (
        /^(tinyint|smallint|int|bigint|float|real|decimal|numeric|bit|money|smallmoney)$/i.test(lower)
    ) {
        return value as number;
    }
    if (/^bit$/i.test(lower)) {
        return value as boolean;
    }
    const strVal = String(value).replace(/[\r\n]/g, '\\n');
    return `N'${strVal}'`;
}

/**
 * Oracle 值包装（日期类型特殊处理）
 */
export function wrapValueOracle(columnType: string, value: unknown): string | number | boolean | null {
    if (value == null) {
        return 'NULL';
    }
    const upper = columnType.toUpperCase();
    if (['DATE', 'TIMESTAMP'].includes(upper)) {
        return `to_timestamp('${value}', 'yyyy-mm-dd hh24:mi:ss')`;
    }
    if (/^(NUMBER|INTEGER|FLOAT|BINARY_FLOAT|BINARY_DOUBLE)$/i.test(upper)) {
        return value as number;
    }
    const strVal = String(value).replace(/[\r\n]/g, '\\n');
    return `'${strVal}'`;
}
