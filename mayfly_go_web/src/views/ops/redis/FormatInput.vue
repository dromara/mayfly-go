<template>
    <div style="width: 100%;">
        <el-input @input="onInput" type="textarea" v-model="modelValue" :autosize="autosize" :rows="rows" />
        <div style="padding: 3px; float: right" class="mr5 format-btns">
            <div>
                <el-button @click="showFormatDialog()" :underline="false" type="success" icon="MagicStick" size="small">
                </el-button>
            </div>
        </div>
        <el-dialog @opened="opened" width="60%" :title="title" v-model="formatDialog.visible"
            :close-on-click-modal="false">
            <monaco-editor ref="monacoEditorRef" :canChangeMode="true" v-model="formatDialog.value" language="json" />
            <template #footer>
                <div>
                    <el-button @click="formatDialog.visible = false">取 消</el-button>
                    <el-button @click="onConfirmValue" type="primary">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>
<script lang="ts" setup>
import { ref, reactive, watch, toRefs, onMounted } from 'vue';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';

const props = defineProps({
    title: {
        type: String,
    },
    modelValue: {
        type: String,
    },
    rows: {
        type: Number,
    },
    autosize: {
        type: Object
    }
})

const emit = defineEmits(['update:modelValue'])

const monacoEditorRef: any = ref(null)

const state = reactive({
    rows: 2,
    autosize: {},
    modelValue: '',
    formatDialog: {
        visible: false,
        value: '',
    }
});

const {
    rows,
    autosize,
    modelValue,
    formatDialog,
} = toRefs(state)

watch(
    () => props.modelValue,
    (val: any) => {
        state.modelValue = val;
    }
);

onMounted(() => {
    state.modelValue = props.modelValue as any;
    state.autosize = props.autosize as any;
    state.rows = props.rows as any;
})

const showFormatDialog = () => {
    state.formatDialog.visible = true;
    state.formatDialog.value = state.modelValue;
}

const opened = () => {
    monacoEditorRef.value.format();
};

const onConfirmValue = () => {
    // 尝试压缩json
    try {
        state.modelValue = JSON.stringify(JSON.parse(state.formatDialog.value));
    } catch (e) {
        state.modelValue = state.formatDialog.value;
    }
    emit('update:modelValue', state.modelValue);
    state.formatDialog.visible = false;
}

const onInput = (value: any) => {
    emit('update:modelValue', value);
}

</script>
<style lang="scss">
.format-btns {
    position: absolute;
    z-index: 2;
    right: 5px;
    top: 4px;
    max-width: 120px;
}
</style>
