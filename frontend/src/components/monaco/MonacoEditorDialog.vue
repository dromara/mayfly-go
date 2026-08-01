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
import { ref, watch } from 'vue';
// import base style
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { Msg } from '@/hooks/useI18n';
import { i18n } from '@/i18n';
import { registerCompletionItemProvider } from './completionItemProvider';
import { MonacoEditorDialogProps } from './MonacoEditorBox';

const editorRef = ref<InstanceType<typeof MonacoEditor> | null>(null);

const props = defineProps<MonacoEditorDialogProps>();

const modelValue = defineModel<string>('modelValue', {
    type: String,
    default: '',
});

const dialogVisible = defineModel<boolean>('visible', {
    type: Boolean,
    default: false,
});

const emit = defineEmits(['close', 'confirm']);

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

watch(
    () => props.language,
    () => {
        // 格式化输出html;
        const language = props.language;
        if (language === 'html' || language == 'xml') {
            modelValue.value = formatXML(modelValue.value);
        }

        if (props.completionItemProvider) {
            registerCompletionItemProvider(language, props.completionItemProvider);
        }

        setTimeout(() => {
            editorRef.value?.focus();
            editorRef.value?.format();
        }, 300);
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
                Msg.error('Invalid json');
                return;
            }
        } catch (e) {
            Msg.error('Invalid json');
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
