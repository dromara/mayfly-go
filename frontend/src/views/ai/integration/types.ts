/**
 * 插件管理（技能 + MCP 服务器）类型定义
 */

/** 插件类型 */
export type PluginType = 'skill' | 'mcp';

/** 技能实体（t_ai_skill） */
export interface Skill {
    id: string;
    code: string;
    name: string;
    description: string;
    allowedTools: string;
    version: string;
    source: string; // builtin / imported / custom
    status: string; // draft / published
}

/** 技能资源（t_ai_skill_resource） */
export interface SkillResource {
    id: string;
    skillId: string;
    path: string;
    content: string;
    size: number;
}

/** MCP 服务器实体（t_ai_mcp_server，仅 HTTP 传输） */
export interface McpServer {
    id: string;
    code: string;
    name: string;
    description: string;
    url: string;
    headers: string; // JSON 字符串
    timeoutSec: number;
    enabled: number; // 1=启用
}

/** MCP 发现到的工具元信息 */
export interface McpToolInfo {
    name: string;
    description: string;
    inputSchema: string;
}

/** 技能保存请求（instructions 为 SKILL.md 正文） */
export interface SkillSaveReq {
    code: string;
    description: string;
    allowedTools: string;
    instructions: string;
}

/** 本地暂存资源（新建态技能未落库时的资源暂存） */
export interface LocalResource {
    _tempId: string;
    path: string;
    content: string;
}

/** 资源树节点 */
export interface TreeNode {
    name: string;
    path: string;
    isDir: boolean;
    children: TreeNode[];
}
