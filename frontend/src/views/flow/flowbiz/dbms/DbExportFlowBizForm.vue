<template>
    <div class="flex flex-col gap-2">
        <el-alert :title="$t('flow.exportFormTip')" type="info" :closable="false" show-icon />
        <el-descriptions :column="3" border label-width="110">
            <el-descriptions-item :span="1" :label="$t('flow.exportDb')">{{ bizForm.dbName || '-' }}</el-descriptions-item>
            <el-descriptions-item :span="1" :label="$t('flow.exportContent')">{{ dumpTypeLabel }}</el-descriptions-item>
            <el-descriptions-item :span="1" :label="$t('flow.exportTableCount')">{{ bizForm.tables?.length ?? 0 }}</el-descriptions-item>
            <el-descriptions-item :span="3" :label="$t('flow.exportTables')">
                <div class="max-h-24 overflow-auto text-xs">{{ (bizForm.tables ?? []).join(', ') || '-' }}</div>
            </el-descriptions-item>
        </el-descriptions>
    </div>
</template>

<script lang="ts" setup>
/**
 * 导出工单的**提单**表单。
 *
 * 它不收集参数：库、表清单与导出类型都来自「申请导出」那一刻的上下文，用户在这里只确认
 * 申请的是什么范围（审批人要能看懂自己批了什么）。
 *
 * 唯一必须做的事是**上报资源**：宿主靠 `changeResourceCode` 反查该库绑定的流程定义，
 * 不上报则抽屉停在「不存在审批节点」、确定按钮永久不可点——表现为“点了申请导出没反应”。
 */
import { computed, nextTick, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { dbApi } from '@/views/ops/db/api';
import { TagResourceTypeEnum } from '@/common/commonEnum';

const props = defineProps({
    // 与宿主约定的业务表单模型；范围字段由 openTicket 预填
    bizForm: {
        type: Object,
        default: () => ({}),
    },
});

// 用类型式 defineEmits：数组式的文件数被仓库棘轮测试卡住（只允许存量不回涨）
const emit = defineEmits<{ (e: 'changeResourceCode', resourceType: number, code: string): void }>();
const { t, te } = useI18n();

// 父组件用 v-model:bizForm 传入同一对象，这里直接用 props 引用即可（本表单不修改它）
const bizForm = computed(() => props.bizForm as Record<string, any>);

const dumpTypeLabel = computed(() => {
    const key = `flow.exportDumpType${bizForm.value?.dumpType ?? '3'}`;
    return te(key) ? t(key) : '-';
});

/**
 * 按 dbId 反查资源编码并上报。
 *
 * 预填的场景没有“选库”动作，必须主动上报一次；反查失败不阻断（用户仍可重选库），
 * 只是这一次会在抽屉里看到“不存在审批节点”的提示
 */
const announceResource = async () => {
    const code = await resolveDbCode();
    if (!code) {
        return;
    }
    // 等一个 tick：宿主要先完成表单接管才能接住这次变更
    nextTick(() => emit('changeResourceCode', TagResourceTypeEnum.Db.value, code));
};

async function resolveDbCode(): Promise<string> {
    const dbId = Number(bizForm.value?.dbId ?? 0);
    if (dbId <= 0) {
        return '';
    }
    if (bizForm.value?.dbCode) {
        return String(bizForm.value.dbCode);
    }
    try {
        const res = await dbApi.dbs.request({ id: dbId });
        return res.list?.[0]?.code ?? '';
    } catch {
        return '';
    }
}

onMounted(announceResource);

/**
 * 宿主（ProcInstEdit）在提交前会调用这个钩子；缺失时是 TypeError 被 catch 成
 * 「请正确填写信息」，表现为“字段都没问题但就是提交不了”。
 * 本表单没有可编辑字段（范围由入口确定），因此直接放行。
 */
const validateBizForm = async () => true;

// 与其它业务表单保持同一暴露面：宿主重置时可能调用，缺失同样是 TypeError
const resetBizForm = async () => true;

defineExpose({ validateBizForm, resetBizForm });
</script>
