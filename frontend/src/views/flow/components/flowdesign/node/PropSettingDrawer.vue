<template>
    <el-drawer
        :append-to-body="false"
        body-class="pt-2!"
        header-class="mb-2!"
        :title="title"
        v-model="visible"
        :before-close="onCancel"
        :destroy-on-close="true"
        :close-on-click-modal="false"
        :size="FLOW_DRAWER.node"
    >
        <template #header>
            <DrawerHeader :header="headerTitle" :back="onCancel" />
        </template>

        <el-form ref="propSettingFormRef" :model="form" label-position="top" :disabled="props.disabled">
            <el-form-item ref="nameRef" :label="$t('common.name')" :rules="[Rules.requiredInput('common.name')]">
                <el-input v-model="name" clearable></el-input>
            </el-form-item>

            <component
                v-if="props.node"
                ref="formItemsRef"
                :is="getCustomNode(props.node.type)?.propSettingComp"
                v-model="form"
                :disabled="disabled"
                :nodes="nodes"
                :node="node"
            >
                <template v-slot:[key]="data" v-for="(item, key) in $slots">
                    <slot :name="key" v-bind="data || {}"></slot>
                </template>
            </component>
        </el-form>

        <template #footer>
            <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
            <el-button v-if="!props.disabled" type="primary" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-drawer>
</template>

<script lang="ts" setup>
import { computed, watch, ref, useTemplateRef, type PropType } from 'vue';
import { useI18n } from 'vue-i18n';
import { cloneNodeProperties } from './nodeProps';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { useI18nFormValidate, useI18nPleaseInput } from '@/hooks/useI18n';
import { Rules } from '@/common/rule';
import LogicFlow from '@logicflow/core';
import { getCustomNode } from '.';
import { notEmpty } from '@/common/assert';
import { FLOW_DRAWER } from '@/views/flow/drawerSize';

const { t } = useI18n();

const props = defineProps({
    data: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
    disabled: {
        type: Boolean,
        default: false,
    },
    node: {
        type: Object as PropType<LogicFlow.NodeData | LogicFlow.EdgeData | null>,
        default: null,
    },
    nodes: {
        type: Array as PropType<Array<LogicFlow.NodeData | LogicFlow.EdgeData>>,
        default: () => [],
    },
    lf: {
        type: LogicFlow,
        default: null,
    },
});

const propSettingFormRef = useTemplateRef('propSettingFormRef');
const formItemsRef = useTemplateRef<{ confirm?: () => void } | null>('formItemsRef');

/** 节点属性抽屉的标题：由画布组件调用，调用方通常不传，缺省也要能自解释 */
const headerTitle = computed(() => props.title || t('flow.nodeProperty'));

const visible = defineModel<boolean>('visible', { default: false });

// 节点名
const name = ref('');

// 节点props表单信息
const form = ref<Record<string, unknown>>({});

watch(
    () => props.node,
    (n) => {
        if (!n) {
            return;
        }
        name.value = n.text instanceof Object ? n.text.value : (n.text ?? '');
        // 深拷贝：条件树等嵌套对象若与画布节点共享引用，编辑中途就地改动会让「取消」无法回滚，
        // 下一次保存流程还会把这些已取消的改动静默写进流程定义
        form.value = cloneNodeProperties(n.properties);
    }
);

const onConfirm = async () => {
    // 连线允许空名：画布上的边默认本就无文本，跳转条件才是边属性的真实内容；
    // 名称必填会让存量未命名连线永远保存不了条件修改（点确定只报「请输入名称」）
    const isEdge = !!props.node && 'sourceNodeId' in props.node;
    if (!isEdge) {
        notEmpty(name.value, useI18nPleaseInput('common.name'));
    }
    if (formItemsRef.value?.confirm) {
        formItemsRef.value.confirm();
    }
    await useI18nFormValidate(propSettingFormRef);
    const nodeId = props.node?.id;
    if (!nodeId) {
        return;
    }
    // 更新流程节点上的文本内容
    props.lf.updateText(nodeId, name.value);
    props.lf.setProperties(nodeId, form.value);
    onCancel();
};

const onCancel = () => {
    visible.value = false;
};
</script>
<style lang="scss"></style>
