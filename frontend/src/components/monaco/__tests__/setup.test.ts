import { describe, expect, it } from 'vitest';
import * as monaco from '../setup';

/**
 * 装配入口守卫
 *
 * setup.ts 的目标是在保住编辑器功能的前提下，把 monaco 从「包入口全量注册」裁剪为最小装配，
 * 因此这里校验两件事：
 * 1. 语言集合必须恰好是项目声明支持的那些——少了意味着某个页面的编辑器失去着色/语言服务，
 *    多了（比如有人改回 `from 'monaco-editor'`）意味着整包 monaco 又回到产物里；
 * 2. 编辑器功能确实完成了注册（以真实创建编辑器后的 action 集合为证据）。
 */

/** 项目实际使用的语言：MonacoEditor.vue 的 languageArr + EXT_LANGUAGE_MAP + getFileType + TmplTypeEnum + 各方言编辑器 */
const EXPECTED_LANGUAGES = ['css', 'dockerfile', 'html', 'java', 'javascript', 'json', 'markdown', 'plaintext', 'python', 'shell', 'sql', 'xml', 'yaml'];

/** 各功能贡献的代表 action：查找、联想补全、注释、括号匹配、折叠 */
const FEATURE_ACTIONS = ['actions.find', 'editor.action.triggerSuggest', 'editor.action.commentLine', 'editor.action.jumpToBracket', 'editor.toggleFold'];

const supportedActions = (language: string) => {
    const container = document.createElement('div');
    document.body.appendChild(container);
    const editor = monaco.editor.create(container, { value: 'select 1', language, automaticLayout: false });
    const ids = editor.getSupportedActions().map((action) => action.id);
    editor.dispose();
    container.remove();
    return ids;
};

describe('monaco 装配入口', () => {
    it('只注册项目用到的语言', () => {
        expect(monaco.languages.getLanguages().map((l) => l.id).sort()).toEqual(EXPECTED_LANGUAGES);
    });

    it('导出编辑器 API 与快捷键常量', () => {
        expect(typeof monaco.editor.create).toBe('function');
        expect(typeof monaco.languages.registerCompletionItemProvider).toBe('function');
        expect(monaco.KeyMod.CtrlCmd).toBeGreaterThan(0);
        expect(monaco.KeyCode.KeyR).toBeGreaterThan(0);
    });

    it('编辑器功能已完成注册', () => {
        const ids = supportedActions('sql');
        for (const action of FEATURE_ACTIONS) {
            expect(ids).toContain(action);
        }
    });
});
