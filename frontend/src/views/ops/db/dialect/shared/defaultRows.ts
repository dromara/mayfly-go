/**
 * 默认审计字段模板
 *
 * 所有方言的 getDefaultRows 都返回相同的 7 列审计字段结构（id、creator_id、creator、
 * create_time、updator_id、updator、update_time），仅类型名和默认值有差异。
 * 本模块提供配置化生成，消除 ~400 行重复代码。
 */
import type { RowDefinition } from '../types';

export interface DefaultRowsConfig {
    /** ID 字段类型（如 bigint、bigserial、NUMBER） */
    idType: string;
    /** ID 字段长度（如 '20'，留空则不拼接） */
    idLength?: string;
    /** 引用 ID 字段类型（如 int8、bigint），用于 creator_id/updator_id 等外键字段 */
    refIdType: string;
    /** 引用 ID 字段长度 */
    refIdLength?: string;
    /** 字符串类型（如 varchar、VARCHAR2、nvarchar） */
    stringType: string;
    /** 字符串默认长度 */
    stringLength: string;
    /** 时间类型（如 datetime、timestamp、DATE） */
    datetimeType: string;
    /** 时间字段默认值（如 CURRENT_TIMESTAMP） */
    datetimeDefault: string;
    /** 字段名大小写风格 */
    nameCase: 'lower' | 'upper' | 'camel';
}

/** 字段名大小写转换 */
function applyCase(name: string, nameCase: DefaultRowsConfig['nameCase']): string {
    switch (nameCase) {
        case 'upper':
            return name.toUpperCase();
        case 'camel':
            return name.replace(/_([a-z])/g, (_, c) => c.toUpperCase());
        case 'lower':
        default:
            return name.toLowerCase();
    }
}

/**
 * 生成标准 7 列审计字段
 */
export function createDefaultRows(config: DefaultRowsConfig): RowDefinition[] {
    const n = (name: string) => applyCase(name, config.nameCase);
    const idLen = config.idLength || '';
    const refIdLen = config.refIdLength || '';
    const refId = config.refIdType;

    return [
        { name: n('id'), type: config.idType, length: idLen, numScale: '', value: '', notNull: true, pri: true, auto_increment: true, remark: '主键 ID' },
        { name: n('creator_id'), type: refId, length: refIdLen, numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '创建人 ID' },
        { name: n('creator'), type: config.stringType, length: config.stringLength, numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '创建人姓名' },
        { name: n('create_time'), type: config.datetimeType, length: '', numScale: '', value: config.datetimeDefault, notNull: true, pri: false, auto_increment: false, remark: '创建时间' },
        { name: n('updator_id'), type: refId, length: refIdLen, numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '修改人 ID' },
        { name: n('updator'), type: config.stringType, length: config.stringLength, numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '修改人姓名' },
        { name: n('update_time'), type: config.datetimeType, length: '', numScale: '', value: config.datetimeDefault, notNull: true, pri: false, auto_increment: false, remark: '修改时间' },
    ];
}

/** 预定义各方言的默认行配置 */
export const defaultRowsConfigs: Record<string, DefaultRowsConfig> = {
    mysql: {
        idType: 'bigint',
        idLength: '20',
        refIdType: 'bigint',
        refIdLength: '20',
        stringType: 'varchar',
        stringLength: '100',
        datetimeType: 'datetime',
        datetimeDefault: 'CURRENT_TIMESTAMP',
        nameCase: 'lower',
    },
    postgresql: {
        idType: 'bigserial',
        refIdType: 'int8',
        stringType: 'varchar',
        stringLength: '100',
        datetimeType: 'timestamp',
        datetimeDefault: 'CURRENT_TIMESTAMP',
        nameCase: 'lower',
    },
    oracle: {
        idType: 'NUMBER',
        refIdType: 'NUMBER',
        stringType: 'VARCHAR2',
        stringLength: '100',
        datetimeType: 'DATE',
        datetimeDefault: 'CURRENT_TIMESTAMP',
        nameCase: 'upper',
    },
    mssql: {
        idType: 'bigint',
        refIdType: 'bigint',
        refIdLength: '20',
        stringType: 'nvarchar',
        stringLength: '100',
        datetimeType: 'datetime2',
        datetimeDefault: 'CURRENT_TIMESTAMP',
        nameCase: 'lower',
    },
    sqlite: {
        idType: 'integer',
        refIdType: 'bigint',
        refIdLength: '20',
        stringType: 'varchar',
        stringLength: '100',
        datetimeType: 'datetime',
        datetimeDefault: 'CURRENT_TIMESTAMP',
        nameCase: 'lower',
    },
    clickhouse: {
        idType: 'UInt64',
        refIdType: 'UInt64',
        stringType: 'String',
        stringLength: '',
        datetimeType: 'DateTime',
        datetimeDefault: '',
        nameCase: 'lower',
    },
    dm: {
        idType: 'BIGINT',
        refIdType: 'BIGINT',
        stringType: 'VARCHAR',
        stringLength: '100',
        datetimeType: 'TIMESTAMP',
        datetimeDefault: 'SYSDATE',
        nameCase: 'upper',
    },
};
