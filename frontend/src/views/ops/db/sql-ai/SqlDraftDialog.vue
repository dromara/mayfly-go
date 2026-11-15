<template>
    <el-dialog v-model="visible" :title="t('db.draft.title')" width="760px" append-to-body destroy-on-close>
        <div class="flex flex-col gap-3">
            <!-- 意图与材料摘要：让用户在提交前就知道「这句话会带哪些材料过去」 -->
            <div class="flex items-center justify-between gap-2">
                <el-radio-group v-model="intent" size="small" @change="onIntentChange">
                    <el-radio-button v-for="item in intents" :key="item.id" :value="item.id">
                        {{ t(item.titleKey) }}
                    </el-radio-button>
                </el-radio-group>
                <div class="flex items-center gap-2 text-xs text-gray-500">
                    <span>{{ t('db.draft.materialSummary', { selection: materials.selection.length, statement: materials.statement.length }) }}</span>
                    <el-switch v-model="includePlans" size="small" :active-text="t('db.draft.includePlans')" />
                </div>
            </div>

            <el-input
                v-model="prompt"
                type="textarea"
                :rows="4"
                resize="vertical"
                :maxlength="MAX_PROMPT_LENGTH"
                show-word-limit
                :placeholder="t(activeIntent?.hintKey ?? 'db.draft.hintDraft')"
                @keydown.ctrl.enter.prevent="onGenerate()"
                @keydown.meta.enter.prevent="onGenerate()"
            />

            <!-- 等待期反馈：一次生成要经过「选表 → 生成 → 校验」，只转圈用户不知道在干什么、也不知道要等多久 -->
            <div v-if="state.loading" class="flex items-center gap-2 text-xs text-gray-500">
                <SvgIcon class="is-loading" name="Loading" />
                <span>{{ t('db.draft.loadingHint', { seconds: elapsed }) }}</span>
                <el-button link type="primary" size="small" @click="onCancelWaiting">{{ t('common.cancel') }}</el-button>
            </div>

            <el-alert v-if="state.error" :title="state.error" type="error" :closable="false" show-icon />
            <el-alert v-if="state.timedOut" :title="t('db.draft.timeout')" type="warning" :closable="false" show-icon />

            <div v-loading="state.loading" class="min-h-16">
                <template v-if="state.result">
                    <div class="mb-2 flex flex-wrap items-center gap-2">
                        <el-tag v-if="state.result.verification !== ''" :type="verificationTagType(state.result.verification)" size="small" disable-transitions>
                            {{ t(`db.draft.verification_${state.result.verification}`) }}
                        </el-tag>
                        <el-tag type="info" size="small" disable-transitions>{{ state.result.dialect }}</el-tag>
                        <span class="text-xs text-gray-500">
                            {{ t('db.draft.calls', { repairs: state.result.repairs, tokens: state.result.usage?.totalTokens ?? 0 }) }}
                        </span>
                        <el-tag v-if="state.result.candidates > 0" type="warning" size="small" disable-transitions>
                            {{ t('db.draft.rerolled', { count: state.result.candidates }) }}
                        </el-tag>
                    </div>
                    <!-- 校验只覆盖「表列存在 + 语法 + 权限 + 计划」，覆盖不到取值域（枚举/状态码等）；
                         真机验证过：metric='query_time' 列都存在但库里没有这个取值，语句能过校验却查不出数据。
                         这句话就是让用户知道「已通过校验」不等于「业务语义已确认」。 -->
                    <div v-if="state.result.verification !== ''" class="mb-2 text-xs text-gray-500">{{ t('db.draft.verificationScopeTip') }}</div>

                    <el-alert
                        v-for="(warning, index) in state.result.warnings ?? []"
                        :key="`w-${index}`"
                        :title="warning"
                        type="warning"
                        :closable="false"
                        show-icon
                        class="mb-1"
                    />

                    <!-- 散文产物（解释）：不提供写回，内容按纯文本渲染 -->
                    <div v-if="state.result.explanation" class="mb-2 rounded border p-2">
                        <div class="mb-1 text-xs text-gray-500">{{ t('db.draft.explanationTitle') }}</div>
                        <pre class="max-h-60 overflow-auto whitespace-pre-wrap break-words text-xs leading-5">{{ state.result.explanation }}</pre>
                    </div>

                    <div v-for="(statement, index) in state.result.statements ?? []" :key="`s-${index}`" class="mb-2 rounded border p-2">
                        <div class="mb-1 flex items-center justify-between gap-2 text-xs text-gray-500">
                            <span>{{ t('db.draft.statementIndex', { index: index + 1 }) }}</span>
                            <span v-if="statement.notes">{{ statement.notes }}</span>
                        </div>
                        <pre class="max-h-48 overflow-auto whitespace-pre-wrap break-all font-mono text-xs leading-5">{{ statement.sql }}</pre>
                        <div v-if="statement.assumptions?.length" class="mt-1 text-xs text-gray-500">
                            {{ t('db.draft.assumptions') }}: {{ joinList(statement.assumptions) }}
                        </div>
                    </div>

                    <div v-if="state.result.checks?.length" class="mt-2">
                        <div class="mb-1 text-xs text-gray-500">{{ t('db.draft.checks', checkSummary) }}</div>
                        <div v-for="(check, index) in state.result.checks" :key="`c-${index}`" class="mb-1 flex items-start gap-2">
                            <el-tag :type="severityTagType(check.severity)" size="small" disable-transitions>
                                {{ t(`db.draft.severity_${check.severity}`) }}
                            </el-tag>
                            <div class="text-xs leading-5">
                                <span>{{ check.message }}</span>
                                <span v-if="check.identifier" class="ml-1 text-gray-400">({{ check.identifier }})</span>
                                <span v-if="check.candidates?.length" class="ml-1 text-gray-400">
                                    {{ t('db.draft.candidates') }}: {{ check.candidates.join(', ') }}
                                </span>
                            </div>
                        </div>
                    </div>

                    <el-collapse v-if="state.result.contextTables?.length" class="mt-2">
                        <el-collapse-item :title="t('db.draft.contextTables', { count: state.result.contextTables.length })" name="ctx">
                            <el-table :data="state.result.contextTables" size="small" border max-height="220">
                                <el-table-column prop="name" :label="t('db.tableName')" min-width="160" />
                                <el-table-column v-if="hasContextComment" prop="comment" :label="t('common.remark')" min-width="160" show-overflow-tooltip />
                                <el-table-column prop="columns" :label="t('db.draft.columns')" width="90" />
                                <el-table-column prop="reason" :label="t('db.draft.reason')" min-width="160" show-overflow-tooltip />
                            </el-table>
                        </el-collapse-item>
                    </el-collapse>
                </template>
                <div v-else-if="!state.loading && !state.error" class="py-6 text-center text-sm text-gray-500">
                    {{ t('db.draft.emptyHint') }}
                </div>
            </div>
        </div>

        <template #footer>
            <div class="flex items-center justify-between">
                <span class="text-xs text-gray-500">{{ t('db.draft.noExecuteTip') }}</span>
                <div>
                    <el-button @click="visible = false">{{ t('common.close') }}</el-button>
                    <el-button :loading="state.loading" @click="onGenerate()">
                        {{ state.result ? t('db.draft.regenerate') : t('db.draft.generate') }}
                    </el-button>
                    <!-- 细化与重做是两件事：细化带上一版语句（保留已对的部分），重做从零再来。
                         合并成一个按钮会让用户无法表达「就在这版上改」，只能反复整题重问 -->
                    <el-tooltip v-if="canRefine" :content="t('db.draft.refineTip')" placement="top">
                        <el-button type="primary" plain :loading="state.loading" @click="onGenerate(true)">
                            {{ t('db.draft.refine') }}
                        </el-button>
                    </el-tooltip>
                    <el-button type="primary" :disabled="!canApply" @click="applyDraft">{{ t('db.draft.apply') }}</el-button>
                </div>
            </div>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
