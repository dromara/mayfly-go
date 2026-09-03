/**
 * 插件管理 API 封装（技能 + MCP 服务器，走统一 Api 封装）
 */
import Api from '@/common/Api';
import type { Skill, SkillResource, McpServer, McpToolInfo, SkillSaveReq } from './types';

/** MCP 服务器保存请求体 */
export interface McpServerSaveReq {
    code: string;
    name: string;
    description: string;
    url: string;
    headers: string;
    timeoutSec: number;
    enabled: number;
}

/** 资源 upsert 请求体（body 含 path + content 即 upsert） */
export interface SkillResourceReq {
    path: string;
    content: string;
    size: number;
}

export const pluginApi = {
    // ============== 技能 ==============
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
    // ============== MCP 服务器 ==============
    listMcpServers: Api.newGet<McpServer[]>('/ai/plugin/mcp/servers'),
    getMcpServer: Api.newGet<McpServer, { id: string }>('/ai/plugin/mcp/servers/{id}'),
    createMcpServer: Api.newPost<void, McpServerSaveReq>('/ai/plugin/mcp/servers'),
    updateMcpServer: Api.newPut<void, McpServerSaveReq & { id: string }>('/ai/plugin/mcp/servers/{id}'),
    deleteMcpServer: Api.newDelete<void, { id: string }>('/ai/plugin/mcp/servers/{id}'),
    discoverMcpTools: Api.newPost<McpToolInfo[], { id: string }>('/ai/plugin/mcp/servers/{id}/discover'),
};
