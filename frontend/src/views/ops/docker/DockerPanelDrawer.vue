<template>
    <div>
        <el-drawer title="Docker" v-model="dialogVisible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="true" size="80%">
            <template #header>
                <DrawerHeader :header="props.host" :back="cancel">
                    <template #extra>
                        <div class="mr20"></div>
                    </template>
                </DrawerHeader>
            </template>

            <DockerPanel :host="props.host" />
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { defineAsyncComponent, ref, Ref } from 'vue';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';

const DockerPanel = defineAsyncComponent(() => import('./DockerPanel.vue'));

const props = defineProps({
    host: {
        type: String,
        required: true,
    },
});

const dialogVisible = defineModel<boolean>('visible');

const emit = defineEmits(['cancel']);

const cancel = () => {
    dialogVisible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