/**
 * SQL 草稿详情面板（模态）。
 *
 * 职责边界：本组件只负责「把一次生成的完整依据摊开」——校验明细、假设、参考表与选用原因；
 * 日常生成走 SQL 助手面板（SqlAiBar）。两者消费父级注入的**同一份** draft 状态，
 * 因此这里看到的结果与面板里采纳的必定是同一版。
 *
 * 不碰编辑器：写回由父级执行并在写回前校验锚点（见 draftEditor.writeback）——把编辑器实例
 * 塞进对话框会让「异步返回时内容已变」这类判定分散到两处。
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { ElMessage } from 'element-plus';
import { listDraftIntents } from './draftIntents';
import {
    MAX_PROMPT_LENGTH,
    buildDraftPayload,
    canWriteback,
    hasAnyComment,
    isEmptyDraft,
    joinList,
    joinStatements,
    severityTagType,
    summarizeChecks,
    validateDraftInput,
    verificationTagType,
} from './draftModel';
import type { DraftIntentId } from './types';
import type { DraftEditorContext } from './draftIntents';
import type { DraftHandle } from './useSqlDraft';

const props = defineProps<{
    dbName: string;
    /** 与 SQL 助手面板共用的草稿状态：结果只有一份 */
    draft: DraftHandle;
    materials: DraftEditorContext & { selection: string };
}>();

