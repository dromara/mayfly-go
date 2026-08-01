/**
 * Tag 模块类型定义
 * 对应后端: tag/domain/entity/*
 */
import type { BaseModel, PageParam, RelateTags } from '@/types/common';

// ==================== Entity ====================

/** 标签树实体 (对应 entity.TagTree) */
export interface TagTree extends BaseModel {
    id: number;
    type: number;
    code: string;
    codePath: string;
    name: string;
    remark: string;
}

/** 资源操作日志 (对应 entity.ResourceOpLog) */
export interface ResourceOpLog {
    id: number;
    codePath: string;
    resourceCode: string;
    relateType: number;
    createTime: string;
    creator: string;
}

/** 资源标签查询参数 */
export interface ResourceTagQueryParam {
    resourceCode?: string;
    resourceType?: number;
}

/** 资源关联的标签信息 (对应 entity.ResourceTag) */
export interface ResourceTag {
    tagId: number;
    codePath: string;
}

/** 授权凭证查询参数 */
export interface AuthCertQueryParam extends PageParam {
    resourceCode?: string;
    resourceType?: number;
    type?: number;
}

/** 团队实体 (对应 entity.Team) */
export interface Team extends RelateTags, BaseModel {
    id: number;
    name: string;
    validityStartDate?: string;
    validityEndDate?: string;
    remark: string;
}

/** 团队成员实体 (对应 entity.TeamMember) */
export interface TeamMember {
    id: number;
    teamId: number;
    accountId: number;
    username: string;
    createTime: string;
    creator: string;
}

// ==================== VO ====================

/** 标签树节点 (用于展示) */
export interface TagTreeNode {
    id: number;
    type: number;
    code: string;
    codePath: string;
    name: string;
    remark: string;
    children?: TagTreeNode[];
}

/** 资源标签路径 */
export interface ResourceTagPath {
    codePath: string;
    tagPaths: string[];
}
