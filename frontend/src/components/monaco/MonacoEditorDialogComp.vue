<template>
    <div>
        <el-dialog :title="state.title" v-model="state.dialogVisible" :width="state.width" @close="cancel">
            <monaco-editor ref="editorRef" :height="state.height" class="editor" :language="state.language" v-model="contentValue" can-change-mode />
            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="cancel">取消</el-button>
                    <el-button @click="confirm" type="primary">确定</el-button>
                </span>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, ref, reactive } from 'vue';
import { ElDialog, ElButton, ElMessage } from 'element-plus';
// import base style
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { MonacoEditorDialogProps } from './MonacoEditorDialog';

const editorRef: any = ref(null);

const state = reactive({
    dialogVisible: false,
    height: '450px',
    width: '800px',
    contentValue: '',
    title: '',
    language: '',
});

let confirmFn: any;
let cancelFn: any;

const { contentValue } = toRefs(state);

function compressHTML(html: string) {
    return (
        html
            .replace(/[\r\n\t]+/g, ' ') // 移除换行符和制表符
            // .replace(/<!--[\s\S]*?-->/g, '') // 移除注释
            .replace(/\s{2,}/g, ' ') // 合并多个空格为一个空格
            .replace(/>\s+</g, '><')
    ); // 移除标签之间的空格
}

/**
 * 确认按钮
 */
const confirm = async () => {
    if (confirmFn) {
        if (state.language === 'json') {
            let val;
            try {
                val = JSON.parse(contentValue.value);
                if (typeof val !== 'object') {
                    ElMessage.error('请输入正确的json');
                    return;
                }
            } catch (e) {
                ElMessage.error('请输入正确的json');
                return;
            }

            // 压缩json字符串
            confirmFn(JSON.stringify(val));
        } else if (state.language === 'html') {
            // 压缩html字符串
            confirmFn(compressHTML(contentValue.value));
        } else {
            confirmFn(contentValue.value);
        }
    }
    state.dialogVisible = false;
    setTimeout(() => {
        state.contentValue = '';
        state.title = '';
    }, 200);
};

const cancel = () => {
    state.dialogVisible = false;
    // 没有执行成功，并且取消回调函数存在，则执行
    cancelFn && cancelFn();
    setTimeout(() => {
        state.contentValue = '';
        state.title = '';
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

const open = (optionProps: MonacoEditorDialogProps) => {
    confirmFn = optionProps.confirmFn;
    cancelFn = optionProps.cancelFn;

    const language = optionProps.language;
    state.language = language;
    state.title = optionProps.title;
    if (optionProps.height) {
        state.height = optionProps.height;
    }

    state.contentValue = optionProps.content;
    // 格式化输出html;
    if (language === 'html' || language == 'xml') {
        state.contentValue = formatXML(optionProps.content);
    }

    setTimeout(() => {
        editorRef.value?.focus();
        editorRef.value?.format();
    }, 300);

    state.dialogVisible = true;
};

defineExpose({ open });
</script>
<style lang="scss" scoped>
.editor {
    font-size: 9pt;
    font-weight: 600;
}
</style>
