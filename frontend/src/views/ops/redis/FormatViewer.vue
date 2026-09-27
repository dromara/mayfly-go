<template>
    <div class="format-viewer-container flex min-h-0 flex-col">
        <!-- 查看方式靠左，读数与复制靠右 -->
        <div class="flex items-center gap-2 pb-1.5">
            <el-radio-group v-model="selectedView" size="small">
                <el-radio-button v-for="item of viewerOptions" :key="item.value" :value="item.value">{{ item.label }}</el-radio-button>
            </el-radio-group>
            <div class="ml-auto flex items-center gap-2">
                <span class="size-text">{{ formatByteSize(state.contentSize) }}</span>
                <el-tooltip :content="$t('common.copy')" placement="top">
                    <el-button text size="small" icon="DocumentCopy" :aria-label="$t('common.copy')" @click="onCopyContent" />
                </el-tooltip>
            </div>
        </div>

        <component :is="components[viewerComponent]" ref="viewerRef" v-model:content="state.content" :readonly="props.readonly" class="min-h-0 flex-1" />
    </div>
</template>
<script lang="ts" setup>
import { ref, reactive, computed, watch, toRefs, onMounted, nextTick, type Component } from 'vue';
import { useI18n } from 'vue-i18n';
import { copyToClipboard } from '@/common/utils/string';
import ViewerText from './ViewerText.vue';
import ViewerJson from './ViewerJson.vue';
import { formatByteSize } from '@/common/utils/format';

const props = defineProps<{
    content?: string;
    /** 只读：值由服务端统计得出（如 HyperLogLog 基数），不允许直接改写 */
    readonly?: boolean;
}>();

/** 正文与服务端快照是否不一致，宿主据此决定「保存」是否可用 */
const emit = defineEmits<{
    change: [dirty: boolean];
}>();

const { t } = useI18n();

const components: Record<string, Component> = {
    ViewerText,
    ViewerJson,
};
const viewerRef = ref<{ getContent: () => string } | null>(null);

const state = reactive({
    content: '',
    contentSize: 0,
    selectedView: 'Text',
});

const viewers: Record<string, { value: string }> = {
    Text: {
        value: 'ViewerText',
    },

    Json: {
        value: 'ViewerJson',
    },
};

const { selectedView } = toRefs(state);

/** 切换器只两种，分段控件比下拉少一次点击，也能一眼看到可选项 */
const viewerOptions = computed(() =>
    Object.keys(viewers).map((name) => ({ value: name, label: name === 'Json' ? t('redis.formatJson') : t('redis.formatText') }))
);

const viewerComponent = computed(() => {
    return viewers[state.selectedView].value;
});

// 服务端下发的快照是「未修改」的基准线，只有偏离它才算脏
let baseline = '';

watch(
    () => props.content,
    (val?: string) => {
        setContent(val ?? '');
    }
);

// 用户编辑后同步读数与脏标记；控件回写的同值内容不会触发本监听
watch(
    () => state.content,
    (val: string) => {
        state.contentSize = new Blob([val]).size;
        emit('change', val !== baseline);
    }
);

onMounted(() => {
    setContent(props.content ?? '');
});

const setContent = (content: string) => {
    baseline = content;
    state.content = content;
    state.contentSize = new Blob([content]).size;
    try {
        JSON.parse(content);
        state.selectedView = 'Json';
    } catch (e) {
        state.selectedView = 'Text';
    }
    nextTick(() => emit('change', false));
};

const onCopyContent = async () => {
    // 复制结果提示由 copyToClipboard 统一发出
    await copyToClipboard(state.content);
};

const getContent = () => {
    return viewerRef.value?.getContent();
};

defineExpose({ getContent });
</script>

<style lang="scss">
.format-viewer-container .size-text {
    color: var(--el-text-color-secondary);
    font-family: var(--el-font-family-mono, ui-monospace, monospace);
    font-size: 12px;
}

// 编辑器容器：统一圆角与描边，深浅色都走变量，避免写死浅色值
.format-viewer-container .text-formated-container {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    padding: 4px 8px;
    clear: both;
}

.format-viewer-container .el-textarea__inner {
    border-radius: 8px;
}

.format-viewer-container .formater-binary-tag {
    font-size: 80%;
}

// 高度完全由外层 flex 容器决定，不再用「100vh - 魔法数」猜可用空间
.format-viewer-container .el-textarea,
.format-viewer-container .text-formated-container {
    height: 100%;
}

.format-viewer-container .el-textarea .el-textarea__inner {
    font-size: 14px;
}
</style>
