<template>
    <el-select
        remote
        :remote-method="getMsgTmpls"
        v-model="tmplId"
        filterable
        :placeholder="$t('msg.selectTmplPlaceholder')"
        v-bind="$attrs"
        :ref="(el: any) => props.focus && el?.focus()"
    >
        <el-option v-for="item in tmpls" :key="item.id" :label="item.name" :value="item.id">
            {{ item.code }}
            <el-divider direction="vertical" />
            {{ item.name }}
        </el-option>
    </el-select>
</template>

<script lang="ts" setup>
import { onMounted, ref } from 'vue';
import { tmplApi } from '../api';

const props = defineProps({
    // 是否获取焦点
    focus: {
        type: Boolean,
        default: false,
    },
});

const tmplId = defineModel('modelValue');

const tmpls: any = ref([]);

onMounted(() => {
    // 如果初始化时有tmplId，则需要获取对应消息模板信息，用于回显
    tmplApi.list.request({ id: tmplId.value }).then((res) => {
        tmpls.value = res.list;
    });
});

const getMsgTmpls = (code: any) => {
    tmplApi.list.request({ code }).then((res) => {
        tmpls.value = res.list;
    });
};
</script>
