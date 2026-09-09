/**
 * 插件管理前端工具函数（ResourcePanel buildTree / languageFromPath / file-type-icon）
 */
import type { SkillResource, LocalResource, TreeNode } from './types';

/** 扁平路径列表转目录+文件树（目录节点按 path 聚合），buildTree */
export function buildTree(items: (SkillResource | LocalResource)[]): TreeNode[] {
    const root: TreeNode[] = [];
    const dirMap = new Map<string, TreeNode>();

    for (const item of items) {
        const parts = item.path.split('/');
        let currentPath = '';

        for (let i = 0; i < parts.length; i++) {
            const part = parts[i];
            const parentPath = currentPath;
            currentPath = currentPath ? `${currentPath}/${part}` : part;
            const isLast = i === parts.length - 1;

            if (isLast) {
                const fileNode: TreeNode = { name: part, path: currentPath, isDir: false, children: [] };
                const parent = parentPath ? dirMap.get(parentPath) : null;
                if (parent) parent.children.push(fileNode);
                else root.push(fileNode);
            } else if (!dirMap.has(currentPath)) {
                const dirNode: TreeNode = { name: part, path: currentPath, isDir: true, children: [] };
                dirMap.set(currentPath, dirNode);
                const parent = parentPath ? dirMap.get(parentPath) : null;
                if (parent) parent.children.push(dirNode);
                else root.push(dirNode);
            }
        }
    }
    return root;
}

/** 扩展名 → Monaco language 映射（EXT_LANGUAGE_MAP，仅保留 mayfly-go Monaco 已注册语言） */
const EXT_LANGUAGE_MAP: Record<string, string> = {
    md: 'markdown',
    markdown: 'markdown',
    json: 'json',
    yaml: 'yaml',
    yml: 'yaml',
    xml: 'xml',
    html: 'html',
    css: 'css',
    py: 'python',
    java: 'java',
    sh: 'shell',
    bash: 'shell',
    zsh: 'shell',
    sql: 'sql',
    txt: 'text',
};

/** 按文件扩展名推断 Monaco language */
export function languageFromPath(path: string): string {
    const ext = path.split('.').pop()?.toLowerCase() ?? '';
    return EXT_LANGUAGE_MAP[ext] ?? 'text';
}

/** 文件类型 → 展示色（编辑器行业惯例配色，集中维护） */
export function fileTypeColor(fileName: string): string {
    if (fileName.endsWith('.md')) return 'var(--el-color-primary)';
    if (/\.(ts|js|tsx|jsx)$/.test(fileName)) return '#d4a017';
    if (fileName.endsWith('.json')) return 'var(--el-color-success)';
    if (/\.(sh|py)$/.test(fileName)) return 'var(--el-color-purple, #9d5bd2)';
    return 'var(--el-text-color-secondary)';
}

/** 收集路径的全部父目录（新建文件后展开父目录用） */
export function parentDirs(path: string): string[] {
    const parts = path.split('/');
    const dirs: string[] = [];
    let p = '';
    for (let i = 0; i < parts.length - 1; i++) {
        p = p ? `${p}/${parts[i]}` : parts[i];
        dirs.push(p);
    }
    return dirs;
}
