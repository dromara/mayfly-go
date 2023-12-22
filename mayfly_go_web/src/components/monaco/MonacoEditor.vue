<template>
    <div class="monaco-editor" style="border: 1px solid var(--el-border-color-light, #ebeef5); height: 100%">
        <div class="monaco-editor-content" ref="monacoTextarea" :style="{ height: height }"></div>
        <el-select v-if="canChangeMode" class="code-mode-select" v-model="languageMode" @change="changeLanguage" filterable>
            <el-option v-for="mode in languageArr" :key="mode.value" :label="mode.label" :value="mode.value"> </el-option>
        </el-select>
    </div>
</template>

<script lang="ts" setup>
import { ref, watch, toRefs, reactive, onMounted, onBeforeUnmount } from 'vue';
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';
// 相关语言
import 'monaco-editor/esm/vs/basic-languages/shell/shell.contribution.js';
import 'monaco-editor/esm/vs/basic-languages/yaml/yaml.contribution.js';
import 'monaco-editor/esm/vs/basic-languages/dockerfile/dockerfile.contribution.js';
import 'monaco-editor/esm/vs/basic-languages/javascript/javascript.contribution.js';
import 'monaco-editor/esm/vs/basic-languages/html/html.contribution.js';
import 'monaco-editor/esm/vs/basic-languages/css/css.contribution.js';
import 'monaco-editor/esm/vs/basic-languages/python/python.contribution.js';
import 'monaco-editor/esm/vs/basic-languages/markdown/markdown.contribution.js';
import 'monaco-editor/esm/vs/basic-languages/java/java.contribution.js';
import 'monaco-editor/esm/vs/basic-languages/sql/sql.contribution.js';
import 'monaco-editor/esm/vs/language/json/monaco.contribution';
// 右键菜单
import 'monaco-editor/esm/vs/editor/contrib/contextmenu/browser/contextmenu.js';
import 'monaco-editor/esm/vs/editor/contrib/caretOperations/browser/caretOperations.js';
import 'monaco-editor/esm/vs/editor/contrib/clipboard//browser/clipboard.js';
import 'monaco-editor/esm/vs/editor/contrib/find/browser/findController.js';
import 'monaco-editor/esm/vs/editor/contrib/format//browser/formatActions.js';
// 提示
import 'monaco-editor/esm/vs/editor/contrib/suggest/browser/suggestController.js';
import 'monaco-editor/esm/vs/editor/contrib/suggest/browser/suggestInlineCompletions.js';

import { editor, languages } from 'monaco-editor';
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker.js?worker';
import JsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';
// 主题仓库 https://github.com/brijeshb42/monaco-themes
// 主题例子 https://editor.bitwiser.in/
// import Monokai from 'monaco-themes/themes/Monokai.json'
// import Active4D from 'monaco-themes/themes/Active4D.json'
// import ahe from 'monaco-themes/themes/All Hallows Eve.json'
// import bop from 'monaco-themes/themes/Birds of Paradise.json'
// import krTheme from 'monaco-themes/themes/krTheme.json'
// import Dracula from 'monaco-themes/themes/Dracula.json'
import SolarizedLight from 'monaco-themes/themes/Solarized-light.json';
import { language as shellLan } from 'monaco-editor/esm/vs/basic-languages/shell/shell.js';
import { ElOption, ElSelect } from 'element-plus';

import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';

const { themeConfig } = storeToRefs(useThemeConfig());

const props = defineProps({
    modelValue: {
        type: String,
    },
    language: {
        type: String,
        default: null,
    },
    height: {
        type: String,
        default: '500px',
    },
    width: {
        type: String,
        default: 'auto',
    },
    canChangeMode: {
        type: Boolean,
        default: false,
    },
    options: {
        type: Object,
        default: null,
    },
});

//定义事件
const emit = defineEmits(['update:modelValue']);

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
        label: 'XML/HTML',
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
};

const monacoTextarea: any = ref();

let monacoEditorIns: editor.IStandaloneCodeEditor = null as any;
let completionItemProvider: any = null;

self.MonacoEnvironment = {
    getWorker(_: any, label: string) {
        if (label === 'json') {
            return new JsonWorker();
        }
        return new EditorWorker();
    },
};

const state = reactive({
    languageMode: 'shell',
});

const { languageMode } = toRefs(state);

onMounted(() => {
    state.languageMode = props.language;
    initMonacoEditorIns();
    setEditorValue(props.modelValue);
    registerCompletionItemProvider();
});

onBeforeUnmount(() => {
    if (monacoEditorIns) {
        monacoEditorIns.dispose();
    }
    if (completionItemProvider) {
        console.log('unmount=> dispose completion item provider');
        completionItemProvider.dispose();
    }
});

watch(
    () => props.modelValue,
    (newValue: any) => {
        if (!monacoEditorIns.hasTextFocus()) {
            state.languageMode = props.language;
            monacoEditorIns?.setValue(newValue);
        }
    }
);

watch(
    () => props.language,
    (newValue: any) => {
        changeLanguage(newValue);
    }
);

// 监听 themeConfig editorTheme配置文件的变化
watch(
    () => themeConfig.value.editorTheme,
    (val) => {
        console.log('monaco editor theme change: ', val);
        monaco?.editor?.setTheme(val);
    }
);

const initMonacoEditorIns = () => {
    console.log('初始化monaco编辑器');
    // options参数参考 https://microsoft.github.io/monaco-editor/api/interfaces/monaco.editor.IStandaloneEditorConstructionOptions.html#language
    // 初始化一些主题
    monaco.editor.defineTheme('SolarizedLight', SolarizedLight);
    defaultOptions.language = state.languageMode;
    defaultOptions.theme = themeConfig.value.editorTheme;
    monacoEditorIns = monaco.editor.create(monacoTextarea.value, Object.assign(defaultOptions, props.options as any));

    // 监听内容改变,双向绑定
    monacoEditorIns.onDidChangeModelContent(() => {
        emit('update:modelValue', monacoEditorIns.getModel()?.getValue());
    });
};

const changeLanguage = (value: any) => {
    console.log('change lan');
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

const setEditorValue = (value: any) => {
    monacoEditorIns.getModel()?.setValue(value);
};

/**
 * 注册联想补全提示
 */
const registerCompletionItemProvider = () => {
    if (completionItemProvider) {
        console.log('exist competion item provider, dispose now');
        completionItemProvider.dispose();
    }
    if (state.languageMode == 'shell') {
        registeShell();
    }
};

const registeShell = () => {
    completionItemProvider = monaco.languages.registerCompletionItemProvider('shell', {
        provideCompletionItems: async () => {
            let suggestions: languages.CompletionItem[] = [];
            shellLan.keywords.forEach((item: any) => {
                suggestions.push({
                    label: item,
                    kind: monaco.languages.CompletionItemKind.Keyword,
                    insertText: item,
                } as any);
            });
            shellLan.builtins.forEach((item: any) => {
                suggestions.push({
                    label: item,
                    kind: monaco.languages.CompletionItemKind.Property,
                    insertText: item,
                } as any);
            });
            return {
                suggestions: suggestions,
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

<style lang="scss">
.monaco-editor {
    .code-mode-select {
        position: absolute;
        z-index: 2;
        right: 10px;
        top: 10px;
        max-width: 130px;
    }
}
</style>
