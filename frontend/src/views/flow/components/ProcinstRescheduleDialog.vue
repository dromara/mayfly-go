<template>
    <el-dialog v-model="visible" :title="$t('flow.reschedule')" width="380px" append-to-body>
        <!-- 绑定统一时间格式字符串（model.JsonTime 约定的 “2006-01-02 15:04:05”），提交前原样发出，无需再转 ISO -->
        <el-date-picker
            v-model="picked"
            type="datetime"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
            :placeholder="$t('flow.plannedExecTimeTip')"
            class="w-full"
        />
        <template #footer>
            <el-button @click="visible = false">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" :loading="loading" @click="submit">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { ref, watch } from 'vue';
import { Msg } from '@/hooks/useI18n';
import { formatDate } from '@/common/utils/format';
import { procinstApi } from '../api';

const props = defineProps<{
    /** 目标工单 id */
    procinstId: number;
    /** 当前计划执行时间（ISO 串），打开时预填；空表示尚未定时 */
    current?: string | null;
}>();

const emit = defineEmits<{ submitted: [] }>();

const visible = defineModel<boolean>('visible', { default: false });

// 绑定统一格式的时间字符串（与 model.JsonTime 入参同一口径）；空串=未定时/改成立即执行
const picked = ref('');
const loading = ref(false);

// 每次打开按工单当前计划时间预填（后端回传可能是 ISO，统一转成展示/提交格式）
watch(visible, (opened) => {
    if (opened) {
        picked.value = props.current ? formatDate(props.current) : '';
    }
});

async function submit() {
    loading.value = true;
    try {
        // 直接发统一格式串（model.JsonTime 会按本地时区解析）；空串会被归一为零值=立即执行
        await procinstApi.rescheduleExec.request({ id: props.procinstId, plannedExecTime: picked.value || '' });
        Msg.operateSuccess();
        visible.value = false;
        emit('submitted');
    } finally {
        loading.value = false;
    }
}
</script>
