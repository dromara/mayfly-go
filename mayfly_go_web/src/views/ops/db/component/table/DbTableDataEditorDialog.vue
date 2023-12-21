<template>
    <div>
        <el-dialog :destroy-on-close="true" :title="state.title" v-model="dialogVisible" :show-close="false" width="800px" @close="cancel">
            <monaco-editor height="600px" class="codesql" :language="state.language" v-model="contentValue" />
            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="cancel">取消</el-button>
                    <el-button @click="submit" type="primary">确定</el-button>
                </span>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, ref, nextTick, reactive } from 'vue';
import { ElDialog, ElButton, InputInstance, ElMessage } from 'element-plus';
// import base style
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';

import { TableDataEditorProps } from './DbTableDataEditorDialog';

const remarkInputRef = ref<InputInstance>();
const state = reactive({
    dialogVisible: false,
    contentValue: '',
    title: '',
    language: '',
});

const { dialogVisible, contentValue } = toRefs(state);

let cancelCallback: any;
let runSuccessCallback: any;
let runSuccess: boolean = false;

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
 * 执行sql
 */
const submit = async () => {
    runSuccess = true;
    if (runSuccessCallback) {
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
            runSuccessCallback(JSON.stringify(val));
        } else if (state.language === 'html') {
            // 压缩html字符串
            runSuccessCallback(compressHTML(contentValue.value));
        } else {
            runSuccessCallback(contentValue.value);
        }
    }
    state.dialogVisible = false;
    setTimeout(() => {
        state.contentValue = '';
        state.title = '';
        runSuccessCallback = null;
        runSuccess = false;
    }, 200);
};

const cancel = () => {
    state.dialogVisible = false;
    // 没有执行成功，并且取消回调函数存在，则执行
    if (!runSuccess && cancelCallback) {
        cancelCallback();
    }
    setTimeout(() => {
        state.contentValue = '';
        state.title = '';
        cancelCallback = null;
        runSuccess = false;
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

const open = (props: TableDataEditorProps) => {
    cancelCallback = props.cancelCallback;
    runSuccessCallback = props.runSuccessCallback;
    // 格式化输出json
    if (props.language === 'json') {
        try {
            state.contentValue = JSON.stringify(JSON.parse(props.content), null, '\t');
        } catch (e) {
            state.contentValue = 'json格式字符串错误: ' + props.content;
        }
    }
    // 格式化输出html
    if (props.language === 'html') {
        state.contentValue = formatXML(props.content);
    }
    state.title = props.title;
    state.language = props.language;
    state.dialogVisible = true;
    nextTick(() => {
        setTimeout(() => {
            remarkInputRef.value?.focus();
        });
    });
};

defineExpose({ open });
</script>
<style lang="scss">
.codesql {
    font-size: 9pt;
    font-weight: 600;
}
</style>
