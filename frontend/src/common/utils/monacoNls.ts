/**
 * monaco editor 语言包加载。
 * monaco 的 localize 在调用时查 globalThis._VSCODE_NLS_MESSAGES，但部分本地化常量
 * 在 monaco 模块加载期即已求值，因此本模块必须先于任何 monaco 模块被导入（见 main.ts 首行导入）。
 */
import 'monaco-editor/nls/lang/zh-cn.js';

import { getThemeConfig } from '@/common/utils/storage';

// 跟随应用当前语言：非中文环境移除中文表，回退 monaco 默认英文
if (!(getThemeConfig()?.globalI18n ?? 'zh-cn').startsWith('zh')) {
    delete (globalThis as Record<string, unknown>)._VSCODE_NLS_MESSAGES;
    delete (globalThis as Record<string, unknown>)._VSCODE_NLS_LANGUAGE;
}
