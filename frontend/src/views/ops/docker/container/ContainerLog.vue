<template>
    <div>
        <el-drawer title="logs" v-model="visible" @close="close" :destroy-on-close="true" :close-on-click-modal="true" size="60%">
            <template #header>
                <DrawerHeader :header="`${props.title}`" :back="() => (visible = false)">
                    <template #extra>
                        <div class="mr20"></div>
                    </template>
                </DrawerHeader>
            </template>

            <div class="flex flex-col flex-1">
                <el-row :gutter="10" class="mb-2">
                    <el-col :span="6">
                        <el-select @change="searchLog" v-model.number="state.tail">
                            <template #prefix>{{ $t('docker.lines') }}</template>
                            <el-option :value="100" :label="100" />
                            <el-option :value="200" :label="200" />
                            <el-option :value="500" :label="500" />
                            <el-option :value="1000" :label="1000" />
                        </el-select>
                    </el-col>

                    <el-col :span="6">
                        <el-checkbox @change="searchLog" border v-model="state.isWatch">
                            {{ $t('docker.follow') }}
                        </el-checkbox>
                    </el-col>
                </el-row>

                <RealLogViewer ref="realLogViewerRef" :ws-url="wsUrl" height="calc(100vh - 200px)" />
            </div>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { computed, reactive, useTemplateRef } from 'vue';
import RealLogViewer from '@/components/monaco/RealLogViewer.vue';
import { getContainerLogSocketUrl } from '../api';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';

const props = defineProps({
    id: {
        type: Number,
        default: '',
    },
    title: {
        type: String,
        default: '',
    },
    containerId: {
        type: String,
        default: '',
    },
});

const visible = defineModel<boolean>('visible');

const realLogViewerRef = useTemplateRef('realLogViewerRef');

const state = reactive({
    since: '',
    tail: 100,
    isWatch: true,
});

const wsUrl = computed(
    () => `${getContainerLogSocketUrl(props.id, props.containerId)}&tail=${state.tail}&follow=${state.isWatch ? '1' : '0'}&since=${state.since}`
);

const searchLog = () => {
    realLogViewerRef.value?.reload(wsUrl.value);
};

const close = () => {
    state.tail = 100;
    state.since = '';
    state.isWatch = true;
};
</script>

<style scoped></style>
