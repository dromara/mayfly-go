<template>
    <div class="in-coder-panel">
        <textarea ref="textarea"></textarea>
        <el-select v-if="canChangeMode" class="code-mode-select" v-model="mode" @change="changeMode">
            <el-option v-for="mode in modes" :key="mode.value" :label="mode.label" :value="mode.value"> </el-option>
        </el-select>
    </div>
</template>

<script lang="ts">
import { ref, nextTick, toRefs, reactive, watch, onMounted, defineComponent } from 'vue';
// 引入全局实例
import _CodeMirror from 'codemirror';

// 核心样式
import 'codemirror/lib/codemirror.css';
// 引入主题后还需要在 options 中指定主题才会生效
import 'codemirror/theme/cobalt.css';
import 'codemirror/addon/selection/active-line.js';
// 匹配括号
import 'codemirror/addon/edit/matchbrackets.js';
import 'codemirror/addon/selection/active-line';
import 'codemirror/addon/comment/comment';

// 需要引入具体的语法高亮库才会有对应的语法高亮效果
// codemirror 官方其实支持通过 /addon/mode/loadmode.js 和 /mode/meta.js 来实现动态加载对应语法高亮库
// 但 vue 貌似没有无法在实例初始化后再动态加载对应 JS ，所以此处才把对应的 JS 提前引入
import 'codemirror/mode/yaml/yaml.js';
import 'codemirror/mode/dockerfile/dockerfile.js';
import 'codemirror/mode/nginx/nginx.js';
import 'codemirror/mode/javascript/javascript.js';
import 'codemirror/mode/css/css.js';
import 'codemirror/mode/xml/xml.js';
import 'codemirror/mode/markdown/markdown.js';
import 'codemirror/mode/python/python.js';
import 'codemirror/mode/shell/shell.js';
import 'codemirror/mode/sql/sql.js';
import 'codemirror/mode/vue/vue.js';
import 'codemirror/mode/textile/textile.js';
import 'codemirror/addon/hint/show-hint.css';
import 'codemirror/addon/hint/show-hint.js';

// 尝试获取全局实例
const CodeMirror = (window as any).CodeMirror || _CodeMirror;

