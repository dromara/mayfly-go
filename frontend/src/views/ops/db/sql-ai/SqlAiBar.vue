<template>
    <div
        v-show="visible"
        class="ai-bar absolute top-2 right-2 z-30 flex flex-col gap-2 rounded-md border p-2 shadow-lg"
        role="dialog"
        :aria-label="t('db.draft.entry')"
    >
        <!-- 意图：chips 而非下拉，一次点击到位；只列当前材料下可用的 -->
        <div class="flex flex-wrap items-center gap-1">
            <button
                v-for="item in intents"
                :key="item.id"
                type="button"
                class="intent-chip"
                :class="{ 'intent-chip-active': item.id === intent }"
                @click="onIntentChange(item.id)"
            >
                {{ t(item.titleKey) }}
            </button>
            <span class="ml-auto text-xs text-gray-400">{{ t('db.draft.barMaterial', { selection: materials.selection.length, statement: materials.statement.length }) }}</span>
            <!-- 常驻关闭入口：Esc 已经与焦点无关，但鼠标用户需要一个看得见的「不要了」 -->
            <button type="button" class="bar-close" :title="t('common.close')" @click="onClose">
                <SvgIcon name="Close" class="text-xs" />
            </button>
        </div>

        <el-input
            ref="inputRef"
            v-model="prompt"
            type="textarea"
            :autosize="{ minRows: 1, maxRows: 4 }"
            resize="none"
            :maxlength="MAX_PROMPT_LENGTH"
            :placeholder="placeholder"
            @keydown.enter.exact.prevent="submit"
            @keydown.tab.prevent="onTab"
        />

        <!-- 键位说明常驻：浮层没有按钮之外的第二个入口，忘了键位就等于卡在原地 -->
        <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-400">
            <span>{{ t('db.draft.barKeys') }}</span>
            <el-checkbox v-model="includePlans" size="small" :label="t('db.draft.includePlans')" />
            <span v-if="zeroInputReady" class="text-blue-500">{{ t('db.draft.barZeroInput') }}</span>
        </div>

        <!-- 注释补全开关：工具条上不常驻，因为它是「不打字也会扣模型费」的唯一入口，
             默认关闭、需要显式打开；放在面板底部是因为这里就是 AI 偏好的家 -->
        <div class="flex items-center gap-2 text-xs text-gray-400">
            <el-switch :model-value="inlineDraft" size="small" @update:model-value="emits('update:inlineDraft', $event === true)" />
            <span>{{ t('db.draft.inlineToggle') }}</span>
            <el-tooltip :show-after="600" effect="dark" :content="t('db.draft.inlineTip')" placement="top">
                <SvgIcon name="InfoFilled" class="text-xs" />
            </el-tooltip>
        </div>

        <!-- 常用需求模板：只在「按描述生成」下出现，点一下就有句子，改几个词即可 -->
        <div v-if="showTemplates" class="flex flex-wrap gap-1">
            <button v-for="item in templatePrompts" :key="item" type="button" class="soft-chip" @click="onPickPrompt(item)">{{ item }}</button>
        </div>

        <!-- 最近使用：同一库里的需求高度重复，复用比重新打字快 -->
        <div v-if="recentPrompts.length" class="flex flex-wrap items-center gap-1">
            <span class="text-xs text-gray-400">{{ t('db.draft.barRecent') }}</span>
            <button v-for="item in recentPrompts" :key="item" type="button" class="soft-chip" @click="onPickPrompt(item)">{{ item }}</button>
        </div>

        <!-- 等待反馈：一次生成要经过「找表 → 生成 → 校验」，只转圈用户不知道在等什么、也不知道能否中断 -->
        <div v-if="draft.state.loading" class="flex items-center gap-2 text-xs text-gray-500">
            <SvgIcon class="is-loading" name="Loading" />
            <span>{{ t('db.draft.loadingHint', { seconds: elapsed }) }}</span>
            <el-button link type="primary" size="small" @click="onCancelWaiting">{{ t('common.cancel') }}</el-button>
        </div>

        <el-alert v-else-if="draft.state.error" :title="draft.state.error" type="error" :closable="false" show-icon />
        <el-alert v-else-if="draft.state.timedOut" :title="t('db.draft.timeout')" type="warning" :closable="false" show-icon />

        <div v-else-if="draft.state.result" class="flex flex-col gap-1">
            <div class="flex flex-wrap items-center gap-2">
                <el-tag v-if="draft.state.result.verification !== ''" :type="verificationTagType(draft.state.result.verification)" size="small" disable-transitions>
                    {{ t(`db.draft.verification_${draft.state.result.verification}`) }}
                </el-tag>
                <el-tag type="info" size="small" disable-transitions>{{ draft.state.result.dialect }}</el-tag>
                <span class="text-xs text-gray-500">
                    {{ t('db.draft.calls', { repairs: draft.state.result.repairs, tokens: draft.state.result.usage?.totalTokens ?? 0 }) }}
                </span>
            </div>

            <!-- 散文产物（解释）没有可采纳的语句，只读展示 -->
            <div v-if="prose" class="max-h-40 overflow-auto whitespace-pre-wrap break-words rounded border p-2 text-xs leading-5">{{ prose }}</div>
            <pre v-else-if="adoptText" class="max-h-40 overflow-auto whitespace-pre-wrap break-all rounded border p-2 font-mono text-xs leading-5">{{ adoptText }}</pre>

            <div v-for="warning in draft.state.result.warnings ?? []" :key="warning" class="text-xs text-orange-500">{{ warning }}</div>
            <!-- 校验只覆盖表列存在与权限，覆盖不到取值语义；结论标签不能替代业务确认 -->
            <div v-if="draft.state.result.verification !== ''" class="text-xs text-gray-400">{{ t('db.draft.verificationScopeTip') }}</div>

            <div class="flex items-center justify-between gap-2">
                <span class="text-xs text-gray-400">{{ t('db.draft.noExecuteTip') }}</span>
                <div class="flex items-center gap-2">
                    <el-button size="small" @click="emits('detail')">{{ t('db.draft.barDetail') }}</el-button>
                    <el-button v-if="canAdopt" size="small" type="primary" plain @click="onAdopt">{{ t('db.draft.barAdopt') }}</el-button>
                </div>
            </div>
        </div>
    </div>
