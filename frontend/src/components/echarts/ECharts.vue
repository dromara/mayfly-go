<template>
    <div ref="chartRef" :style="echartsStyle" />
</template>

<script setup lang="ts" name="ECharts">
import { ref, onMounted, onBeforeUnmount, watch, computed, markRaw, nextTick } from 'vue';
import { EChartsType, ECElementEvent } from 'echarts/core';
import echarts, { ECOption } from './config';
import { useDebounceFn, useEventListener } from '@vueuse/core';
import { light } from './config/theme';
// import { useThemeConfig } from '@/store/themeConfig';
// import { storeToRefs } from 'pinia';

interface Props {
    option: ECOption;
    renderer?: 'canvas' | 'svg';
    resize?: boolean;
    theme?: Record<string, unknown> | string;
    width?: number | string;
    height?: number | string;
    onClick?: (event: ECElementEvent) => void;
}

const props = withDefaults(defineProps<Props>(), {
    renderer: 'canvas',
    resize: true,
});

const echartsStyle = computed(() => {
    return props.width || props.height ? { height: props.height + 'px', width: props.width + 'px' } : { height: '100%', width: '100%' };
});

const chartRef = ref<HTMLDivElement | HTMLCanvasElement>();
const chartInstance = ref<EChartsType>();

const draw = () => {
    if (chartInstance.value) {
        chartInstance.value.setOption(props.option, { notMerge: true });
    }
};

watch(
    () => props.option,
    () => {
        draw();
    },
    { deep: true }
);

const handleClick = (event: ECElementEvent) => props.onClick && props.onClick(event);

const init = () => {
    if (!chartRef.value) return;
    chartInstance.value = echarts.getInstanceByDom(chartRef.value);

    if (!chartInstance.value) {
        chartInstance.value = markRaw(
            echarts.init(chartRef.value, props.theme ?? light, {
                renderer: props.renderer,
            })
        );
        chartInstance.value.on('click', handleClick);
        draw();
    }
};

const resize = () => {
    if (chartInstance.value && props.resize) {
        chartInstance.value.resize({ animation: { duration: 300 } });
    }
};

const debouncedResize = useDebounceFn(resize, 300, { maxWait: 800 });

// 容器尺寸变化监听：弹窗/折叠面板内图表常在 0 尺寸时初始化（如对话框展开动画、下方区域尚未布局），
// 仅靠 window resize 无法恢复，导致画布恒为 0x0 不渲染；用 ResizeObserver 在容器获得真实尺寸后重绘
let resizeObserver: ResizeObserver | undefined;

onMounted(() => {
    nextTick(() => init());
    useEventListener('resize', debouncedResize);
    if (chartRef.value && typeof ResizeObserver !== 'undefined') {
        resizeObserver = new ResizeObserver(() => debouncedResize());
        resizeObserver.observe(chartRef.value);
    }
});

onBeforeUnmount(() => {
    resizeObserver?.disconnect();
    chartInstance.value?.dispose();
});

defineExpose({
    getInstance: () => chartInstance.value,
    resize,
    draw,
});
</script>
