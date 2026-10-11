import { findChildByLocateCode, getContributor, getRootContributor } from './tree/registry';
import { ERROR_KIND, LOADING_KIND, type LocateAccess, type LocateOutcome } from './tree/types';

/**
 * codePath 定位解析（资产域知识收口处）
 *
 * 后端标签资源以 codePath 标识自身：`tag1/tag2/type|code/`（协议见 server/internal/tag/domain/entity/tag_tree.go）。
 * 本模块把这样一条标识解析成资源树中的节点 key，供通用树容器展开定位；容器只消费 LocateResolver 契约，
 * 不认识 codePath / 各资源 code 字段等任何资产协议。
 *
 * 定位能力由各贡献者的 `locateCode` 声明提供（见 tree/registry.ts）：
 * 新增资产类型只需在自身贡献者声明 locateCode，本模块与容器均无需改动（开闭原则）。
 */

/** codePath 段分隔符（与后端 CodePathSeparator 对齐） */
const CODE_PATH_SEPARATOR = '/';
/** 资源段内「类型|code」分隔符（与后端 CodePathResourceSeparator 对齐） */
const RESOURCE_SEPARATOR = '|';

/** 标签资源类型值（codePath 资源段的 type） */
export interface CodePathSection {
    type: number;
    code: string;
}

/** codePath 解析结果：标签路径 + 自顶向下的有序资源段 */
export interface CodePathSections {
    /** 标签路径段拼合结果，形如 `tag1/tag2/`；纯资源根下为空串 */
    tagPath: string;
    resources: CodePathSection[];
}

/**
 * 解析 codePath 为标签路径与有序资源段。
 * 标签段不含 `|`；资源段为 `类型|code`
 */
export function parseCodePath(codePath: string): CodePathSections {
    let tagPath = '';
    const resources: CodePathSection[] = [];
    for (const section of codePath.split(CODE_PATH_SEPARATOR)) {
        if (!section) {
            continue;
        }
        const sepIndex = section.indexOf(RESOURCE_SEPARATOR);
        if (sepIndex < 0) {
            tagPath += section + CODE_PATH_SEPARATOR;
            continue;
        }
        resources.push({ type: Number(section.slice(0, sepIndex)), code: section.slice(sepIndex + 1) });
    }
    return { tagPath, resources };
}

/** 内置占位行（加载中/错误）不是资源节点，不参与定位诊断 */
const isPlaceholderKind = (kind: string) => kind === LOADING_KIND || kind === ERROR_KIND;

/**
 * DEV 诊断：资源段未匹配到子节点时，区分两种成因——
 * 子节点都已声明 locateCode 却无人命中 → 该段确实不是树层级（正常，如数据库实例下的「账号」段）；
 * 存在未声明 locateCode 的子节点 → 可能是新资产漏声明定位契约，显式告警避免静默降级
 */
function warnUnmatchedSection(parentKey: string, section: CodePathSection, access: LocateAccess) {
    const children = (access.getNode(parentKey)?.children ?? []).filter((c) => !isPlaceholderKind(c.kind));
    const missing = [...new Set(children.filter((c) => !getContributor(c.kind)?.locateCode).map((c) => c.kind))];
    if (children.length && missing.length) {
        console.warn(
            `[locate] codePath 段 ${section.type}|${section.code} 未匹配到子节点，父节点 ${parentKey} 下这些 kind 未声明 locateCode：${missing.join(', ')}（新增资源需在贡献者声明 locateCode 才能被跳转定位）`
        );
    }
}

/** 纯标签定位：标签节点 key 即其 codePath（见 resource.ts flatten），根已加载仍缺失即判 missing */
function resolveTagNode(tagPath: string, access: LocateAccess): LocateOutcome {
    if (access.getNode(tagPath)) {
        return { status: 'resolved', key: tagPath };
    }
    return access.rootLoaded ? { status: 'missing' } : { status: 'pending' };
}

/** 取标签/资源节点已水合的子节点（必要时展开该节点） */
async function loadedChildren(key: string, access: LocateAccess) {
    const node = access.getNode(key);
    if (node && !node.loaded) {
        await access.expandNode(key);
    }
    return access.getNode(key);
}

/**
 * codePath → 树节点 key 解析器（实现容器注入的 LocateResolver 契约）。
 *
 * 层级依据：标签节点（key 即其 codePath）→ 各级资源节点（按 locateCode 匹配）。
 * 标签节点下不再有一层「机器/数据库」类型分组节点，但定位仍从标签分支往下走：
 * 同一资源挂在多个标签下时，每条 codePath 沿自己的标签分支展开，副本歧义在根上就消了。
 * codePath 段数可能多于树层级（如数据库为 实例→账号→库名，树中无账号节点），未命中的段直接跳过。
 */
export async function resolveCodePathLocate(codePath: string, access: LocateAccess): Promise<LocateOutcome> {
    const { tagPath, resources } = parseCodePath(codePath);
    if (!resources.length) {
        return tagPath ? resolveTagNode(tagPath, access) : { status: 'missing' };
    }

    const [root, ...deeper] = resources;
    // 标签节点就是资源的直接父层：标签不存在时区分「根未加载可重试」与「确实无权限/已删」
    if (!access.getNode(tagPath)) {
        return access.rootLoaded ? { status: 'missing' } : { status: 'pending' };
    }

    const tag = await loadedChildren(tagPath, access);
    // 首段自带资源类型：反查根贡献者的 kind 收窄匹配。不同资产的 code 各有一套命名空间，
    // 同一标签下跨类型重名时只按 code 会挑错节点（旧实现靠分组节点天然隔离，去掉那层后必须显式收窄）
    const first = tag ? findChildByLocateCode(tag, root.code, getRootContributor(root.type)?.kind) : undefined;
    if (!first) {
        return { status: 'missing' };
    }

    let currentKey = first.key;
    for (const section of deeper) {
        const current = await loadedChildren(currentKey, access);
        const child = findChildByLocateCode(current, section.code);
        if (child) {
            currentKey = child.key;
        } else if (import.meta.env.DEV) {
            warnUnmatchedSection(currentKey, section, access);
        }
    }

    return access.getNode(currentKey) ? { status: 'resolved', key: currentKey } : { status: 'missing' };
}
