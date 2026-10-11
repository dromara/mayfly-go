<template>
    <el-select
        remote
        :remote-method="getMsgTmpls"
        v-model="tmplValue"
        filterable
        :disabled="!canReadTmpl"
        :placeholder="canReadTmpl ? $t('msg.selectTmplPlaceholder') : $t('msg.tmplNoReadPerm')"
        v-bind="$attrs"
        :ref="(el: any) => props.focus && el?.focus()"
    >
        <el-option v-for="item in tmpls" :key="item.id" :label="item.name" :value="item[props.valueField]">
            {{ item.code }}
            <el-divider direction="vertical" />
            {{ item.name }}
        </el-option>
    </el-select>
</template>

<script lang="ts" setup>
import { onMounted, ref, type PropType } from 'vue';
import { hasPerm } from '@/components/auth/auth';
import { tmplApi } from '../api';
import type { MsgTemplate } from '@/views/system/msg/types';

/**
 * 读取消息模板列表的权限码
 *
 * 无权时不能发请求：接口以业务码 501 拒绝，全局请求层会直接把当前页跳到无权限页，
 * 用户只是想配个通知却被踢出当前页面。
 */
const TMPL_READ_PERM = 'msg:tmpl:base';

const props = defineProps({
    // 是否获取焦点
    focus: {
        type: Boolean,
        default: false,
    },
    /**
     * 选中值绑定的模板字段：id（主键）或 code（模板编码）
     *
     * 消息模板既可按主键关联（如流程定义通知），也可按编码关联（如计划任务按编码发送通知），
     * 两种关联方式统一由本选择器承载：业务侧一律选择模板，不手填编码或 id。
     */
    valueField: {
        type: String as PropType<'id' | 'code'>,
        default: 'id',
    },
});

const tmplValue = defineModel<string | number | null>('modelValue');

const tmpls = ref<MsgTemplate[]>([]);

const canReadTmpl = hasPerm(TMPL_READ_PERM);

/**
 * 按关键字拉取候选模板（编码模糊匹配）
 *
 * remote 模式下展开下拉框也会以空关键字回调，故空关键字即取首页模板列表，可直接浏览选择；
 * 有初始值时以初始值为关键字，保证选中项存在于候选中（否则下拉框只能显示原始值而非模板名称）。
 */
const loadTmpls = (keyword?: string | number | null) => {
    if (!canReadTmpl) {
        return;
    }
    const query = props.valueField === 'code' ? { code: keyword } : { id: keyword };
    tmplApi.list.request(query).then((res) => {
        tmpls.value = res?.list ?? [];
    });
};

onMounted(() => {
    loadTmpls(tmplValue.value);
});

const getMsgTmpls = (keyword: string) => loadTmpls(keyword);
</script>
