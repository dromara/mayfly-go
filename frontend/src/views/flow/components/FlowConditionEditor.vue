<template>
    <div class="flow-condition-editor">
        <!-- 预置行不重复渲染标题：宿主面板已经给出「审批模式 / 跳转条件」这类标签 -->
        <div v-if="showPresets && presets.length" class="preset-row">
            <el-tooltip v-for="preset in presets" :key="preset.key" :content="$t(preset.descriptionKey)" placement="top">
                <el-button link type="primary" size="small" :disabled="disabled" :class="{ active: isActivePreset(preset) }" @click="applyPreset(preset)">
                    {{ $t(preset.titleKey) }}
                </el-button>
            </el-tooltip>
        </div>

        <div v-if="!scenario" class="editor-pending">{{ $t('flow.policy.loading') }}</div>
        <!-- 未配置时给一个明确的添加入口而不是预先塞一个空分组：
             空分组会被后端判非法，一打开面板就飘红等于设计缺陷 -->
        <el-button v-else-if="!current" :disabled="disabled" size="small" icon="plus" plain @click="startEditing">
            {{ $t('flow.condition.addCondition') }}
        </el-button>
        <template v-else>
            <PolicyConditionNode :node="current" :scenario="scenario" :disabled="disabled" @remove="clear" />
            <el-button v-if="!disabled" link type="danger" size="small" @click="clear">{{ $t('flow.condition.clear') }}</el-button>
        </template>

        <div v-if="summary" class="editor-preview">
            <span class="preview-label">{{ $t('flow.policy.preview') }}</span>
            <span class="preview-text">{{ summary }}</span>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { PolicyConditionNode } from '@/components/policy-builder';
import { createConditionNode, createGroupNode, describeNode, type ConditionPreset, type PolicyScenario, type RuleNode } from '@/components/policy-builder';

/**
 * 流程内部条件编辑器：连线跳转条件与用户任务完成条件共用。
 *
 * 两类条件与触发策略是同一种结构（条件树 + 同一套操作符），因此这里直接复用 policy-builder
 * 的条件树组件，只把字段字典换成流程实例变量那份；新增可比较的流程变量只需在后端注册字段，
 * 本组件与 policy-builder 都无需改动。
 */
const props = withDefaults(
    defineProps<{
        scenario: PolicyScenario | null;
        disabled?: boolean;
        /** 是否展示审批模式预置（只签 / 会签 / 过半）：仅节点完成条件需要，连线条件套它会毫无意义 */
        showPresets?: boolean;
    }>(),
    { disabled: false, showPresets: false }
);

// 条件树（v-model）。只要宿主给过一棵树就渲染它：空壳子要能被看见并就地补条件，
// 否则点「添加条件」只会得到一个判非法又看不见的空分组（死路）
const current = defineModel<RuleNode | null>({ default: null });

const { t } = useI18n();

const summary = computed(() => describeNode(current.value, props.scenario, t));

// 预置由后端随 schema 下发：预置里引用的字段 key 与字段字典同源，改字段名不会留下对不上的旧预设
const presets = computed<ConditionPreset[]>(() => props.scenario?.conditionPresets ?? []);

// 预置高亮按语义归一化比较：落库条件树的键序（field,kind,op,value）与预置键序（kind,field,op,value）
// 不同，JSON.stringify 全串比较会在刷新回读后恒不高亮；按语义序转成数组即与键序解耦
const normalizeNode = (node: RuleNode | null | undefined): unknown => {
    if (!node) {
        return null;
    }
    if (node.kind === 'group') {
        return [node.kind, node.logic, (node.items ?? []).map(normalizeNode)];
    }
    if (node.kind === 'segment') {
        return [node.kind, node.ref];
    }
    return [node.kind, node.field, node.op, node.value];
};

const isActivePreset = (preset: ConditionPreset) => JSON.stringify(normalizeNode(preset.ruleNode)) === JSON.stringify(normalizeNode(current.value));

function applyPreset(preset: ConditionPreset) {
    // 预置来自后端 schema 的响应式对象，直接赋引用会让用户改条件树时改到共享的预置本体；
    // structuredClone 又克隆不了 Proxy，因此走 JSON 深拷贝（规则树是纯数据）
    current.value = JSON.parse(JSON.stringify(preset.ruleNode)) as RuleNode;
}

function clear() {
    current.value = null;
}

function startEditing() {
    // 建组即给一行：空分组会被保存校验判非法，点一下按钮就飘红等于设计缺陷
    const node = createGroupNode('all');
    node.items = [createConditionNode(props.scenario?.fields[0])];
    current.value = node;
}
</script>

<style lang="scss" scoped>
.flow-condition-editor {
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 100%;

    .preset-row {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 4px 10px;
    }

    .preset-row .active {
        font-weight: 600;
    }

    .editor-pending {
        font-size: 12px;
        color: var(--el-text-color-secondary);
    }

    .editor-preview {
        font-size: 12px;
        line-height: 18px;
        color: var(--el-text-color-secondary);
        word-break: break-word;
    }

    .preview-label {
        margin-right: 6px;
        padding: 1px 6px;
        border-radius: 4px;
        background: var(--el-fill-color-light);
    }
}
</style>
