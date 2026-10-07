<template>
    <div v-if="items.length" class="policy-history">
        <div class="history-head">
            <span class="history-title">{{ $t('flow.policyHistory.title') }}</span>
            <el-button link type="primary" size="small" @click="collapsed = !collapsed">
                {{ collapsed ? $t('flow.policyHistory.expand') : $t('flow.policyHistory.collapse') }}
            </el-button>
        </div>

        <el-timeline v-if="!collapsed" class="history-timeline">
            <el-timeline-item v-for="item in items" :key="item.id" :timestamp="formatDate(item.createTime)" :type="actionTagType(item.action)" placement="top">
                <div class="history-item">
                    <div class="item-head">
                        <EnumTag :enums="ProcdefPolicyAction" :value="item.action" />
                        <span class="item-who">{{ item.creator }}</span>
                    </div>
                    <ul class="item-lines">
                        <li v-for="(line, index) in item.lines" :key="index" :class="`diff-${line.kind}`">
                            <span class="diff-mark">{{ DIFF_MARKS[line.kind] }}</span>
                            <span class="diff-text">{{ line.text }}</span>
                        </li>
                    </ul>
                </div>
            </el-timeline-item>
        </el-timeline>
    </div>
</template>

<script lang="ts" setup>
import { ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { diffPolicy, type PolicyDiffLine, type PolicySchema } from '@/components/policy-builder';
import EnumTag from '@/components/enum-tag/EnumTag.vue';
import { formatDate } from '@/common/utils/format';
import { ProcdefPolicyAction } from '@/views/flow/enums';
import { procdefApi } from '@/views/flow/api';
import type { ProcdefPolicyHis } from '@/views/flow/types';

/**
 * 触发策略变更时间线：谁在什么时候把哪条规则改成了什么。
 *
 * 差异在前端按前后两份快照现算，后端只存快照：规则的展示文案会随字段字典与 i18n 演进，
 * 把算好的文案落库等于把它焊死在历史里，将来加一个检查项就会让旧条目的描述说不通。
 * 没有任何变化（如只改名、只换标签）的保存不产生记录，因此空列表直接不渲染这块区域。
 */
const props = defineProps<{ procdefId: number; schema: PolicySchema | null }>();

const { t } = useI18n();

const collapsed = ref(true);
const items = ref<{ id: number; action: number; creator: number | string; createTime: string; lines: PolicyDiffLine[] }[]>([]);

const DIFF_MARKS: Record<PolicyDiffLine['kind'], string> = { added: '+', changed: '~', removed: '-' };

// 时间线节点的着色与 EnumTag 的取值保持一致：新建绿、修改蓝、删除红
const ACTION_TAG_TYPES: Record<number, 'success' | 'primary' | 'danger'> = {
    [ProcdefPolicyAction.Create.value]: 'success',
    [ProcdefPolicyAction.Update.value]: 'primary',
    [ProcdefPolicyAction.Delete.value]: 'danger',
};

const actionTagType = (action: number) => ACTION_TAG_TYPES[action] ?? 'primary';

async function load() {
    if (!props.procdefId || !props.schema) return;
    // 该流程定义没有过策略变更时，接口返回的 data 是 null：声明上的数组在运行时并不成立，
    // 少一次兜底就是每次打开编辑抽屉一条未捕获 rejection（界面不崩，但控制台常年带错）
    const records = (await procdefApi.policyHistory.request({ id: props.procdefId })) ?? [];
    items.value = records
        .map((record) => ({
            id: record.id,
            action: record.action,
            creator: record.creator,
            createTime: record.createTime,
            lines: diffPolicy(record.before, record.after, props.schema, t),
        }))
        // 只改名称、换标签的保存也会进来，时间线若列出「什么都没改」的条目，
        // 真正的改动反而被淹没在里面
        .filter((record) => record.lines.length > 0);
}

watch(() => [props.procdefId, props.schema], load, { immediate: true });
</script>

<style lang="scss" scoped>
.policy-history {
    margin-top: 10px;
    padding-top: 8px;
    border-top: 1px dashed var(--el-border-color-lighter);
}

.history-head {
    display: flex;
    align-items: center;
    gap: 8px;
}

.history-title {
    font-size: 12px;
    color: var(--el-text-color-secondary);
}

.history-timeline {
    margin-top: 10px;
    padding-left: 4px;
}

.history-item {
    .item-head {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 4px;
    }

    .item-who {
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }

    .item-lines {
        margin: 0;
        padding-left: 0;
        list-style: none;
        font-size: 12px;
        line-height: 20px;
    }

    .diff-text {
        word-break: break-word;
    }

    .diff-added .diff-text {
        color: var(--app-color-success-readable);
    }

    .diff-removed .diff-text {
        color: var(--app-color-danger-readable);
        text-decoration: line-through;
    }

    .diff-changed .diff-text {
        color: var(--app-color-warning-readable);
    }

    .diff-mark {
        display: inline-block;
        width: 14px;
        font-weight: 600;
    }
}
</style>
