<template>
    <el-drawer
        body-class="!pt-2"
        header-class="!mb-2"
        :title="title"
        v-model="visible"
        :before-close="onCancel"
        :destroy-on-close="true"
        :close-on-click-modal="false"
        size="40%"
    >
        <template #header>
            <DrawerHeader :header="title" :back="onCancel" />
        </template>

        <el-form ref="propSettingFormRef" :model="form" label-position="top" :disabled="props.disabled">
            <el-form-item ref="nameRef" :label="$t('common.name')" :rules="[Rules.requiredInput('common.name')]">
                <el-input v-model="name" clearable></el-input>
            </el-form-item>

            <component ref="formItemsRef" :is="getCustomNode(props.node.type)?.propSettingComp" v-model="form" :disabled="disabled" :nodes="nodes" :node="node">
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
import { watch, ref, useTemplateRef } from 'vue';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { useI18nFormValidate, useI18nPleaseInput } from '@/hooks/useI18n';
import { Rules } from '@/common/rule';
import LogicFlow from '@logicflow/core';
import { getCustomNode } from '.';
import { notEmpty } from '@/common/assert';

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
        type: Object,
        default: {},
    },
    nodes: {
        type: Array,
        default: () => [],
    },
    lf: {
        type: LogicFlow,
        default: null,
    },
});

const propSettingFormRef = useTemplateRef('propSettingFormRef');
const formItemsRef: any = useTemplateRef('formItemsRef');

const visible = defineModel<boolean>('visible', { default: false });

// 节点名
const name = ref('');

// 节点props表单信息
const form = ref({});

watch(
    () => props.node,
    (n) => {
        if (!n) {
            return;
        }
        name.value = n.text instanceof Object ? n.text.value : n.text;
        form.value = { ...n.properties };
    }
);

const onConfirm = async () => {
    notEmpty(name.value, useI18nPleaseInput('common.name'));
    if (formItemsRef.value?.confirm) {
        formItemsRef.value?.confirm();
    }
    await useI18nFormValidate(propSettingFormRef);
    const nodeId = props.node.id;
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
