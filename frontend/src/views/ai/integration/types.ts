/**
 * 插件管理（统一插件实例 + 技能）类型定义
 */

/**
 * 插件类型（后端 PluginTypeHandler 注册表下发）。
 *
 * 开放为 string 而不是 'skill' | 'mcp' 封闭联合：后端注册表可以新增类型，
 * 前端把值域定死只会让新类型在卡片上被错标成默认那一种
 */
export type PluginType = string;

/**
 * 插件类型元数据（单一真源）。
 *
 * 此前同一事实在三处各写一份：PluginCard 模板里 8 处 `pluginType === 'skill' ? … : …`、
 * PluginManagement 的 typeMeta 表、handlePick 的 if/else。新增一种插件类型要改三个文件，
 * 漏改一处就出现「标签写「MCP 插件」但表单开了技能编辑器」这类不一致。
 *
 * capabilities 用能力位而不是类型名：模板按「有没有长连接/有没有版本」渲染，
 * 新类型只需在表里声明能力，卡片零改动
 */
export interface PluginTypeMeta {
    /** 展示名 i18n key；未登记类型回退后端 code */
    labelKey?: string;
    /** 类型选择器的副标题 i18n key */
    descKey?: string;
    /** 类型选择器的主色（CSS 变量） */
    accentColor: string;
    /** el-tag 语义色 */
    tagType: 'primary' | 'success' | 'info';
    /** 图标名（由 pluginTypeIcons 映射到实际组件，本层不依赖 UI 库） */
    icon: 'skill' | 'mcp';
    /** 新建/编辑走哪个表单 */
    form: 'skill' | 'mcp';
    capabilities: {
        /** 展示实例健康与启停（有长连接才有健康态） */
        health: boolean;
        /** 展示发布/草稿状态与版本号（技能才有版本） */
        versioning: boolean;
        /** 展示连接测试与工具查看 */
        connectivity: boolean;
    };
}

export const PLUGIN_TYPE_META: Readonly<Record<string, PluginTypeMeta>> = {
    skill: {
        labelKey: 'ai.integration.skillPlugin',
        descKey: 'ai.integration.skillPluginDesc',
        accentColor: 'var(--el-color-primary)',
        tagType: 'primary',
        icon: 'skill',
        form: 'skill',
        capabilities: { health: false, versioning: true, connectivity: false },
    },
    mcp: {
        labelKey: 'ai.integration.mcpPlugin',
        descKey: 'ai.integration.mcpPluginDesc',
        accentColor: 'var(--el-color-success)',
        tagType: 'success',
        icon: 'mcp',
        form: 'mcp',
        capabilities: { health: true, versioning: false, connectivity: true },
    },
};

/**
 * 未登记类型的兜底：不猜它是哪一种，给中性色 + 只读展示（无健康、无版本、无连接测试）。
 *
 * 表单也不自动分发：新类型的编辑入口需要显式登记，不能把未知类型当 MCP 开表单（旧实现就是这么错标的）
 */
export const FALLBACK_PLUGIN_TYPE_META: PluginTypeMeta = {
    accentColor: 'var(--el-color-info)',
    tagType: 'info',
    icon: 'mcp',
    form: 'mcp',
    capabilities: { health: false, versioning: false, connectivity: false },
};

export function pluginTypeMeta(code: string): PluginTypeMeta {
    return PLUGIN_TYPE_META[code] ?? FALLBACK_PLUGIN_TYPE_META;
}

/** 类型展示名：未登记类型直接显示后端 code（不伪装成已知类型的文案） */
export function pluginTypeLabel(code: string, translate: (key: string) => string): string {
    const meta = PLUGIN_TYPE_META[code];
    return meta?.labelKey ? translate(meta.labelKey) : code;
}

/** 实例健康状态（discover 时回写） */
export enum InstanceStatus {
    Unknown = 0,
    Healthy = 1,
    Error = 2,
}

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

/** MCP 实例内联配置（t_ai_plugin_instance.config，pluginType=mcp） */
export interface McpInstanceConfig {
    url: string;
    headers: string; // JSON 字符串
    timeoutSec: number;
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
