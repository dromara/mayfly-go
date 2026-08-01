import { defineStore } from 'pinia';
import { computed, reactive } from 'vue';

export interface FileClipboard {
    /** 待复制/移动的文件路径列表 */
    paths: string[];
    /** 操作类型: cp | mv */
    type: 'cp' | 'mv';
    /** 来源路径 */
    fromPath: string;
}

/**
 * 机器文件管理模块级 Store
 *
 * 管理跨组件/跨实例共享的文件管理状态：
 * - 文件剪贴板：复制/移动操作可跨文件浏览器 tab 粘贴
 * - 路径历史：记录各文件浏览器实例的当前路径
 *
 * 使用动态 id 支持多实例隔离（如需）。
 */
export const useMachineFileStore = (id: string = 'machineFileStore') =>
    defineStore(id, () => {
        /** 文件剪贴板（跨文件浏览器实例共享） */
        const clipboard = reactive<FileClipboard>({
            paths: [],
            type: 'cp',
            fromPath: '',
        });

        /** 当前文件浏览路径（按 tabKey 索引） */
        const pathHistory = reactive<Record<string, string>>({});

        /** 剪贴板是否有内容 */
        const hasClipboard = computed(() => clipboard.paths.length > 0);
        /** 是否为复制模式 */
        const isCopyMode = computed(() => clipboard.type === 'cp');

        /** 设置剪贴板内容 */
        const setClipboard = (paths: string[], type: 'cp' | 'mv', fromPath: string) => {
            clipboard.paths = paths;
            clipboard.type = type;
            clipboard.fromPath = fromPath;
        };

        /** 追加路径到剪贴板（去重） */
        const addToClipboard = (paths: string[], type: 'cp' | 'mv', fromPath: string) => {
            for (const p of paths) {
                if (!clipboard.paths.includes(p)) {
                    clipboard.paths.push(p);
                }
            }
            clipboard.type = type;
            clipboard.fromPath = fromPath;
        };

        /** 清空剪贴板 */
        const clearClipboard = () => {
            clipboard.paths = [];
            clipboard.type = 'cp';
            clipboard.fromPath = '';
        };

        /** 记录文件浏览器当前路径 */
        const setPath = (tabKey: string, path: string) => {
            pathHistory[tabKey] = path;
        };

        /** 获取文件浏览器路径 */
        const getPath = (tabKey: string): string => {
            return pathHistory[tabKey] ?? '/';
        };

        return {
            clipboard,
            pathHistory,
            hasClipboard,
            isCopyMode,
            setClipboard,
            addToClipboard,
            clearClipboard,
            setPath,
            getPath,
        };
    })();