export default defineComponent({
    name: 'codemirror',
    props: {
        modelValue: {
            type: String,
        },
        language: {
            type: String,
            default: null,
        },
        canChangeMode: {
            type: Boolean,
            default: false,
        },
        options: {
            type: Object,
            default: null,
        },
    },

    setup(props: any, { emit }) {
        let { modelValue, language } = toRefs(props);
        const textarea: any = ref(null);
        // 编辑器实例
        let coder = null as any;

        const state = reactive({
            coder: null as any,
            content: '',
            // 默认的语法类型
            mode: 'x-sh',
            // 默认配置
            options: {
                // 缩进格式
                tabSize: 2,
                // 主题，对应主题库 JS 需要提前引入
                theme: 'cobalt',
                // 显示行号
                lineNumbers: true,
                line: true,
                indentWithTabs: true,
                smartIndent: true,
                matchBrackets: true,
                autofocus: true,
                styleSelectedText: true,
                styleActiveLine: true, // 高亮选中行
                foldGutter: true, // 块槽
                // extraKeys: { Tab: 'autocomplete' }, // 自定义快捷键
                hintOptions: {
                    // 当匹配只有一项的时候是否自动补全
                    completeSingle: false,
                },
            },
            // 支持切换的语法高亮类型，对应 JS 已经提前引入
            // 使用的是 MIME-TYPE ，不过作为前缀的 text/ 在后面指定时写死了
            modes: [
                {
                    value: 'x-sh',
                    label: 'Shell',
                },
                {
                    value: 'x-yaml',
                    label: 'Yaml',
                },
                {
                    value: 'x-dockerfile',
                    label: 'Dockerfile',
                },
                {
                    value: 'x-nginx-conf',
                    label: 'Nginx',
                },
                {
                    value: 'html',
                    label: 'XML/HTML',
                },
                {
                    value: 'x-python',
                    label: 'Python',
                },
                {
                    value: 'x-sql',
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
                    value: 'x-java',
                    label: 'Java',
                },
                {
                    value: 'x-vue',
                    label: 'Vue',
                },
                {
                    value: 'markdown',
                    label: 'Markdown',
                },
                {
                    value: 'text/x-textile',
                    label: 'text',
                },
            ],
        });

        onMounted(() => {
            init();
        });

        watch(
            () => props.modelValue,
            (newValue, oldValue) => {
                handerCodeChange(newValue);
            }
        );

        watch(
            () => props.options,
            (newValue, oldValue) => {
                console.log('options change', newValue);
                for (const key in newValue) {
                    coder.setOption(key, newValue[key]);
                }
            }
        );

        const init = () => {
            if (props.options) {
                state.options = props.options;
            }
            // 初始化编辑器实例，传入需要被实例化的文本域对象和默认配置
            coder = CodeMirror.fromTextArea(textarea.value, state.options);
            coder.setValue(modelValue.value || state.content);

            // 支持双向绑定
            coder.on('change', (coder: any) => {
                state.content = coder.getDoc().getValue();
                emit('update:modelValue', state.content);
            });

            coder.on('inputRead', (instance: any, changeObj: any) => {
                if (/^[a-zA-Z]/.test(changeObj.text[0])) {
                    instance.showHint();
                }
            });

            // 修改编辑器的语法配置
            setMode(language.value);

            [
                'scroll',
                'changes',
                'beforeChange',
                'cursorActivity',
                'keyHandled',
                'inputRead',
                'electricInput',
                'beforeSelectionChange',
                'viewportChange',
                'swapDoc',
                'gutterClick',
                'gutterContextMenu',
                'focus',
                'blur',
                'refresh',
                'optionChange',
                'scrollCursorIntoView',
                'update',
            ].forEach((event) => {
                // 循环事件，并兼容 run-time 事件命名
                coder.on(event, (...args: any) => {
                    // console.log('当有事件触发了', event, args);
                    emit(event, ...args);
                    const lowerCaseEvent = event.replace(/([A-Z])/g, '-$1').toLowerCase();
                    if (lowerCaseEvent !== event) {
                        emit(lowerCaseEvent, ...args);
                    }
                });
            });

            state.coder = coder;
            // 不加无法显示内容，需点击后才可显示
            refresh();
        };

        const refresh = () => {
            nextTick(() => {
                coder.refresh();
            });
        };

        // 设置模式
        const setMode = (val: string) => {
            if (val) {
                // 获取具体的语法类型对象
                let modeObj = getLanguage(val);
                // 判断父容器传入的语法是否被支持
                if (modeObj) {
                    state.mode = modeObj.value;
                }
            }
            // 修改编辑器的语法配置
            coder.setOption('mode', `text/${state.mode}`);
        };

        // 获取当前语法类型
        const getLanguage = (language: string) => {
            // 在支持的语法类型列表中寻找传入的语法类型
            return state.modes.find((mode: any) => {
                // 所有的值都忽略大小写，方便比较
                let currentLanguage = language.toLowerCase();
                let currentLabel = mode.label.toLowerCase();
                let currentValue = mode.value.toLowerCase();

                // 由于真实值可能不规范，例如 java 的真实值是 x-java ，所以讲 value 和 label 同时和传入语法进行比较
                return currentLabel === currentLanguage || currentValue === currentLanguage;
            });
        };

        // 更改模式
        const changeMode = (val: string) => {
            setMode(val);
            // 获取修改后的语法
            let label = (getLanguage(val) as any).label.toLowerCase();

            // 允许父容器通过以下函数监听当前的语法值
            emit('language-change', label);
        };

        const handerCodeChange = (newVal: string) => {
            const cm_value = coder.getValue();
            if (newVal !== cm_value) {
                const scrollInfo = coder.getScrollInfo();
                coder.setValue(newVal);
                state.content = newVal;
                coder.scrollTo(scrollInfo.left, scrollInfo.top);
            }
        };

        return {
            ...toRefs(state),
            textarea,
            changeMode,
        };
    },
});
</script>

<style lang="scss">
.CodeMirror {
    height: 500px;
}
.in-coder-panel {
    flex-grow: 1;
    display: flex;
    position: relative;
    .CodeMirror {
        flex-grow: 1;
        z-index: 1;
        .CodeMirror-code {
            line-height: 19px;
        }
    }

    .code-mode-select {
        position: absolute;
        z-index: 2;
        right: 10px;
        top: 10px;
        max-width: 130px;
    }
}
</style>