</template>

<script lang="ts" setup>
/**
 * SQL 助手面板（非模态）。
 *
 * 为什么钉在编辑器右上角而不是跟着光标：光标锚定的浮层会盖住正要对照的那段 SQL，
 * 而「贴在选区下方」还需要在滚动、换行、改字号时不断重算坐标，一次漏算就飘到无关代码上。
 * 固定位置后，面板始终在工具条入口正下方，SQL 文本永远不被它盖住；
 * 打开方式（工具条按钮 / 右键菜单 / ⌘K）只决定意图与材料，不改变面板位置。
 *
 * 状态不在这里持有：`draft` 由父级注入，详情面板消费同一份结果，避免出现
 * 「弹窗和面板各存一份草稿」的双真源；写回也不在这里做，本组件不认识 monaco 编辑器。
 */
import { computed, nextTick, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { ElMessage } from 'element-plus';
import { canRunWithoutPrompt, listDraftIntents } from './draftIntents';
import {
    MAX_PROMPT_LENGTH,
    buildDraftPayload,
    canWriteback,
    isEmptyDraft,
    joinStatements,
    pushRecentPrompt,
    readRecentPrompts,
    validateDraftInput,
    verificationTagType,
} from './draftModel';
import type { DraftIntentId } from './types';
import type { DraftIntentDef } from './draftIntents';
import type { EditorMaterials } from './draftEditor';
import type { DraftHandle } from './useSqlDraft';

const props = defineProps<{
    dbId: number;
    dbName: string;
    /** 打开那一刻快照的材料：生成期间编辑器内容变了要靠它复现「当时问的是什么」 */
    materials: EditorMaterials;
    /** 与详情面板共用的草稿状态（同一份结果，两个视图） */
    draft: DraftHandle;
    /**
     * 打开即生成：右键菜单里用户已经点明了意图，再要求按一次回车就是把「一步」变「三步」。
     * 只在零输入可成立的意图上生效，否则仍是「等用户写完描述」。
     */
    runOnOpen?: boolean;
    /** 注释行补全开关（状态与持久化由编辑器持有：provider 的 scope 只能在编辑器侧同步） */
    inlineDraft?: boolean;
}>();

const visible = defineModel<boolean>({ default: false });
const intent = defineModel<DraftIntentId>('intent', { required: true });

const emits = defineEmits<{
    /** 采纳并写回：锚点校验由父级负责（写回是唯一接触编辑器的动作） */
    apply: [text: string];
    /** 打开详情面板看校验明细与参考表依据 */
    detail: [];
    /** 注释补全开关变更：持久化与 monaco provider scope 同步都在父级 */
    'update:inlineDraft': [value: boolean];
}>();

const { t } = useI18n();

const prompt = ref('');
const includePlans = ref(false);
const elapsed = ref(0);
const recentPrompts = ref<string[]>([]);
const inputRef = ref<{ focus: () => void; blur: () => void } | null>(null);

const intents = computed(() => listDraftIntents(props.materials));
const activeIntent = computed(() => intents.value.find((item) => item.id === intent.value));
const placeholder = computed(() => t(activeIntent.value?.hintKey ?? 'db.draft.hintDraft'));

/** 材料里有 SQL、意图允许空描述、且此刻输入框确实为空：回车即可，不必先打字 */
const zeroInputReady = computed(() => !prompt.value.trim() && canRunWithoutPrompt(activeIntent.value, props.materials));

/** 结果正文：散文产物取 explanation，否则取可写回的语句 */
const prose = computed(() => (props.draft.state.result?.verification === '' ? (props.draft.state.result?.explanation ?? '') : ''));
const adoptText = computed(() => joinStatements(props.draft.state.result?.statements));
const canAdopt = computed(() => {
    const result = props.draft.state.result;
    return !!result && canWriteback(result.verification) && !isEmptyDraft(result.statements);
});

/** 模板 chips 只服务「按描述生成且还没开始写」：已有材料或已写了一半时，它们只会把视线从输入框上挤走 */
const showTemplates = computed(() => activeIntent.value?.promptOptional === false && !prompt.value.trim());

/** 常用需求模板：本地化后的整句，点一下填进输入框再改几个词 */
const templatePrompts = computed(() => [t('db.draft.templateCount'), t('db.draft.templateRecent'), t('db.draft.templateTop')]);

/** 等待秒数：只在 loading 期间计数，结束后清零（让「上次等了 20 秒」不残留到下一次） */
let timer: ReturnType<typeof setInterval> | null = null;
watch(
    () => props.draft.state.loading,
    (loading) => {
        if (timer) {
            clearInterval(timer);
            timer = null;
        }
        if (!loading) {
            elapsed.value = 0;
            return;
        }
        const startedAt = Date.now();
        timer = setInterval(() => {
            elapsed.value = Math.floor((Date.now() - startedAt) / 1000);
        }, 1000);
    }
);

const submit = async () => {
    const intentDef: DraftIntentDef | undefined = activeIntent.value;
    const issue = validateDraftInput({
        prompt: prompt.value,
        selection: props.materials.selection,
        statement: props.materials.statement,
        requiresSelection: intentDef?.requiresSelection ?? false,
        promptOptional: intentDef?.promptOptional ?? false,
        // 上一版结果也是材料：不计入就会绕过材料总长闸门，把整段脚本再发一遍
        refine: adoptText.value,
    });
    if (issue) {
        ElMessage.warning(t(issue.key, issue.params ?? {}));
        return;
    }
    const text = prompt.value.trim();
    // 已有结果时默认「继续细化」而不是「重做」：浮层里连续回车是对话式收敛的自然预期
    const previous = canAdopt.value ? adoptText.value : '';
    const result = await props.draft.generate(
        buildDraftPayload({
            db: props.dbName,
            intent: intent.value,
            prompt: text,
            selection: props.materials.selection,
            statement: props.materials.statement,
            refine: previous,
            includePlans: includePlans.value,
        })
    );
    if (result && text) {
        recentPrompts.value = pushRecentPrompt(window.localStorage, props.dbId, text);
    }
};

const onIntentChange = (value: DraftIntentId) => {
    if (value === intent.value) {
        return;
    }
    intent.value = value;
    includePlans.value = intents.value.find((item) => item.id === value)?.includePlansByDefault ?? false;
    // 换意图 = 换一个问题，上一版结论不能再被当成本次答案。作废只写在用户点击处：
    // 打开浮层时父级也会改 intent（右键菜单点具体意图那一轮），写成 watch 会把刚发起的那次请求作废掉
    props.draft.cancel();
};

const onPickPrompt = (text: string) => {
    prompt.value = text;
    nextTick(() => inputRef.value?.focus());
};

/** Tab 采纳：结果可用时一口吞掉按键，否则 Tab 归还给焦点流转（不能让键位随结果状态漂移） */
const onTab = () => {
    if (!canAdopt.value) {
        return;
    }
    onAdopt();
};

const onAdopt = () => {
    if (!canAdopt.value) {
        return;
    }
    emits('apply', adoptText.value);
    // 采纳即完成这一轮问答：留着输入框里的描述会在下一轮被误当成新材料
    prompt.value = '';
};

/** 关闭即放弃这一轮：watch(visible) 统一中断在途请求，× 与 Esc 两个入口走同一条路径 */
const onClose = () => {
    visible.value = false;
};

const onCancelWaiting = () => {
    props.draft.cancel();
};

// 打开时：聚焦输入框、同步默认材料选项；runOnOpen 的零输入路径直接跑起来
watch(
    () => visible.value,
    (open) => {
        if (!open) {
            elapsed.value = 0;
            // 收起即放弃这一轮：留着在途请求，下次打开会看到上一版结果突然冒出来
            props.draft.cancel();
            return;
        }
        recentPrompts.value = readRecentPrompts(window.localStorage, props.dbId);
        includePlans.value = activeIntent.value?.includePlansByDefault ?? false;
        nextTick(() => {
            inputRef.value?.focus();
            // 右键菜单点「优化/解释/改写」已经表达了意图：材料已在手，不该再要一次回车
            if (props.runOnOpen && zeroInputReady.value) {
                void submit();
            }
        });
    },
    { immediate: true }
);
</script>

<style scoped>
.ai-bar {
    width: 520px;
    max-width: calc(100% - 16px);
    /* 面板钉在右上角后不再按光标翻转：内容（结果 / 模板 / 最近）可能很长，
       超出编辑器高度时由面板自己滚动，而不是把外层分栏撑出滚动条 */
    max-height: calc(100% - 16px);
    overflow-y: auto;
    border-color: var(--el-border-color);
    background: var(--el-bg-color);
}

.intent-chip {
    padding: 2px 10px;
    border: 1px solid var(--el-border-color);
    border-radius: 999px;
    font-size: 12px;
    line-height: 18px;
    color: var(--el-text-color-regular);
    background: transparent;
}

.intent-chip:hover {
    border-color: var(--el-color-primary);
    color: var(--el-color-primary);
}

.intent-chip-active {
    border-color: var(--el-color-primary);
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
}

/* 关闭按钮：不抢意图 chips 的视觉权重，悬停才变主色 */
.bar-close {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    border: 0;
    border-radius: 4px;
    color: var(--el-text-color-secondary);
    background: transparent;
    cursor: pointer;
}

.bar-close:hover {
    color: var(--el-color-primary);
    background: var(--el-fill-color);
}

.soft-chip {
    padding: 1px 8px;
    border-radius: 4px;
    font-size: 12px;
    line-height: 18px;
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);
}

.soft-chip:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
}
</style>
