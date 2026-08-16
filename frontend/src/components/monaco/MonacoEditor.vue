<template>
    <div class="monaco-editor-custom relative h-full">
        <div class="monaco-editor-content" ref="monacoTextareaRef" :style="{ height: height }"></div>
        <el-select v-if="canChangeMode" class="code-mode-select" v-model="languageMode" @change="changeLanguage" filterable size="small">
            <el-option v-for="mode in languageArr" :key="mode.value" :label="mode.label" :value="mode.value" />
        </el-select>
    </div>
</template>

<script lang="ts" setup>
import { watch, toRefs, reactive, onMounted, onBeforeUnmount, useTemplateRef, Ref } from 'vue';
import * as monaco from 'monaco-editor';
// 相关语言
import 'monaco-editor/languages/definitions/shell/register.js';
import 'monaco-editor/languages/definitions/yaml/register.js';
import 'monaco-editor/languages/definitions/dockerfile/register.js';
import 'monaco-editor/languages/definitions/javascript/register.js';
import 'monaco-editor/languages/definitions/html/register.js';
import 'monaco-editor/languages/definitions/css/register.js';
import 'monaco-editor/languages/definitions/python/register.js';
import 'monaco-editor/languages/definitions/markdown/register.js';
import 'monaco-editor/languages/definitions/java/register.js';
import 'monaco-editor/languages/definitions/sql/register.js';
import 'monaco-editor/language/json/monaco.contribution';
// 右键菜单
import 'monaco-editor/editor/contrib/contextmenu/browser/contextmenu.js';
import 'monaco-editor/editor/contrib/caretOperations/browser/caretOperations.js';
import 'monaco-editor/editor/contrib/clipboard//browser/clipboard.js';
import 'monaco-editor/editor/contrib/find/browser/findController.js';
import 'monaco-editor/editor/contrib/format//browser/formatActions.js';
// 提示
import 'monaco-editor/editor/contrib/suggest/browser/suggestController.js';
import 'monaco-editor/editor/contrib/suggest/browser/suggestInlineCompletions.js';
import { editor, languages } from 'monaco-editor';
import EditorWorker from 'monaco-editor/editor/editor.worker.js?worker';
import JsonWorker from 'monaco-editor/language/json/json.worker?worker';
import HtmlWorker from 'monaco-editor/language/html/html.worker?worker';
import SolarizedLight from './themes/Solarized-light.json';
import SolarizedDark from './themes/Solarized-dark.json';
import { language as shellLan } from 'monaco-editor/languages/definitions/shell/shell.js';

import { ElOption, ElSelect } from 'element-plus';

import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';

const { themeConfig } = storeToRefs(useThemeConfig());

const props = withDefaults(
    defineProps<{
        language?: string | null;
        height?: string;
        width?: string;
        canChangeMode?: boolean;
        options?: object;
    }>(),
    { language: null, height: '500px', width: 'auto', canChangeMode: false, options: () => ({}) }
);

const modelValue = defineModel<string | null | undefined>('modelValue', { required: true });

const languageArr = [
    {
        value: 'shell',
        label: 'Shell',
    },
    {
        value: 'json',
        label: 'JSON',
    },
    {
        value: 'yaml',
        label: 'Yaml',
    },
    {
        value: 'dockerfile',
        label: 'Dockerfile',
    },
    {
        value: 'html',
        label: 'Html',
    },
    {
        value: 'xml',
        label: 'Xml',
    },
    {
        value: 'python',
        label: 'Python',
    },
    {
        value: 'sql',
        label: 'SQL',
    },
    {
        value: 'css',
        label: 'CSS',
    },
    {
        value: 'javascript',
        label: 'Javascript',
    },
    {
        value: 'java',
        label: 'Java',
    },
    {
        value: 'markdown',
        label: 'Markdown',
    },
    {
        value: 'text',
        label: 'text',
    },
];

const defaultOptions = {
    language: 'shell',
    theme: 'SolarizedLight',
    automaticLayout: true, //自适应宽高布局
    foldingStrategy: 'indentation', //代码可分小段折叠
    folding: true,
    roundedSelection: false, // 禁用选择文本背景的圆角
    matchBrackets: 'near',
    linkedEditing: true,
    cursorBlinking: 'smooth', // 光标闪烁样式
    mouseWheelZoom: true, // 在按住Ctrl键的同时使用鼠标滚轮时，在编辑器中缩放字体
    overviewRulerBorder: false, // 不要滚动条的边框
    tabSize: 4, // tab 缩进长度
    // fontFamily: 'JetBrainsMono', // 字体 暂时不要设置，否则光标容易错位
    fontWeight: 'bold',
    // fontSize: 12,
    // letterSpacing: 1, 字符间距
    // quickSuggestions:false, // 禁用代码提示
    minimap: {
        enabled: false, // 不要小地图
    },
    renderLineHighlight: 'all',
    selectOnLineNumbers: false,
    readOnly: false,
    scrollBeyondLastLine: false,
    lineNumbers: 'on',
    lineNumbersMinChars: 3,
    fixedOverflowWidgets: true, // 使弹出层不被容器限制
} as editor.IStandaloneEditorConstructionOptions;

