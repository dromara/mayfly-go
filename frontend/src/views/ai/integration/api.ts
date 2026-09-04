/**
 * 插件管理 API 封装（统一插件实例 + 技能，走统一 Api 封装）
 */
import Api from '@/common/Api';
import type { Skill, SkillResource, McpToolInfo, SkillSaveReq, PluginType } from './types';

/** 插件实例（t_ai_plugin_instance，统一插件视图唯一事实源） */
export interface PluginInstance {
    id: string;
    /** 插件类型：skill / mcp / ...（后端类型注册表） */
    pluginType: PluginType;
    code: string;
    name: string;
    description: string;
    /** 类型化配置（对象，schema 由 pluginType 约定：skill → {skillCode}，mcp → {url, headers, timeoutSec}） */
    config: Record<string, any>;
    /** 启停：1 启用 / 0 停用（运行时总开关） */
    enabled: number;
    /** 实例健康状态：0 未知 / 1 健康 / 2 异常 */
    status: number;
    /** 以下仅 skill 类型填充（引用技能的展示字段，实例 id 与技能 id 不同） */
    skillId?: string;
    skillStatus?: string; // draft / published
    skillVersion?: string;
}

/** 插件实例保存请求体（config 为类型化配置对象） */
export interface InstanceSaveReq {
    pluginType: PluginType;
    code: string;
    name?: string;
    description?: string;
    config: Record<string, any>;
}

/** 插件类型元信息（GET /plugin/types，类型选择器/过滤下拉数据源） */
export interface PluginTypeInfo {
    code: PluginType;
}

/** 插件实例列表查询参数 */
export interface PluginListParams {
    pageNum?: number;
    pageSize?: number;
    /** 匹配 name/code/description */
    keyword?: string;
    /** skill / mcp，空查询全部 */
    type?: string;
}

/** 资源 upsert 请求体（body 含 path + content 即 upsert） */
export interface SkillResourceReq {
    path: string;
    content: string;
    size: number;
}

export const pluginApi = {
    // ============== 插件类型（注册表下发） ==============
    listTypes: Api.newGet<PluginTypeInfo[]>('/ai/plugin/types'),
    // ============== 插件实例（统一插件视图，单表分页） ==============
    listInstances: Api.newGet<{ total: number; list: PluginInstance[] }, PluginListParams>('/ai/plugin/instances'),
    getInstance: Api.newGet<PluginInstance, { id: string }>('/ai/plugin/instances/{id}'),
    createInstance: Api.newPost<PluginInstance, InstanceSaveReq>('/ai/plugin/instances'),
    updateInstance: Api.newPut<void, InstanceSaveReq & { id: string }>('/ai/plugin/instances/{id}'),
    deleteInstance: Api.newDelete<void, { id: string }>('/ai/plugin/instances/{id}'),
    discoverInstance: Api.newPost<McpToolInfo[], { id: string }>('/ai/plugin/instances/{id}/discover'),
    toggleInstance: Api.newPost<void, { id: string; enabled: number }>('/ai/plugin/instances/{id}/enabled'),
    // ============== 技能（实例由后端随技能联动注册） ==============
    listSkills: Api.newGet<Skill[]>('/ai/plugin/skills'),
    getSkill: Api.newGet<Skill, { id: string }>('/ai/plugin/skills/{id}'),
    createSkill: Api.newPost<Skill, SkillSaveReq>('/ai/plugin/skills'),
    updateSkill: Api.newPut<void, SkillSaveReq & { id: string }>('/ai/plugin/skills/{id}'),
    deleteSkill: Api.newDelete<void, { id: string }>('/ai/plugin/skills/{id}'),
    getInstructions: Api.newGet<{ content: string }, { id: string }>('/ai/plugin/skills/{id}/instructions'),
    publishSkill: Api.newPost<void, { id: string }>('/ai/plugin/skills/{id}/publish'),
    unpublishSkill: Api.newPost<void, { id: string }>('/ai/plugin/skills/{id}/unpublish'),
    // zip 导入导出
    importSkillZip: Api.newPost<Skill, FormData>('/ai/plugin/skills/import'),
    // ============== 技能资源 ==============
    listResources: Api.newGet<SkillResource[], { id: string }>('/ai/plugin/skills/{id}/resources'),
    upsertResource: Api.newPost<SkillResource, SkillResourceReq & { id: string }>('/ai/plugin/skills/{id}/resources'),
    deleteResource: Api.newDelete<void, { id: string; path: string }>('/ai/plugin/skills/{id}/resources'),
};
