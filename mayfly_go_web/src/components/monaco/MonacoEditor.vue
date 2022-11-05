<template>
    <div class="monaco-editor">
        <div ref="monacoTextarea" :style="{ height: height }"></div>
        <el-select v-if="canChangeMode" class="code-mode-select" v-model="languageMode" @change="changeLanguage">
            <el-option v-for="mode in languages" :key="mode.value" :label="mode.label" :value="mode.value"> </el-option>
        </el-select>
    </div>
</template>

<script lang="ts" setup>
import { ref, watch, toRefs, reactive, onMounted, onBeforeUnmount } from 'vue';
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker.js?worker';
import * as monaco from 'monaco-editor';
import { editor } from 'monaco-editor';

// 主题仓库 https://github.com/brijeshb42/monaco-themes 
// 主题例子 https://editor.bitwiser.in/
// import Monokai from 'monaco-themes/themes/Monokai.json'
// import Active4D from 'monaco-themes/themes/Active4D.json'
// import ahe from 'monaco-themes/themes/All Hallows Eve.json'
// import bop from 'monaco-themes/themes/Birds of Paradise.json'
// import krTheme from 'monaco-themes/themes/krTheme.json'
// import Dracula from 'monaco-themes/themes/Dracula.json'
import SolarizedLight from 'monaco-themes/themes/Solarized-light.json'
import { language as shellLan } from 'monaco-editor/esm/vs/basic-languages/shell/shell.js';

import { ElOption, ElSelect } from 'element-plus';

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
})

//定义事件
const emit = defineEmits(['update:modelValue'])

const languages = [
    {
        value: 'shell',
        label: 'Shell',
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
        value: 'json',
        label: 'JSON',
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

const options = {
    language: 'shell',
    theme: 'SolarizedLight',
    automaticLayout: true, //自适应宽高布局
    foldingStrategy: 'indentation',//代码可分小段折叠
    roundedSelection: false, // 禁用选择文本背景的圆角
    matchBrackets: 'near',
    linkedEditing: true,
    cursorBlinking: 'smooth',// 光标闪烁样式
    mouseWheelZoom: true, // 在按住Ctrl键的同时使用鼠标滚轮时，在编辑器中缩放字体
    overviewRulerBorder: false, // 不要滚动条的边框
    tabSize: 2, // tab 缩进长度
    fontFamily: 'JetBrainsMono', // 字体 暂时不要设置，否则光标容易错位
    fontWeight: 'bold',
    // fontSize: 12,
    // letterSpacing: 1, 字符间距
    // quickSuggestions:false, // 禁用代码提示
    minimap: {
        enabled: false, // 不要小地图
    },
}

const state = reactive({
    languageMode: 'shell',
})

const {
    languageMode,
} = toRefs(state)

onMounted(() => {
    state.languageMode = props.language;
    initMonacoEditor();
    setEditorValue(props.modelValue);
    registerCompletionItemProvider();
});

onBeforeUnmount(() => {
    if (monacoEditor) {
        monacoEditor.dispose();
    }
    if (completionItemProvider) {
        completionItemProvider.dispose();
    }
})

watch(() => props.modelValue, (newValue: any) => {
    if (!monacoEditor.hasTextFocus()) {
        state.languageMode = props.language
        monacoEditor?.setValue(newValue)
        changeLanguage(state.languageMode)
    }
})


const monacoTextarea: any = ref(null);

let monacoEditor: editor.IStandaloneCodeEditor = null;
let completionItemProvider: any = null;

self.MonacoEnvironment = {
    getWorker() {
        return new EditorWorker();
    }
};

const initMonacoEditor = () => {
    console.log('初始化monaco编辑器')
    // options参数参考 https://microsoft.github.io/monaco-editor/api/interfaces/monaco.editor.IStandaloneEditorConstructionOptions.html#language
    // 初始化一些主题
    monaco.editor.defineTheme('SolarizedLight', SolarizedLight);
    options.language = state.languageMode;
    monacoEditor = monaco.editor.create(monacoTextarea.value, Object.assign(options, props.options));

    // 监听内容改变,双向绑定
    monacoEditor.onDidChangeModelContent(() => {
        emit('update:modelValue', monacoEditor.getModel().getValue());
    })

    // 动态设置主题
    // monaco.editor.setTheme('hc-black');
};

const changeLanguage = (value: any) => {
    console.log('change lan');
    // 获取当前的文档模型
    let oldModel = monacoEditor.getModel()
    // 创建一个新的文档模型
    let newModel = monaco.editor.createModel(oldModel.getValue(), value)
    // 设置成新的
    monacoEditor.setModel(newModel)
    // 销毁旧的模型
    if (oldModel) {
        oldModel.dispose()
    }

    registerCompletionItemProvider();
}

const setEditorValue = (value: any) => {
    monacoEditor.getModel().setValue(value)
}

/**
 * 注册联想补全提示
 */
const registerCompletionItemProvider = () => {
    if (completionItemProvider) {
        completionItemProvider.dispose();
    }
    if (state.languageMode == 'shell') {
        registeShell()
    }
}

const registeShell = () => {
    completionItemProvider = monaco.languages.registerCompletionItemProvider('shell', {
        provideCompletionItems: async () => {
            let suggestions: languages.CompletionItem[] = []
            shellLan.keywords.forEach((item: any) => {
                suggestions.push({
                    label: item,
                    kind: monaco.languages.CompletionItemKind.Keyword,
                    insertText: item,
                });
            })
            shellLan.builtins.forEach((item: any) => {
                suggestions.push({
                    label: item,
                    kind: monaco.languages.CompletionItemKind.Property,
                    insertText: item,
                });
            })
            return {
                suggestions: suggestions
            };
        }
    })
};
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