import type * as monaco from 'monaco-editor';

/**
 * monaco 编辑器的对外类型契约
 *
 * 单独成文件而非挂在组件上：这些类型会被弹窗组件、异步封装与各处调用方引用，
 * 放在组件文件里会让「只想拿类型」的模块被迫静态引入 monaco。
 */
export type MonacoEditorDialogProps = {
    content: string;
    title: string;
    language: string;
    height?: string;
    width?: string;
    options?: monaco.editor.IStandaloneEditorConstructionOptions; // 可选项,如字体大小等
    canChangeLang?: boolean; // 是否可以切换语言
    showConfirmButton?: boolean;
    confirmFn?: (value: string) => void; // 点击确认的回调函数,入参editor value
    closeFn?: () => void; // 点击取消 或 关闭弹窗的回调函数
    useDrawer?: boolean; // 是否使用drawer而不是dialog,默认false
    drawerSize?: string | number; // drawer尺寸,默认'50%'
};

/**
 * monaco 编辑器组件对外暴露的能力
 *
 * 编辑器按需加载后，使用方的 ref 在实例就绪前恒为 null，且异步组件无法再用
 * `InstanceType<typeof MonacoEditor>` 推导实例类型，故在此显式声明契约：
 * 由 MonacoEditor.vue 的 defineExpose 反向校验，两侧不会漂移。
 */
export type MonacoEditorExpose = {
    getEditor(): monaco.editor.IStandaloneCodeEditor;
    format(): void;
    focus(): void;
};
