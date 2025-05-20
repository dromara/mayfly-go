<template>
    <el-drawer
        :title="title"
        v-model="visible"
        :before-close="cancel"
        :destroy-on-close="true"
        :close-on-click-modal="false"
        size="80%"
        body-class="!p-2"
        header-class="!mb-2"
    >
        <template #header>
            <DrawerHeader :header="title" :back="cancel" />
        </template>

        <FlowDesign :disabled="props.disabled" :data="props.data" @save="(data) => emit('save', data)" />
    </el-drawer>
</template>

<script lang="ts" setup>
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import FlowDesign from './FlowDesign.vue';

const props = defineProps({
    disabled: {
        type: Boolean,
        default: false,
    },
    data: {
        type: [Object],
    },
    title: {
        type: String,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits(['cancel', 'save']);

const cancel = () => {
    visible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
