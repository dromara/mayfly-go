<template>
    <el-descriptions border size="small" :title="`${progress.title}`">
        <el-descriptions-item label="时间">{{ state.elapsedTime }}</el-descriptions-item>
        <el-descriptions-item label="已处理">{{ progress.executedStatements }}</el-descriptions-item>
    </el-descriptions>
</template>
<script lang="ts" setup>
import { onMounted, onUnmounted, reactive } from 'vue';
import { formatTime } from 'element-plus/es/components/countdown/src/utils';
import { buildProgressProps } from './progress-notify';

const props = defineProps(buildProgressProps());

const state = reactive({
    elapsedTime: '00:00:00',
});

let timer: any = undefined;
const startTime = Date.now();

onMounted(async () => {
    timer = setInterval(() => {
        const elapsed = Date.now() - startTime;
        state.elapsedTime = formatTime(elapsed, 'HH:mm:ss');
    }, 1000);
});

onUnmounted(async () => {
    if (timer != undefined) {
        clearInterval(timer); // 在Vue实例销毁前，清除我们的定时器
        timer = undefined;
    }
});
</script>
