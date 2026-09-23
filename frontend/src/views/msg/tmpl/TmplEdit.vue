<template>
    <div>
        <auto-form-drawer v-model:visible="visible" :title="title" :items="items" :data="editData" size="40%" :confirm-api="saveFormExec" @submitted="(form) => emit('success', form)" @cancel="emit('cancel')" @opened="onOpened">
            <!-- 消息渠道多选（选项需展示类型/编码/名称组合信息，自定义插槽） -->
            <template #channelIds="{ form: f }">
                <el-select v-model="f.channelIds" multiple clearable filterable class="w-full!">
                    <el-option v-for="item in channels" :key="item.id" :label="item.name" :value="item.id">
                        {{ $t(EnumValue.getLabelByValue(ChannelTypeEnum, item.type)) }}
                        <el-divider direction="vertical" />
                        {{ item.code }}
                        <el-divider direction="vertical" />
                        {{ item.name }}
                    </el-option>
                </el-select>
            </template>
            <!-- 模板内容（monaco 语言随 msgType 联动，自定义插槽） -->
            <template #tmpl="{ form: f }">
                <MonacoEditor class="w-full!" height="200px" v-model="f.tmpl" :language="EnumValue.getLabelByValue(TmplTypeEnum, f.msgType ?? '')" />
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import EnumValue from '@/common/Enum';
import { AutoFormDrawer, defineFormItems } from '@/components/auto-form';
import { useAutoFormModel } from '@/hooks/useAutoFormModel';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { computed, ref, watch, type PropType } from 'vue';
import { channelApi, tmplApi } from '../api';
import { ChannelStatusEnum, ChannelTypeEnum, TmplStatusEnum, TmplTypeEnum } from '../enums';
import type { MsgChannel, MsgTemplate } from '@/views/system/msg/types';

/** 消息模板编辑表单类型 */
interface TmplForm extends Omit<Partial<MsgTemplate>, 'id' | 'name' | 'msgType' | 'status'> {
    id?: number | null;
    name?: string | null;
    msgType?: string | number;
    status?: string | number;
    channelIds?: number[];
}

const props = defineProps({
    form: {
        type: Object as PropType<MsgTemplate | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['cancel', 'success']);

const visible = defineModel<boolean>('visible', { default: false });

/** 表单声明（defineFormItems<TmplForm>，渲染 + 校验唯一数据源；channelIds/tmpl 为自定义插槽） */
const items = defineFormItems<TmplForm>([
    { prop: 'name', label: 'msg.name', required: true },
    { prop: 'status', label: 'common.status', type: 'enum', enums: ChannelStatusEnum },
    { prop: 'remark', label: 'common.remark', type: 'textarea' },
    { prop: 'channelIds', label: 'msg.msgChannel', slot: 'channelIds' },
    { prop: 'msgType', label: 'common.type', type: 'enum', enums: TmplTypeEnum, required: true },
    { prop: 'title', label: 'msg.title' },
    { prop: 'tmpl', label: 'msg.tmpl', type: 'monaco', required: true, tooltip: 'msg.msgTmplTooltip', slot: 'tmpl' },
]);

const defaultForm = (): TmplForm => {
    return {
        id: null,
        name: null,
        msgType: TmplTypeEnum.Text.value,
        title: '',
        tmpl: '',
        status: TmplStatusEnum.Enable.value,
        remark: '',
        channelIds: [],
        extra: {},
    };
};

const channels = ref<MsgChannel[]>([]);

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<TmplForm>(() => {
    return props.form ? { ...defaultForm(), ...props.form } : defaultForm();
});

const { isFetching: saveBtnLoading, execute: saveFormExec } = tmplApi.save.useApi();

// 抽屉打开时加载渠道选项
watch(visible, (v) => {
    if (!v) {
        return;
    }
    channelApi.list.request({ pageNum: 1, pageSize: 200 }).then((res) => {
        channels.value = res?.list ?? [];
    });
});

/** 回填完成后异步补充关联渠道（列表行数据不携带关联关系，需按模板 id 单查） */
const { openedWith } = useAutoFormModel<TmplForm>();

const onOpened = openedWith((form) => {
    if (!props.form) {
        return;
    }
    tmplApi.relateChannels.request({ id: props.form.id }).then((res) => {
        form.channelIds = res.map((item: MsgChannel) => item.id);
    });
});

// 统一提交：confirmApi 由 AutoFormDrawer 内置逻辑驱动（校验 → 保存 → 成功提示 → submitted → 关闭抽屉，全程 loading 防重复提交）
</script>
<style lang="scss"></style>
