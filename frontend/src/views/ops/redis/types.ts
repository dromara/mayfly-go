/**
 * Redis 模块类型定义
 *
 * 数据面类型（视角描述符、成员、分页）与后端 internal/redis/domain/entity/keyview.go 一一对应：
 * 表格列、能力位、表单结构、操作清单全部来自后端描述符，新增数据类型时前端不需要新增类型
 */
import type { AutoFormJsonSchema } from '@/components/auto-form';
import type { BaseModel, PageParam } from '@/types/common';

/** Redis 实体 (对应 entity.Redis) */
export interface Redis extends BaseModel {
    id: number;
    code: string;
    name: string;
    host: string;
    port?: number;
    mode: string;
    db: string;
    sshTunnelMachineId: number;
    remark: string;
}

/** Redis 列表查询参数 (对应后端 entity.RedisQuery) */
export interface RedisListParam extends PageParam {
    tagPath?: string;
    id?: number;
}

/** Redis 保存表单 */
export interface RedisSaveForm {
    id?: number | null;
    code?: string;
    tagCodePaths?: string[];
    name?: string | null;
    mode?: string;
    host?: string;
    username?: string | null;
    password?: string | null;
    redisNodePassword?: string | null;
    db?: string;
    remark?: string;
    sshTunnelMachineId?: number;
}

/** Redis 信息 */
export interface RedisInfo {
    [key: string]: string;
}

/** Redis scan 结果 (对应后端 vo.Keys) */
export interface RedisScanRes {
    /** 各节点 scan 游标 */
    cursor: Record<string, number>;
    keys: string[];
    dbSize: number;
    /** 本批 key 的类型/过期摘要，随 SCAN 批次一并返回（对应 vo.Keys.summaries）；旧后端未返回时按空处理 */
    summaries?: RedisKeySummary[];
}

/** key 列表批量摘要 (对应 entity.KeySummary) */
export interface RedisKeySummary {
    key: string;
    type: string;
    /** 剩余秒数，-1 永久，-2 不存在 */
    ttl: number;
}

/**
 * 成员表单结构：直接复用 AutoForm 的 v1 JSON Schema 类型，避免在模块内再造一份平行定义
 * （后端 ViewDescriptor.Form 下发的就是该结构）
 */
export type RedisFormSchema = AutoFormJsonSchema;

/** 成员表格列，field 对应成员字段名或 extra 的 key */
export interface RedisViewColumn {
    field: string;
    label: string;
    width: number;
    /** text 文本 | number 数字 | code 代码/JSON | tag 标签 | time 毫秒时间戳 | ttl 剩余秒数 */
    value: string;
    sortable: boolean;
}

/** 视角扩展操作 */
export interface RedisViewOp {
    name: string;
    label: string;
    form?: RedisFormSchema;
    write: boolean;
}

/** 视角能力位：前端按位显隐入口，不按类型名写分支 */
export interface RedisViewCaps {
    create: boolean;
    update: boolean;
    delete: boolean;
    batchDelete: boolean;
    keyword: boolean;
    rankPaging: boolean;
    cursorPaging: boolean;
    ops: boolean;
}

/** 数据视角描述符 (对应 entity.ViewDescriptor) */
export interface RedisViewDescriptor {
    view: string;
    label: string;
    types: string[];
    default: boolean;
    /** table 多行成员表格 | value 单值面板 */
    layout: string;
    caps: RedisViewCaps;
    columns: RedisViewColumn[];
    form?: RedisFormSchema;
    updateForm?: RedisFormSchema;
    ops: RedisViewOp[];
    /** 命令控制台的快捷命令模板，{key} 占位符在渲染时换成当前 key 名 */
    consoleHints: string[];
    /** 本视角「读内容」等价的命令名：面板读值按它过触发策略，「申请查看」也按它拼命令（三处同口径） */
    readCmd: string;
}

/** 实例命令目录条目 (对应 entity.CommandSpec)，命令控制台的输入提示与执行前确认依据 */
export interface RedisCommandSpec {
    name: string;
    /** 参数个数，负数表示「至少 |arity| 个」 */
    arity: number;
    flags: string[];
    /** 第一个键参数位置（1 为命令名后的第一个参数），0 表示无键参数 */
    firstKey: number;
    /** 最后一个键参数位置，-1 表示直到末尾 */
    lastKey: number;
    step: number;
    /** 执行前需要二次确认，与后端高危命令判定同源 */
    needConfirm: boolean;
}

/** key 元信息 (对应 entity.KeyMeta) */
export interface RedisKeyMeta {
    key: string;
    type: string;
    view: string;
    views: { view: string; label: string }[];
    encoding: string;
    /** 剩余秒数，-1 永久 */
    ttl: number;
    memuse: number;
    size: number;
    caps: RedisViewCaps;
    exists: boolean;
}

/** 一行成员数据 (对应 entity.Member) */
export interface RedisKeyMember {
    index: number;
    field: string;
    value: string;
    score: number;
    id: string;
    extra?: Record<string, string>;
}

/** 成员分页 (对应 entity.MemberPage) */
export interface RedisMemberPage {
    total: number;
    /** 空串表示游标已到末尾 */
    cursor: string;
    members: RedisKeyMember[];
}

/** 成员读写请求 (对应 entity.MemberWrite) */
/** 写操作共有的「仅提醒」确认位（读操作不涉及处置，故不放 RedisTargetParam 基类） */
export interface RedisWarnAckParam {
    /** 操作者已选「直接执行」；首次请求不带，此时后端不执行而是返回确认码让界面去问 */
    ackWarn?: boolean;
}

export interface RedisMemberWriteForm extends RedisTargetParam, RedisWarnAckParam {
    key: string;
    view?: string;
    op: string;
    member?: RedisKeyMember | null;
    members?: RedisKeyMember[];
    args?: Record<string, string>;
    ttl?: number;
}

/** 视角操作请求 (对应 entity.OpRequest) */
export interface RedisViewOpForm extends RedisTargetParam, RedisWarnAckParam {
    key: string;
    view?: string;
    op: string;
    args?: Record<string, string>;
}

/** 数据面接口的公共定位参数：实例 id 与库号会填进 url 路径 */
export interface RedisTargetParam {
    id: number;
    db: number;
}

/** key 级定位参数 */
export interface RedisKeyTargetForm extends RedisTargetParam {
    key: string;
    view?: string;
}

/** 批量 key 参数 */
export interface RedisKeysForm extends RedisTargetParam, RedisWarnAckParam {
    keys: string[];
}

/** 成员分页查询参数 */
export interface RedisMemberQueryForm extends RedisTargetParam, RedisWarnAckParam {
    key: string;
    view?: string;
    cursor?: string;
    offset?: number;
    size?: number;
    keyword?: string;
}

/** key 重命名 / 复制参数 */
export interface RedisKeyRenameForm extends RedisTargetParam, RedisWarnAckParam {
    key: string;
    newKey: string;
    /** 目标库，缺省表示当前库 */
    targetDb?: number;
    replace?: boolean;
}

/** key TTL 参数 */
export interface RedisKeyTtlForm extends RedisTargetParam, RedisWarnAckParam {
    key: string;
    ttl: number;
}
