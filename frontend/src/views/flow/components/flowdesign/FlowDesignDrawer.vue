<template>
    <el-drawer
        :append-to-body="false"
        :title="title"
        v-model="visible"
        :before-close="cancel"
        :destroy-on-close="true"
        :close-on-click-modal="false"
        :size="FLOW_DRAWER.canvas"
        body-class="p-2!"
        header-class="mb-2!"
    >
        <template #header>
            <DrawerHeader :header="headerTitle" :back="cancel" />
        </template>

        <FlowDesign :disabled="props.disabled" :data="props.data" @save="(data) => emit('save', data)" />
    </el-drawer>
</template>

<script lang="ts" setup>
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import FlowDesign from './FlowDesign.vue';
import { computed, type PropType } from 'vue';
import { useI18n } from 'vue-i18n';
import { FLOW_DRAWER } from '@/views/flow/drawerSize';

const props = defineProps({
    disabled: {
        type: Boolean,
        default: false,
    },
    data: {
        type: [Object] as PropType<Record<string, unknown> | null>,
    },
    title: {
        type: String,
    },
});

const { t } = useI18n();

/**
 * 头部标题：调用方没传时也要有名字。
 *
 * 这里曾长期是空白头部（列表页只传 data/visible，title 无默认值），
 * 抽屉顶栏只剩一个返回箭头，用户不知道自己开的是哪个流程的设计
 */
const headerTitle = computed(() => props.title || t('flow.flowDesign'));

const visible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits(['cancel', 'save']);

const cancel = () => {
    visible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
