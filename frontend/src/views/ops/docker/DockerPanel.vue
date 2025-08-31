<template>
    <div class="card h-full">
        <el-tabs v-model="activeName" @tab-change="handleTabChange">
            <el-tab-pane :label="$t('docker.container')" :name="containerTab">
                <ContainerList :host="props.host" />
            </el-tab-pane>

            <el-tab-pane :label="$t('docker.image')" :name="imageTab">
                <ImageList v-if="activeName == imageTab" :host="props.host" />
            </el-tab-pane>
        </el-tabs>
    </div>
</template>
<script lang="ts" setup>
import { toRefs, reactive, onMounted, defineAsyncComponent } from 'vue';

const ContainerList = defineAsyncComponent(() => import('./container/ContainerList.vue'));
const ImageList = defineAsyncComponent(() => import('./image/ImageList.vue'));

const props = defineProps({
    host: {
        type: String,
        required: true,
    },
});

const containerTab = 'containerTab';
const imageTab = 'imageTab';

const state = reactive({
    activeName: containerTab,
    cmdConfs: [],
});

const { activeName } = toRefs(state);

onMounted(async () => {
    state.activeName = containerTab;
});

const handleTabChange = (tabName: any) => {};
</script>
