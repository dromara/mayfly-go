<template>
    <div class="format-viewer-container">
        <div class="mb5 fr">
            <el-select v-model="selectedView" class='format-selector' size='mini' placeholder='Text'>
                <template #prefix>
                    <SvgIcon name="view" />
                </template>
                <el-option v-for="item of Object.keys(viewers)" :key="item" :label="item" :value="item">
                </el-option>
            </el-select>
        </div>

        <component ref='viewerRef' :is='components[viewerComponent]' :content='state.content' :name="selectedView">
        </component>
    </div>
</template>
<script lang="ts" setup>
import { ref, reactive, computed, shallowReactive, watch, toRefs, onMounted } from 'vue';
import ViewerText from './ViewerText.vue';
import ViewerJson from './ViewerJson.vue';

const props = defineProps({
    content: {
        type: String,
    },
})

const components = shallowReactive({
    ViewerText, ViewerJson
})
const viewerRef: any = ref(null)

const state = reactive({
    content: '',
    selectedView: 'Text',
});

const viewers = {
    "Text": {
        value: 'ViewerText',
    },

    "Json": {
        value: 'ViewerJson',
    }
}

const {
    selectedView,
} = toRefs(state)

const viewerComponent = computed(() => {
    return viewers[state.selectedView].value;
});

watch(
    () => props.content,
    (val: any) => {
        state.content = val;
    }
);

onMounted(() => {
    state.content = props.content as any;
})

const getContent = () => {
    return viewerRef.value.getContent();
}

defineExpose({ getContent })

</script>
<style lang="scss">
.format-selector {
    width: 130px;
}

.format-selector .el-input__inner {
    height: 22px !important;
}

/*outline same with text viewer's .el-textarea__inner*/
.format-viewer-container .text-formated-container {
    border: 1px solid #dcdfe6;
    padding: 5px 10px;
    border-radius: 4px;
    clear: both
}

.format-viewer-container .formater-binary-tag {
    font-size: 80%;
}

// 默认文本框样式
.format-viewer-container .el-textarea textarea {
    font-size: 14px;
    height: calc(100vh - 536px);
}

.format-viewer-container .monaco-editor-content {
    height: calc(100vh - 550px) !important;
}
</style>