const visible = defineModel<boolean>({ default: false });
const intent = defineModel<DraftIntentId>('intent', { required: true });

const emits = defineEmits<{
    /** 用户确认写回：父级负责锚点校验与插入/替换 */
    apply: [text: string];
}>();

const { t } = useI18n();
const state = props.draft.state;

const prompt = ref('');
const includePlans = ref(false);

// 只列当前材料下可用的意图：与 SQL 助手面板走同一份注册表，两处不会出现不同的选项
const intents = computed(() => listDraftIntents(props.materials));

const activeIntent = computed(() => intents.value.find((item) => item.id === intent.value));
const checkSummary = computed(() => summarizeChecks(state.result?.checks));
// 参考表备注列：整列都空时不渲染——空列白占 160px，会把真正有信息量的「选用原因」挤到截断
const hasContextComment = computed(() => hasAnyComment(state.result?.contextTables));

// 等待秒数：只在 loading 期间计数，结束后清零（让"上次等了 20 秒"不会残留到下一次）
const elapsed = ref(0);
let elapsedTimer: ReturnType<typeof setInterval> | null = null;
const stopElapsed = () => {
    if (elapsedTimer) {
        clearInterval(elapsedTimer);
        elapsedTimer = null;
    }
};
watch(
    () => state.loading,
    (loading) => {
        stopElapsed();
        if (!loading) {
            elapsed.value = 0;
            return;
        }
        const startedAt = Date.now();
        elapsed.value = 0;
        elapsedTimer = setInterval(() => {
            elapsed.value = Math.floor((Date.now() - startedAt) / 1000);
        }, 1000);
    }
);
onBeforeUnmount(stopElapsed);
const canApply = computed(() => !!state.result && canWriteback(state.result.verification) && !isEmptyDraft(state.result.statements));
// 上一版草稿：细化轮的输入。散文产物（explain）没有可续写的语句，故不参与
const previousDraft = computed(() => (canApply.value ? joinStatements(state.result?.statements) : ''));
const canRefine = computed(() => previousDraft.value.length > 0);

// 打开时不清在途结果：详情面板与助手面板共用的就是这一版结论（清掉等于把用户刚拿到的结果弄丢）；
// 但输入框从空开始：上一轮的描述留着会被误当成本轮的追问
watch(
    visible,
    (open) => {
        if (open) {
            prompt.value = '';
            includePlans.value = activeIntent.value?.includePlansByDefault ?? false;
            return;
        }
        // 关掉面板即放弃这一轮：否则在途结果会在下一次打开时突然冒出来
        if (state.loading) {
            props.draft.cancel();
        }
    },
    { immediate: true }
);

// 换意图时重算「默认是否取执行计划」：不同意图的默认材料不一样（父级或助手面板改的 intent 也要同步）
watch(intent, () => {
    includePlans.value = activeIntent.value?.includePlansByDefault ?? false;
});

// 换意图 = 换一个问题，上一版结论不能再被当成当前结果。
// 与助手面板同判据：只写在用户点击处，父级打开面板时改 intent 不走这里，否则会把刚发起的请求作废掉
const onIntentChange = () => {
    props.draft.cancel();
};

/** 取消等待：中断在途请求并立即停下转圈（只作废序号不中断的话，界面看着就是「点了没反应」） */
const onCancelWaiting = () => {
    props.draft.cancel();
};

const onGenerate = async (refine = false) => {
    // 细化只在有上一版时成立：按钮已按 canRefine 隐藏，这里再兜一层（快捷键/程序化调用同守卫）
    const refineOf = refine ? previousDraft.value : '';
    const issue = validateDraftInput({
        prompt: prompt.value,
        selection: props.materials.selection,
        statement: props.materials.statement,
        refine: refineOf,
        requiresSelection: activeIntent.value?.requiresSelection ?? false,
        promptOptional: activeIntent.value?.promptOptional ?? false,
    });
    if (issue) {
        // 没有插值参数的原因也要给一个空对象：t() 的重载不接受 undefined
        ElMessage.warning(t(issue.key, issue.params ?? {}));
        return;
    }
    // 请求体只由 buildDraftPayload 装配：字段映射（statement → anchorStatement）与
    // 「空值不占位」的规则只有一份实现，界面不再手搓一份平行的 payload
    await props.draft.generate(
        buildDraftPayload({
            db: props.dbName,
            intent: intent.value,
            prompt: prompt.value,
            selection: props.materials.selection,
            statement: props.materials.statement,
            refine: refineOf,
            includePlans: includePlans.value,
        })
    );
};

const applyDraft = () => {
    if (!state.result) {
        return;
    }
    const text = joinStatements(state.result.statements);
    if (!text) {
        return;
    }
    emits('apply', text);
};
</script>
