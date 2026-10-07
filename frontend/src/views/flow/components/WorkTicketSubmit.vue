<template>
    <el-button v-if="!hidden" link type="primary" @click="open">{{ $t('flow.submitTicket') }}</el-button>

    <!-- 复用「发起流程」抽屉：业务类型与语句由被拦下的操作带过来，用户只需确认资源与备注。
         不另写一套提单表单，是为了保证入口再多也走同一条校验与提交链路 -->
    <ProcInstEdit v-if="rendered" v-model="startForm" v-model:visible="visible" :title="title" @val-change="emit('submitted')" />
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import type { ProcInstStartForm, TicketPrefill } from '@/views/flow/types';
import ProcInstEdit from '@/views/flow/ProcInstEdit.vue';

/**
 * 「提交工单审批」入口。
 *
 * 被触发策略拦下时，用户原本要自己切到「我的工单 → 发起流程」，再重选资源、重贴同一条
 * SQL/命令。这里把已经确定下来的业务类型与语句直接带进提单表单，只剩资源与备注需要确认。
 */
const props = withDefaults(
    defineProps<{
        /** 业务场景，取值为后端注册的 bizType；由拦截处给出，本组件不猜默认场景 */
        bizType?: string;
        /** 预填的业务表单（SQL、命令、库等） */
        bizForm?: Record<string, unknown>;
        /** 预填的工单备注 */
        remark?: string;
        /** 隐藏内置按钮，只暴露 open() 给宿主自行触达 */
        hidden?: boolean;
    }>(),
    { bizType: '', bizForm: () => ({}), remark: '', hidden: false }
);

const emit = defineEmits<{ submitted: [] }>();

const { t } = useI18n();

const visible = ref(false);
// 首次打开才挂载抽屉：ProcInstEdit 内部会按业务类型拉取流程定义与审批节点，
// 每个失败结果都预挂一份会让页面一打开就发一串无意义请求
const rendered = ref(false);

const title = computed(() => t('flow.startProcess'));

const startForm = ref<ProcInstStartForm>({
    bizType: '',
    procdefId: 0,
    status: null,
    remark: '',
    bizKey: '',
    bizForm: {},
});

function open(prefill?: TicketPrefill) {
    // 预填内容优先取入参：宿主通常是「刚算好内容就调用 open()」，而 Vue 的 props 要等父组件
    // 本轮渲染完才传进来，此刻读 props 读到的是上一次的旧值，表现为抽屉里 SQL/命令全空
    const bizType = prefill?.bizType ?? props.bizType;
    const bizForm = prefill?.bizForm ?? props.bizForm;
    const remark = prefill?.remark ?? props.remark;

    rendered.value = true;
    startForm.value = {
        bizType,
        procdefId: 0,
        status: null,
        remark,
        bizKey: '',
        bizForm: { ...bizForm },
    };
    visible.value = true;
}

defineExpose({ open });
</script>
