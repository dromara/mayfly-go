<template>
    <div>
        <!-- Dialog 模式 -->
        <el-dialog :title="props.title" v-model="dialogVisible" :width="props.width" @close="close" v-if="!props.useDrawer">
            <monaco-editor
                ref="editorRef"
                :height="props.height"
                class="editor"
                :language="props.language"
                v-model="modelValue"
                :options="props.options"
                :can-change-mode="props.canChangeLang"
                @ready="onEditorReady"
            />
            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="dialogVisible = false">{{ i18n.global.t('common.cancel') }}</el-button>
                    <el-button v-if="props.showConfirmButton" @click="confirm" type="primary">{{ i18n.global.t('common.confirm') }}</el-button>
                </span>
            </template>
        </el-dialog>

        <!-- Drawer 模式 -->
        <el-drawer
            :append-to-body="false"
            :title="props.title"
            v-model="dialogVisible"
            :size="props.drawerSize || '50%'"
            @close="close"
            :destroy-on-close="true"
            :close-on-click-modal="true"
            class="monaco-editor-drawer"
            v-else
        >
            <monaco-editor
                ref="editorRef"
                :height="props.height || 'calc(100vh  - 120px)'"
                class="editor"
                :language="props.language"
                v-model="modelValue"
                :options="props.options"
                :can-change-mode="props.canChangeLang"
                @ready="onEditorReady"
            />
            <template #footer>
                <div class="drawer-footer">
                    <el-button @click="dialogVisible = false">{{ i18n.global.t('common.cancel') }}</el-button>
                    <el-button v-if="props.showConfirmButton" @click="confirm" type="primary">{{ i18n.global.t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { ElButton, ElDialog, ElDrawer } from 'element-plus';
import { defineAsyncComponent, ref, watch } from 'vue';
import { Msg } from '@/hooks/useI18n';
import { i18n } from '@/i18n';
import type { MonacoEditorDialogProps, MonacoEditorExpose } from './types';

// 编辑器主体在生产是约 967KB(gzip) 的独立 chunk，而本弹窗被 MonacoEditorBox 静态引用（该入口又常被
// api 模块等底层文件静态引入），所以编辑器必须动态引入，否则整份 monaco 会顺着静态图回流到首屏。
// main.ts 在首屏空闲时预取该 chunk，故此处异步解析通常瞬时完成，弹窗打开即显示编辑器。
const MonacoEditor = defineAsyncComponent(() => import('./MonacoEditor.vue'));

const editorRef = ref<MonacoEditorExpose | null>(null);

const props = defineProps<MonacoEditorDialogProps>();

const modelValue = defineModel<string>('modelValue', {
    type: String,
    default: '',
});

const dialogVisible = defineModel<boolean>('visible', {
    type: Boolean,
    default: false,
});

const emit = defineEmits<{
    /** 弹窗关闭（取消或确认后都会触发），命令式挂载方据此销毁容器 */
    close: [];
    /** 确认，回传压缩后的字符串 */
    confirm: [value: string];
}>();

const formatXML = function (xml: string, tab?: string) {
    let formatted = '',
        indent = '';
    tab = tab || '    ';
    xml.split(/>\s*</).forEach(function (node) {
        if (node.match(/^\/\w/)) indent = indent.substring(tab!.length);
        formatted += indent + '<' + node + '>\r\n';
        if (node.match(/^<?\w[^>]*[^\/]$/)) indent += tab;
    });
    return formatted.substring(1, formatted.length - 3);
};

// 编辑器是异步组件，下面 watch 的 immediate 那次执行时它还没挂载，ref 恒为 null，此刻 focus/format 必然落空。
// 所以未就绪时直接跳过即可：编辑器挂载完成会发出 ready，由它补上这一枪（重开弹窗时同理，每次就绪都格式化一遍）。
let editorReady = false;

const requestFocusFormat = () => {
    if (!editorReady) {
        return;
    }
    // 保留原有的 300ms，等抽屉/弹窗的展开动画与内容写入完成
    setTimeout(() => {
        editorRef.value?.focus();
        editorRef.value?.format();
    }, 300);
};

const onEditorReady = () => {
    editorReady = true;
    requestFocusFormat();
};

watch(
    () => props.language,
    () => {
        // 格式化输出html;
        const language = props.language;
        if (language === 'html' || language == 'xml') {
            modelValue.value = formatXML(modelValue.value);
        }

        requestFocusFormat();
    },
    { immediate: true }
);

/**
 * 确认按钮
 */
const confirm = async () => {
    let value = modelValue.value;
    if (props.language === 'json') {
        let val;
        try {
            val = JSON.parse(value);
            if (typeof val !== 'object') {
                Msg.error('common.invalidJson');
                return;
            }
        } catch (e) {
            Msg.error('common.invalidJson');
            return;
        }

        // 压缩json字符串
        value = JSON.stringify(val);
    } else if (props.language === 'html') {
        // 压缩html字符串
        value = compressHTML(value);
    }

    emit('confirm', value);
    close();
};

const close = () => {
    dialogVisible.value = false;
    emit('close');
    setTimeout(() => {
        modelValue.value = '';
    }, 200);
};

function compressHTML(html: string) {
    return html
        .replace(/[\r\n\t]+/g, ' ') // 移除换行符和制表符
        .replace(/\s{2,}/g, ' ') // 合并多个空格为一个空格
        .replace(/>\s+</g, '><'); // 移除标签之间的空格
}
</script>
<style lang="scss" scoped>
.editor {
    font-size: 9pt;
    font-weight: 600;
}

.drawer-footer {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    padding: 10px 0;
}

:deep(.monaco-editor-drawer) {
    .el-drawer__header {
        margin-bottom: 20px;
    }
    .el-drawer__body {
        padding: 0;
    }
    .el-drawer__footer {
        padding: 0 20px;
    }
}
</style>
