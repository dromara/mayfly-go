<template>
    <div>
        <el-drawer :append-to-body="false" v-model="visible" :before-close="cancel" size="50%" body-class="flex flex-col">
            <template #header>
                <DrawerHeader :header="props.title" :back="cancel">
                    <template #extra>
                        <EnumTag :enums="LogTypeEnum" :value="log?.type" class="mr-4.5" />
                    </template>
                </DrawerHeader>
            </template>

            <el-descriptions class="mb-2" :column="1" border v-if="extra">
                <el-descriptions-item v-for="(value, key) in extra" :key="key" :span="1" :label="key">{{ value }}</el-descriptions-item>
            </el-descriptions>

            <TerminalBody class="mb-2 flex-1" ref="terminalRef" />
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import TerminalBody from './TerminalBody.vue';
import { logApi } from '../../views/system/api';
import type { SysLog } from '@/views/system/types';
import { LogTypeEnum } from '@/views/system/enums';
import { useIntervalFn } from '@vueuse/core';
import EnumTag from '@/components/enum-tag/EnumTag.vue';

const props = withDefaults(
    defineProps<{
        title?: string;
    }>(),
    { title: 'Log' }
);

const visible = defineModel<boolean>('visible', { default: false });
const logId = defineModel<number>('logId', { default: 0 });

const terminalRef = ref<InstanceType<typeof TerminalBody> | null>(null);
const nowLine = ref(0);
const log = ref<SysLog | null>(null);

const extra = computed(() => {
    if (log.value?.extra) {
        try {
            return JSON.parse(log.value.extra);
        } catch {
            return null;
        }
    }
    return null;
});

// 定时获取最新日志
const { pause, resume } = useIntervalFn(() => {
    writeLog();
}, 500);

watch(
    () => logId.value,
    (logId: number) => {
        terminalRef.value?.clear();
        if (!logId) {
            return;
        }
        writeLog();
    }
);

const cancel = () => {
    visible.value = false;
    logId.value = 0;
    nowLine.value = 0;
    pause();
};

const writeLog = async () => {
    const log = await getLog();
    if (!log) {
        return;
    }
    writeLog2Term(log);

    // 如果不是还在执行中的日志，则暂停轮询
    if (log.type != LogTypeEnum.Running.value) {
        pause();
        return;
    }
    resume();
};

const writeLog2Term = (log: SysLog) => {
    if (!log) {
        return;
    }
    const lines = log.resp.split('\n');
    for (let line of lines.slice(nowLine.value)) {
        nowLine.value += 1;
        terminalRef.value?.writeln2Term(line);
    }
    terminalRef.value?.focus();
};

const getLog = async () => {
    if (!logId.value) {
        return;
    }
    const logRes = await logApi.detail.request({
        id: logId.value,
    });
    log.value = logRes;
    return logRes;
};
</script>

<style lang="scss"></style>
