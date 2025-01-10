<template>
    <div>
        <el-dialog :title="props.title" v-model="dialogVisible" :width="props.width" @close="close">
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
    </div>
</template>

<script lang="ts" setup>
import { ref, watch } from 'vue';
import { ElDialog, ElButton, ElMessage } from 'element-plus';
// import base style
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { MonacoEditorDialogProps } from './MonacoEditorBox';
import { i18n } from '@/i18n';
import { registerCompletionItemProvider } from './completionItemProvider';

const editorRef: any = ref(null);

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
                ElMessage.error('Invalid json');
                return;
            }
        } catch (e) {
            ElMessage.error('Invalid json');
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

function compressHTML(html: string) {
    return (
        html
            .replace(/[\r\n\t]+/g, ' ') // 移除换行符和制表符
            // .replace(/<!--[\s\S]*?-->/g, '') // 移除注释
            .replace(/\s{2,}/g, ' ') // 合并多个空格为一个空格
            .replace(/>\s+</g, '><')
    ); // 移除标签之间的空格
}
</script>
<style lang="scss" scoped>
.editor {
    font-size: 9pt;
    font-weight: 600;
}
</style>