const monacoTextareaRef = useTemplateRef<HTMLElement>('monacoTextareaRef');

let monacoEditorIns: editor.IStandaloneCodeEditor = null!;
let completionItemProvider: monaco.IDisposable | null = null;

self.MonacoEnvironment = {
    getWorker(_: string, label: string) {
        if (label === 'json') {
            return new JsonWorker();
        }
        if (label === 'html') {
            return new HtmlWorker();
        }
        return new EditorWorker();
    },
};

const state = reactive({
    languageMode: 'shell',
});

const { languageMode } = toRefs(state);

onMounted(() => {
    state.languageMode = props.language ?? 'shell';
    initMonacoEditorIns();
    setEditorValue(modelValue.value ?? '');
    registerCompletionItemProvider();
});

onBeforeUnmount(() => {
    if (monacoEditorIns) {
        monacoEditorIns.dispose();
    }
    if (completionItemProvider) {
        completionItemProvider.dispose();
    }
});

watch(modelValue, (newValue: string | null | undefined) => {
    if (!monacoEditorIns.hasTextFocus()) {
        state.languageMode = props.language ?? 'shell';
        monacoEditorIns?.setValue(newValue ?? '');
    }
});

watch(
    () => props.language,
    (newValue: string | null) => {
        changeLanguage(newValue ?? 'shell');
    }
);

// 监听 themeConfig editorTheme配置文件的变化
watch(
    () => themeConfig.value.editorTheme,
    (val) => {
        monaco?.editor?.setTheme(val);
    }
);

const initMonacoEditorIns = () => {
    // options参数参考 https://microsoft.github.io/monaco-editor/api/interfaces/monaco.editor.IStandaloneEditorConstructionOptions.html#language
    // 初始化一些主题
    monaco.editor.defineTheme('SolarizedLight', SolarizedLight as editor.IStandaloneThemeData);
    monaco.editor.defineTheme('SolarizedDark', SolarizedDark as editor.IStandaloneThemeData);
    defaultOptions.language = state.languageMode;
    defaultOptions.theme = themeConfig.value.editorTheme;
    let options = { ...defaultOptions, ...(props.options as editor.IStandaloneEditorConstructionOptions) };
    if (!monacoTextareaRef.value) {
        return;
    }
    monacoEditorIns = monaco.editor.create(monacoTextareaRef.value, options);

    if (!options.readOnly) {
        // 监听内容改变,双向绑定
        monacoEditorIns.onDidChangeModelContent(() => {
            modelValue.value = monacoEditorIns.getModel()?.getValue() ?? '';
        });
    }
};

const changeLanguage = (value: string) => {
    // 获取当前的文档模型
    let oldModel = monacoEditorIns.getModel();
    if (!oldModel) {
        return;
    }
    // 创建一个新的文档模型
    let newModel = monaco.editor.createModel(oldModel.getValue(), value);
    // 设置成新的
    monacoEditorIns.setModel(newModel);
    // 销毁旧的模型
    if (oldModel) {
        oldModel.dispose();
    }

    registerCompletionItemProvider();
};

const setEditorValue = (value: string) => {
    if (value) {
        monacoEditorIns.getModel()?.setValue(value);
    }
};

/**
 * 注册联想补全提示
 */
const registerCompletionItemProvider = () => {
    if (completionItemProvider) {
        completionItemProvider.dispose();
    }
    if (state.languageMode == 'shell') {
        registerShell();
    }
};

const registerShell = () => {
    completionItemProvider = monaco.languages.registerCompletionItemProvider('shell', {
        provideCompletionItems: async () => {
            let suggestions: { label: string; kind: languages.CompletionItemKind; insertText: string }[] = [];
            shellLan.keywords.forEach((item: string) => {
                suggestions.push({
                    label: item,
                    kind: monaco.languages.CompletionItemKind.Keyword,
                    insertText: item,
                });
            });
            shellLan.builtins.forEach((item: string) => {
                suggestions.push({
                    label: item,
                    kind: monaco.languages.CompletionItemKind.Property,
                    insertText: item,
                });
            });
            return {
                suggestions: suggestions as languages.CompletionItem[],
            };
        },
    });
};

const format = () => {
    // 触发自动格式化;
    monacoEditorIns.trigger('', 'editor.action.formatDocument', '');
};

const focus = () => {
    monacoEditorIns.focus();
};

const getEditor = () => {
    return monacoEditorIns;
};

defineExpose({ getEditor, format, focus });
</script>

<style lang="scss" scoped>
.monaco-editor-custom {
    :deep(.code-mode-select) {
        position: absolute;
        z-index: 2;
        right: 10px;
        top: 10px;
        max-width: 100px;
    }

    border: 1px solid var(--el-border-color-light, #ebeef5);
    width: 100%;
}
</style>
