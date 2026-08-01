/**
 * Sys 模块类型定义
 * 对应后端: sys/domain/entity/*
 */
import type { BaseModel, PageParam } from '@/types/common';

// ==================== Entity ====================

/** 账户实体 (对应 entity.Account) */
export interface Account extends BaseModel {
    id: number;
    name: string;
    username: string;
    mobile: string;
    email: string;
    status: number;
    lastLoginTime?: string;
    lastLoginIp: string;
    otpSecret?: string;
    extra?: Record<string, unknown>;
}

/** 系统配置实体 (对应 entity.Config) */
export interface SysConfig extends BaseModel {
    id: number;
    name: string;
    key: string;
    params: string;
    value: string;
    remark: string;
    permission: string;
}

/** 资源 meta 信息（meta 字段 JSON 字符串解析后的结构，对应前端路由 meta） */
export interface ResourceMeta {
    routeName: string;
    icon: string;
    redirect?: string;
    component?: string;
    isKeepAlive: boolean;
    isHide: boolean;
    isAffix: boolean;
    linkType: number;
    link?: string;
}

/** 系统资源实体 (对应 entity.Resource) */
export interface SysResource extends BaseModel {
    id: number;
    pid: number;
    ui_path: string;
    type: number;
    status: number;
    code: string;
    name: string;
    weight: number;
    /** meta 信息，接口返回 JSON 字符串，前端解析后为 ResourceMeta 对象 */
    meta: string | ResourceMeta;
    /** 子资源节点（树形结构，叶子节点为 null） */
    children?: SysResource[] | null;
}

/** 账号列表查询参数 (对应后端 entity.AccountQuery) */
export interface AccountListParam extends PageParam {
    username?: string;
    name?: string;
    ids?: string | number | number[];
}

/** 系统角色实体 (对应 entity.Role) */
export interface SysRole extends BaseModel {
    id: number;
    status: number;
    name: string;
    remark: string;
    code: string;
    type: number;
}

/** 角色资源关联 (对应 entity.RoleResource) */
export interface RoleResource {
    id: number;
    roleId: number;
    resourceId: number;
    createTime: string;
    creator: string;
}

/** 账号角色关联 (对应 entity.AccountRole) */
export interface AccountRole {
    id: number;
    accountId: number;
    roleId: number;
    createTime: string;
    creator: string;
}

/** 系统操作日志 (对应 entity.SysLog) */
export interface SysLog {
    id: number;
    type: number;
    description: string;
    reqParam: string;
    resp: string;
    extra: string;
    createTime: string;
    creator: string;
}

// ==================== VO ====================

/** 账户详情 (含角色信息) */
export interface AccountDetail extends Account {
    roles?: SysRole[];
    resources?: SysResource[];
}

/** 登录返回结果 */
export interface LoginResult {
    token: string;
    refresh_token?: string;
    username: string;
    name: string;
    lastLoginTime?: string;
    lastLoginIp?: string;
    otp?: number;
    otpUrl?: string;
    action?: string;
    isFirstOauth2Login?: boolean;
}

/** 权限信息 */
export interface PermissionInfo {
    accountId: number;
    username: string;
    name: string;
    resources: SysResource[];
    codes: string[];
    permissions?: string[];
    menus?: unknown[];
}

/** OAuth2 配置 */
export interface OAuth2Config {
    clientId: string;
    clientSecret: string;
    authUrl: string;
    tokenUrl: string;
    redirectUrl: string;
    scopes: string;
}
