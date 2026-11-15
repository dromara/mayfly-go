<template>
    <div class="h-full flex flex-col">
        <SqlEditorToolbar
            :token="token"
            :upload-url="getUploadSqlFileUrl()"
            :upload-fn="handleSqlFileUpload"
            :atomic="state.atomic"
            :ai-enabled="aiEnabled"
            @update:atomic="(value: boolean) => (state.atomic = value)"
            @run="onRunSql()"
            @run-all="onRunAllSql()"
            @format="onFormatSql()"
            @review="onReviewSql()"
            @precheck="onPrecheckDdl()"
            @save="saveSql()"
            @open-sql-assistant="onToggleSqlAssistant()"
        />

        <!-- 详情面板：日常生成走右上角的 SQL 助手面板，这里只承担「把一次生成的完整依据摊开」，
             与面板共用同一份草稿状态与材料快照（两处看到的结果必定同一版） -->
        <SqlDraftDialog
            v-model="state.detailVisible"
            v-model:intent="state.barIntent"
            :db-name="dbName"
            :draft="draft"
            :materials="draftMaterials"
            @apply="applyDraftText"
        />

        <DdlPrecheckDialog
            v-model="state.precheckVisible"
            :db-id="dbId"
            :db-name="dbName"
            :sql="state.precheckSql"
        />

        <el-splitter ref="splitterRef" class="flex-1 min-h-0" layout="vertical" @resize-end="syncEditorPanelSize">
            <el-splitter-panel :size="state.editorSize" max="80%">
                <!-- 浮层容器：SQL 助手面板钉在编辑器右上角（工具条入口正下方），
                     必须与编辑器共用一个 relative 容器；高度扣除 mt-1(4px) + 组件边框(2px)，
                     使内容恰好填满面板，避免溢出滚动条 -->
                <div class="relative h-full">
                    <MonacoEditor
                        ref="monacoEditorRef"
                        class="mt-1"
                        v-model="state.sql"
                        language="sql"
                        height="calc(100% - 6px)"
                        :id="'MonacoTextarea-' + getKey()"
                        :options="editorOptions"
                        @ready="initMonacoEditor"
                    />

                    <!-- 自然语言 → SQL 草稿：产物是文本，执行仍走原有闸口（本组件不提供任何 AI 执行入口） -->
                    <SqlAiBar
                        v-if="aiEnabled"
                        v-model="state.barVisible"
                        v-model:intent="state.barIntent"
                        :db-id="dbId"
                        :db-name="dbName"
                        :draft="draft"
                        :materials="draftMaterials"
                        :run-on-open="state.barRunOnOpen"
                        :inline-draft="state.inlineDraft"
                        @update:inline-draft="onInlineDraftChange"
                        @apply="applyDraftText"
                        @detail="state.detailVisible = true"
                    />
                </div>
            </el-splitter-panel>

            <el-splitter-panel>
                <SqlExecResultTabs
                    :exec-res-tabs="state.execResTabs"
                    :active-tab="state.activeTab"
                    :db-id="dbId"
                    :db="dbName"
                    :table-data-empty-text="state.tableDataEmptyText"
                    @tab-remove="onRemoveTab"
                    @tab-change="active"
                    @submit-update-fields="submitUpdateFields"
                    @cancel-update-fields="cancelUpdateFields"
                    @change-updated-field="changeUpdatedField"
                    @data-delete="onDeleteData"
                    @submit-ticket="(sql: string) => submitTicket(sql)"
                />
            </el-splitter-panel>
        </el-splitter>

        <!-- 非查询语句执行前确认框：填备注 + 查看执行计划 -->
        <SqlRunRemarkDialog ref="remarkDialogRef" />

        <!-- 被策略拦下的语句直接带进提单表单，省掉切页面、重选库、重贴 SQL -->
        <WorkTicketSubmit ref="ticketRef" :biz-type="FlowBizType.DbSqlExec.value" hidden />
    </div>
</template>

<script lang="ts" setup>
import { notBlank } from '@/common/assert';
import config from '@/common/config';
import { getToken } from '@/common/utils/storage';
import { ElMessageBox } from 'element-plus';
import { defineAsyncComponent, onBeforeUnmount, onMounted, reactive, ref, useTemplateRef, watch } from 'vue';

// 纯类型导入（编译期擦除）：monaco 运行时经 setup 动态取用，见 initMonacoEditor 的注释
import type { editor, IRange } from 'monaco-editor';

import { dbApi, uploadSqlFile } from '../api';
import SqlRunRemarkDialog from './SqlRunRemarkDialog.vue';
import { openSqlPlan } from './sqlPlanService';
import { DbInst } from '../db';

import { joinClientParams } from '@/common/request';
import type { MonacoEditorExpose } from '@/components/monaco/types';
import type { TicketPrefill } from '@/views/flow/types';
import { Msg } from '@/hooks/useI18n';
import { useDebounceFn, useEventListener } from '@vueuse/core';
import { useI18n } from 'vue-i18n';

import { FlowBizType } from '@/views/flow/enums';
import SqlEditorToolbar from './SqlEditorToolbar.vue';
import SqlExecResultTabs from './SqlExecResultTabs.vue';
import DdlPrecheckDialog from './DdlPrecheckDialog.vue';
import WorkTicketSubmit from '@/views/flow/components/WorkTicketSubmit.vue';
import { formatSql } from './utils/formatSql';
import SqlDraftDialog from '../sql-ai/SqlDraftDialog.vue';
import SqlAiBar from '../sql-ai/SqlAiBar.vue';
import {
    captureAnchor,
    readEditorMaterials,
    resolveEscapeTarget,
    writeback,
    type DraftAnchor,
    type EditorMaterials,
} from '../sql-ai/draftEditor';
import { useSqlDraft } from '../sql-ai/useSqlDraft';
import { useSqlAiEnabled } from '../sql-ai/useSqlAiEnabled';
import { DRAFT_INTENTS, canRunWithoutPrompt, defaultDraftIntent, findDraftIntent } from '../sql-ai/draftIntents';
import type { DraftIntentId } from '../sql-ai/types';
import { useSqlExec, type ExecResTab, type ExecResTabState } from './composables/useSqlExec';

// 编辑器主体约 967KB(gzip)，本组件所在 tab 只在点开 SQL 编辑器时才挂载，故保持动态引入
// （main.ts 已在首屏空闲时预取，实际不会出现等待）
const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

const emits = defineEmits<{
    /** 脚本保存成功，回传所属库，父级据此刷新该库下的脚本树节点 */
    saveSqlSuccess: [dbId: number, dbName: string];
    /** 编辑器初始化完成，上报 model URI，父级据此登记到补全作用域使多编辑器共存时按模型路由 */
    editorReady: [modelUri: string];
}>();

const { t } = useI18n();

/**
 * 编辑器构建选项。
 *
 * inlineSuggest 必须显式开启：monaco 独立版默认关闭它，此时即便注册了 InlineCompletionsProvider、
 * 用户也打开了「注释补全」开关，`editor.action.inlineSuggest.trigger` 依旧什么都不做——
 * 真机表现为"开关能打开、永远不出建议"。没装 provider（开关关闭/未配置模型）时不会有任何可见变化。
 */
const editorOptions = { inlineSuggest: { enabled: true } };

/**
 * AI 可用性：未配置模型时编辑器里不出现任何 AI 入口（工具栏按钮、注释补全开关、右键动作）。
 * 判据来自全站共享的 /ai/status 探测（与 AI 助手同一份），不在本组件里另发请求。
 */
const { aiEnabled } = useSqlAiEnabled();

const props = defineProps({
    dbId: {
        type: Number,
        required: true,
    },
    dbName: {
        type: String,
        required: true,
    },
    // sql脚本名，若有则去加载该sql内容
    sqlName: {
        type: String,
    },
    // 库的资源编码：被策略拦下时提单要靠它才能直接解析出审批流程
    dbCode: {
        type: String,
        default: '',
    },
    // 实例类型（方言）：语句切割与草稿召回都按方言语义走，缺省时按默认词法
    dbType: {
        type: String,
        default: '',
    },
});

const token = getToken();
const monacoEditorRef = useTemplateRef<MonacoEditorExpose>('monacoEditorRef');

// 被策略拦下时用于预填提单表单：语句与库已确定，用户只需确认备注
const ticketRef = useTemplateRef<{ open: (prefill?: TicketPrefill) => void }>('ticketRef');

// 非查询语句执行前的确认框（备注 + 查看执行计划）；open 返回备注或 null（取消）
const remarkDialogRef = useTemplateRef<{ open: (id: number, db: string, sql: string) => Promise<string | null> }>('remarkDialogRef');
const askExecRemark = (sql: string) => remarkDialogRef.value?.open(props.dbId, props.dbName, sql) ?? Promise.resolve('');

const submitTicket = (sql: string) => {
    // dbCode 决定这张工单归哪个流程审批：带上它，业务表单挂载时才能自己上报资源，
    // 否则抽屉会停在「不存在审批节点」且确定按钮不可点
    ticketRef.value?.open({ bizForm: { dbId: props.dbId, dbName: props.dbName, dbCode: props.dbCode, sql } });
};

// 编辑器实例在 ready 前不存在（编辑器为按需加载），故可空；useSqlExec 的入参契约本就是 | null
let monacoEditor: editor.IStandaloneCodeEditor | null = null;

/** 空的编辑器材料：浮层未打开时的占位，与 readEditorMaterials 的返回形状一致 */
const emptyMaterials = (): EditorMaterials => ({ selection: '', statement: '', hasSelection: false });

const state = reactive({
    editorSize: 50, // editor高度比例
    sql: '', // 当前编辑器的sql内容
    sqlName: '', // sql模板名称
    execResTabs: [] as ExecResTabState[],
    activeTab: 1,
    // 原子执行开关：整批语句共用后端一个事务，全成功才提交
    atomic: false,
    tableDataEmptyText: t('db.tableDataEmptyTextTips'),
    // DDL 预检框：sql 单独存一份，避免执行后再打开时拿到已变的编辑器内容
    precheckVisible: false,
    precheckSql: '',
    // 注释行补全：默认关闭，开启状态持久化（每次触发都会调用模型，必须显式选择）
    inlineDraft: getStoredInlineDraft(),
    // SQL 助手面板：材料与写回锚点在打开那一刻快照，异步返回后据此判断内容是否已被改动
    barVisible: false,
    barIntent: 'draft' as DraftIntentId,
    // 打开即生成：从右键菜单点了具体意图时置 true（用户已经表达完意图）
    barRunOnOpen: false,
    // 详情面板：与助手面板共用一份结果，只负责摊开校验明细与参考表依据
    detailVisible: false,
});

// 草稿状态只有一份：助手面板与详情面板共用同一个 handle，两处不会出现不同的结论
const draft = useSqlDraft(props.dbId, props.dbName);

// 打开助手面板那一刻的材料快照（与详情面板共用）：不放进 reactive，整体替换更不易错
const draftMaterials = ref<EditorMaterials>(emptyMaterials());

// 草稿写回锚点：不放进 reactive（它只是发起时的快照，不需要响应式；放进去反而会触发无谓渲染）
let draftAnchor: DraftAnchor | null = null;

// 行内补全用到的 model uri 与 monaco 侧 provider 的解除函数（动态导入后填充）
let inlineDraftUri = '';
let inlineDraftRelease: (() => void) | null = null;

/** 读取注释补全开关（坏值一律按关闭处理：错误的默认值会静默产生模型费用） */
function getStoredInlineDraft(): boolean {
    try {
        return window.localStorage.getItem('db-sql-inline-draft') === '1';
    } catch {
        return false;
    }
}

/**
 * 把注释补全的启用态同步给 monaco provider（scope 尚未登记时为 no-op）。
 *
 * 判据是「用户开了开关 **且** 系统里配了可用模型」。开关状态来自 localStorage，
 * 管理员把模型停用后它仍然是开的——只按开关放行，provider 会在每个注释行触发时
 * 打一次注定失败的接口：界面表现是按 Tab 毫无反应，后端却在刷错误日志。
 */
const syncInlineDraftScope = async (enabled: boolean) => {
    if (!inlineDraftUri) {
        return;
    }
    const { setInlineDraftScope } = await import('../sql-ai/inlineDraft');
    setInlineDraftScope(inlineDraftUri, { dbId: props.dbId, dbName: props.dbName, enabled: enabled && aiEnabled.value });
};

/** 注释补全开关变更：持久化并同步给 monaco provider 的 scope */
const onInlineDraftChange = async (value: boolean) => {
    state.inlineDraft = value;
    try {
        // 隐私模式下写入会失败：本次会话仍然生效，只是下次需要重新打开
        window.localStorage.setItem('db-sql-inline-draft', value ? '1' : '0');
    } catch {
        // 持久化失败不能挡住开关本身生效
    }
    await syncInlineDraftScope(value);
};

const {
    getNowDbInst,
    pushNewTab,
    onRunSql,
    onRunAllSql,
    changeUpdatedField,
    onDeleteData,
    submitUpdateFields,
    cancelUpdateFields,
    onRemoveTab,
    activeTab: active,
} = useSqlExec({
    dbId: props.dbId,
    dbName: props.dbName,
    state,
    atomic: () => state.atomic,
    // 需审批语句自动开抽屉（预填库+SQL），与结果区「提交工单」按钮同一入口
    onNeedApproval: submitTicket,
    // 非查询语句执行前弹自定义确认框（含查看执行计划），取代原生 prompt
    askExecRemark,
    get monacoEditor() {
        return monacoEditor ?? null;
    },
});

// 探测结果可能在编辑器就绪之后才回来（登录后补探）：就绪事件里已尝试注册，这里补一次
watch(aiEnabled, (enabled: boolean) => {
    if (enabled) {
        void registerAiActions();
        return;
    }
    // 模型被停用：助手面板与详情面板立即收起，不让用户对着一个注定失败的入口继续按回车
    state.barVisible = false;
    state.detailVisible = false;
    // 两个方向都要重新收敛：模型被启用时补全要真的能用，被停用时 provider 要立刻停止发请求
    void syncInlineDraftScope(state.inlineDraft);
});

onBeforeUnmount(() => {
    // 编辑器销毁即注销 scope：多标签页共存时，残留的 uri→库 映射会把建议补到已关闭的库上
    inlineDraftRelease?.();
    inlineDraftRelease = null;
});

onMounted(async () => {
    // 第一个pane为sql editor
    syncEditorPanelSize(0, [-1]);
    useEventListener(
        'resize',
        useDebounceFn(() => syncEditorPanelSize(0, [-1]), 200)
    );

    // 默认新建一个结果集tab
    pushNewTab(1);

    state.sqlName = props.sqlName ?? '';
    if (props.sqlName) {
        const res = await dbApi.getSql.request({ id: props.dbId, type: 1, db: props.dbName, name: props.sqlName });
        state.sql = res.sql;
    }
    // 编辑器为按需加载，快捷键等初始化挂在其 @ready 上（见模板），此处不再用定时器猜就绪时机
    // 元数据分级：此处仅 fire-and-forget 预热「表名清单」（轻，已 localStorage 缓存），编辑器立即可交互；
    // 列补全改走按表 fragment（敲到 tbl./别名. 时才经 loadColumns → c-metadata 取该表列，前端按表缓存 + 服务端进程内缓存）。
    // 结构新鲜度由事件驱动，本地缓存失效只有两个入口：SQL 执行（编辑器按 isDdlSql 判定、表编辑弹框按构造一律失效）
    // 与资源树节点重载（右击刷新、建/改/改名/删/复制表回调，见 DbDataOp.reloadNode），两者都汇到 DbInst.invalidateSchema。
    // 本地表清单缓存无 TTL：他端/外部工具改表不会触发上述入口，需右击刷新才会重取（服务端 schema 缓存另有 TTL 兜底）。
    getNowDbInst()
        .loadTables(props.dbName)
        .then(() => {});
});

const splitterRef = useTemplateRef<{ $el: HTMLElement }>('splitterRef');

/**
 * 同步编辑器面板高度（首次挂载、窗口尺寸变化、拖动分隔条后）。
 *
 * 结果区不再跟着算像素高度：表格改为在页签面板里 flex 撑满剩余空间（见 SqlExecResultTabs），
 * 由布局给出高度。此前这里按「容器高 - 编辑器高 - 固定扣减」硬算，没算页签头与「表格/图表」切换行，
 * 结果表格比可用空间高出一截，而 `.el-tabs__content` 是 overflow:hidden，被裁掉的正好是表格底部——
 * 即 el-table-v2 的横向滚动条，表现为「列放不下却没有横向滚动条」。
 */
const syncEditorPanelSize = (index: number, sizes: number[]) => {
    if (!sizes || sizes.length === 0) {
        return;
    }

    // 基于splitter容器实际高度计算，兼容全屏模式
    const splitterEl = splitterRef.value?.$el || splitterRef.value;
    const containerHeight = splitterEl ? (splitterEl as HTMLElement).getBoundingClientRect().height : window.innerHeight - 220;
    const plitpaneHeight = containerHeight - 10;

    let editorHeight = sizes[0];
    if (editorHeight < 0 || editorHeight > plitpaneHeight - 43) {
        // 默认占50%
        editorHeight = plitpaneHeight / 2;
    }

    state.editorSize = editorHeight;
};

const getKey = () => {
    if (props.sqlName) {
        return `${props.dbId}:${props.dbName}.${props.sqlName}`;
    }
    return props.dbId + ':' + props.dbName;
};

const saveSql = async () => {
    const sql = monacoEditor?.getModel()?.getValue();
    notBlank(sql, t('db.sqlCannotEmpty'));

    let sqlName = state.sqlName;
    if (!sqlName) {
        try {
            const input = await ElMessageBox.prompt(t('db.enterSqlScriptNameTips'), 'SQL Name', {
                confirmButtonText: t('common.confirm'),
                cancelButtonText: t('common.cancel'),
                inputPattern: /.+/,
                inputErrorMessage: t('db.enterSqlScriptNameTips'),
            });
            sqlName = input.value;
            state.sqlName = sqlName;
        } catch (e) {
            return;
        }
    }

    await dbApi.saveSql.request({ id: props.dbId, db: props.dbName, sql: sql, type: 1, name: sqlName });
    Msg.saveSuccess();
    // 保存sql脚本成功事件
    emits('saveSqlSuccess', props.dbId, props.dbName);
};

/**
 * 格式化sql
 */
const onFormatSql = async () => {
    const editorIns = monacoEditor;
    const selection = editorIns?.getSelection();
    const model = editorIns?.getModel();
    if (!selection || !model) {
        return;
    }

    const formatDialect = getNowDbInst().getDialect().getInfo().formatSqlDialect;
    // 有选中sql则格式化并替换选中sql, 否则格式化编辑器所有内容
    const read = () => (selection.isEmpty() ? model.getValue() : model.getValueInRange(selection));
    const source = read();

    const formatted = await formatSql(source, formatDialect);
    // 格式化器按需加载，首次点击存在加载窗口：期间关掉查询 tab 会销毁 model（继续读写必抛错），
    // 用户改过内容也不能再覆盖，两种情况都丢弃本次结果
    if (model.isDisposed() || read() !== source) {
        return;
    }
    if (selection.isEmpty()) {
        model.setValue(formatted);
        return;
    }
    replaceSelection(formatted, selection);
};

/**
 * 查看当前编辑器内容的执行计划（整段脚本交给后端按目标库方言切分，与执行同一口径）。
 */
const onReviewSql = () => {
    const sql = state.sql?.trim();
    if (!sql) {
        Msg.error(t('db.noSqlToRunMsg'));
        return;
    }
    // 执行计划走全局单例对话框（layout 挂载），多条可在框内逐条选中
    openSqlPlan(props.dbId, props.dbName, sql);
};

/** DDL 预检：把当前编辑器整段脚本交给后端切分与判定，只读、不执行 */
const onPrecheckDdl = () => {
    const sql = state.sql?.trim();
    if (!sql) {
        Msg.error(t('db.noSqlToRunMsg'));
        return;
    }
    state.precheckSql = sql;
    state.precheckVisible = true;
};

/**
 * 注册 AI 动作：右键菜单项 + ⌘K 打开面板（仅当模型已配置时）。
 *
 * 菜单项来自意图注册表，顺序由声明决定（新增意图不需要改本文件）。
 * 这里注册**全部**意图而不是按当前选区过滤：monaco 的菜单项在注册时定型，
 * 若按「此刻有没有选区」注册，用户在空白处右键后就再也看不到「按描述改写」；
 * 可用性改在点击时判定（openCommandBar 内）。
 *
 * 右键、工具条与快捷键打开的是同一个面板：意图只是面板的入参，
 * 面板固定在编辑器右上角，三条入口不必各学一套交互。
 */
let aiActionsRegistered = false;
const registerAiActions = async () => {
    if (!monacoEditor || aiActionsRegistered || !aiEnabled.value) {
        return;
    }
    // 标记先置再等：编辑器就绪与探测返回可能同一轮里各调一次，等完再置会注册出两份同名动作
    aiActionsRegistered = true;
    // 键位常量属于 monaco 运行时，经装配入口动态取（本组件的静态图不拖 monaco）
    const { KeyCode, KeyMod } = await import('@/components/monaco/setup');
    for (const intent of [...DRAFT_INTENTS].sort((a, b) => a.order - b.order)) {
        monacoEditor.addAction({
            id: `sql-draft-action-${intent.id}-` + getKey(),
            label: t(intent.titleKey),
            precondition: undefined,
            keybindingContext: undefined,
            contextMenuGroupId: 'navigation',
            contextMenuOrder: 4 + intent.order / 100,
            run: async function () {
                // 零输入就能成立的意图（优化/解释/改写）从右键进来也直接跑：
                // 用户选中语句再点「解释」，已经把这句话说完了（能不能直接跑由 openCommandBar 按同一判据定）
                openCommandBar(intent.id, true);
            },
        });
    }

    // 快捷键：⌘/Ctrl+K 打开助手面板（与 Cursor 的内联编辑同一键位预期）
    monacoEditor.addAction({
        id: 'sql-draft-bar-action-' + getKey(),
        label: t('db.draft.barEntry'),
        precondition: undefined,
        keybindingContext: undefined,
        keybindings: [KeyMod.chord(KeyMod.CtrlCmd | KeyCode.KeyK, 0)],
        run: async function () {
            openCommandBar();
        },
    });
};

/**
 * 打开 SQL 助手面板：在打开那一刻快照「选区 + 光标所在语句 + 模型版本」。
 *
 * 材料与写回锚点必须在同一时刻取：分开取会出现「材料是旧的、版本号是新的」这种自相矛盾的
 * 组合，写回时反而放行了过期结果。面板位置与选区无关（钉在编辑器右上角）。
 *
 * runOnOpen 只在「零输入可成立」时为 true：面板打开后自己回车一次就能出结果，
 * 描述类意图仍等用户写完（否则会出现一条没人想要的空描述请求）。
 */
const openCommandBar = (intentId?: DraftIntentId, runOnOpen = false) => {
    // 双保险：入口本身按 aiEnabled 隐藏，这里再挡一次（快捷键/程序化调用也走同一守卫）
    if (!aiEnabled.value || !monacoEditor) {
        return;
    }
    const materials = readEditorMaterials(monacoEditor, props.dbType);
    const intent = findDraftIntent(intentId ?? defaultDraftIntent(materials));
    if (!intent) {
        return;
    }
    if (intent.requiresSelection && !materials.hasSelection && !materials.statement.trim()) {
        Msg.warning(t('db.draft.sqlRequired'));
        return;
    }
    draftAnchor = captureAnchor(monacoEditor);
    draftMaterials.value = materials;
    // 新一轮提问从干净状态开始：上一版结果挂着会让用户误以为它就是本次的答案
    draft.cancel();
    state.barIntent = intent.id;
    state.barRunOnOpen = runOnOpen && canRunWithoutPrompt(intent, materials);
    state.barVisible = true;
};

/**
 * 工具条入口：点一下打开、再点一下收起（下拉面板的常规语义）。
 *
 * 收起只改可见态：面板的 watch(visible) 负责中断在途请求，不在两处各写一遍作废逻辑。
 */
const onToggleSqlAssistant = () => {
    if (state.barVisible) {
        state.barVisible = false;
        return;
    }
    openCommandBar();
};

/**
 * Esc 关闭浮层：挂在 window 捕获阶段，与焦点在哪无关。
 *
 * 为什么不绑在浮层根节点上：用户点回编辑器后焦点就离开了浮层，此时 Esc 什么也不会发生，
 * 而浮层还挂在屏幕上——用户感知就是「Esc 不灵活」。捕获阶段也早于 monaco 与 EP 自己的处理，
 * 保证一次按键只被一处消费。
 *
 * 让位规则（模态 > 查找条 > 助手面板）写在 resolveEscapeTarget 里，便于表驱动单测钉住。
 */
const onGlobalEscape = (event: KeyboardEvent) => {
    if (event.key !== 'Escape') {
        return;
    }
    const owner = resolveEscapeTarget({
        // 提单抽屉等模态弹层叠在编辑器上时，Esc 归弹层（全站已有 useEscapeClosesDrawer 接缝）。
        // 必须用 computed display 而不是 offsetParent：.el-overlay 是 position:fixed，
        // 而 fixed 元素的 offsetParent 恒为 null，用它做可见性判据会把模态误判成「没开」
        modalOpen: [...document.querySelectorAll<HTMLElement>('.el-overlay')].some((el) => getComputedStyle(el).display !== 'none'),
        findWidgetOpen: !!document.querySelector('.find-widget.visible'),
        barVisible: state.barVisible,
    });
    if (owner === 'bar') {
        event.preventDefault();
        // 只改可见态：面板的 watch(visible) 负责中断在途请求，不在两处各写一遍作废逻辑
        state.barVisible = false;
    }
};
useEventListener(window, 'keydown', onGlobalEscape, true);

/**
 * 把草稿写回编辑器。
 *
 * 写回前校验锚点：期间内容有改动时**不写入**并明确提示，而不是按旧坐标插到错误位置
 * （与 formatSql 的「内容变了就丢弃」同一判据）。
 */
const applyDraftText = (text: string) => {
    const failure = writeback(monacoEditor, draftAnchor, text);
    if (failure === 'stale') {
        Msg.error(t('db.draft.staleContent'));
        return;
    }
    if (failure === 'disposed') {
        Msg.error(t('db.draft.editorUnavailable'));
        return;
    }
    // 两个视图一起收：内容已变，挂着的旧结论与旧锚点再点一次就是写错位置
    state.barVisible = false;
    state.detailVisible = false;
    Msg.success(t('db.draft.applied'));
};

/**
 * 替换选中的内容
 */
const replaceSelection = (str: string, selection: IRange) => {
    const editorIns = monacoEditor;
    const model = editorIns?.getModel();
    if (!editorIns || !model) {
        return;
    }
    if (!selection) {
        model.setValue(str);
        return;
    }
    const { startLineNumber, endLineNumber, startColumn, endColumn } = selection;

    const textBeforeSelection = model.getValueInRange({
        startLineNumber: 1,
        startColumn: 0,
        endLineNumber: startLineNumber,
        endColumn: startColumn,
    });

    const textAfterSelection = model.getValueInRange({
        startLineNumber: endLineNumber,
        startColumn: endColumn,
        endLineNumber: model.getLineCount(),
        endColumn: model.getLineMaxColumn(model.getLineCount()),
    });

    editorIns.setValue(textBeforeSelection + str + textAfterSelection);
    editorIns.focus();
    editorIns.setPosition({
        lineNumber: startLineNumber,
        column: 0,
    });
};

// 自定义SQL文件上传处理
const handleSqlFileUpload = (options: { file: File }) => {
    const { file } = options;

    const { uploadId, abort } = uploadSqlFile(
        file,
        {
            dbId: props.dbId as number,
            dbName: props.dbName as string,
        },
        {
            onSuccess: () => {
                Msg.success('db.scriptFileUploadSuccess', { filename: file.name });
            },
            onError: (error) => {
                Msg.error('db.scriptFileUploadFailed', { filename: file.name, error: error.message });
            },
        }
    );

    return { abort };
};

// 获取sql文件上传执行url
const getUploadSqlFileUrl = () => {
    return `${config.baseApiUrl}/dbs/${props.dbId}/exec-sql-file?db=${props.dbName}&${joinClientParams()}`;
};

const initMonacoEditor = async () => {
    const editorInstance = monacoEditorRef.value?.getEditor();
    if (!editorInstance) {
        return;
    }
    monacoEditor = editorInstance;

    // 上报 model URI：父级（DbDataOp）登记到补全作用域，使多编辑器共存时 provider 按模型路由
    const modelUri = monacoEditor.getModel()?.uri.toString();
    if (modelUri) {
        emits('editorReady', modelUri);
        // 行内注释补全：provider 按 model uri 路由到本编辑器的库上下文。
        // 动态导入是为了不在本组件的静态图里拖入 monaco（同 completion/lazy 的理由）
        inlineDraftUri = modelUri;
        const inline = await import('../sql-ai/inlineDraft');
        // 动态装配 ghost 所需贡献模块：异步但无需等待，首个触发时已完成
        void inline.ensureInlineDraftProvider();
        // 启用态的计算只有一处（syncInlineDraftScope），这里不重复写一遍与 aiEnabled 的合取
        await syncInlineDraftScope(state.inlineDraft);
        inlineDraftRelease = () => inline.removeInlineDraftScope(modelUri);
    }

    // 快捷键常量属于 monaco 运行时。本函数由编辑器的 ready 事件触发，此刻编辑器 chunk 已在内存，
    // 动态导入不会多下一份，却能让本组件的静态图与 monaco 脱钩（否则 SQL 编辑器 tab 要等整份编辑器下载完才出现）
    const { KeyCode, KeyMod } = await import('@/components/monaco/setup');

    // 注册快捷键：ctrl + R 运行选中的sql
    monacoEditor.addAction({
        id: 'run-sql-action' + getKey(),
        label: t('db.runSql'),
        precondition: undefined,
        keybindingContext: undefined,
        keybindings: [KeyMod.chord(KeyMod.CtrlCmd | KeyCode.KeyR, 0)],
        contextMenuGroupId: 'navigation',
        contextMenuOrder: 1.5,
        run: async function () {
            try {
                await onRunSql();
            } catch (e: unknown) {
                e instanceof Error && e.message && Msg.error(e.message);
            }
        },
    });

    // 注册快捷键：ctrl + shift + R 新tab运行选中的sql
    monacoEditor.addAction({
        id: 'run-sql-action-on-newtab' + getKey(),
        label: t('db.newTabRunSql'),
        precondition: undefined,
        keybindingContext: undefined,
        keybindings: [KeyMod.chord(KeyMod.CtrlCmd | KeyMod.Shift | KeyCode.KeyR, 0)],
        contextMenuGroupId: 'navigation',
        contextMenuOrder: 1.6,
        run: async function () {
            try {
                await onRunSql(true);
            } catch (e: unknown) {
                e instanceof Error && e.message && Msg.error(e.message);
            }
        },
    });

    // 注册快捷键：ctrl + shift + enter 执行编辑器内的全部sql（忽略选区与光标）
    monacoEditor.addAction({
        id: 'run-all-sql-action' + getKey(),
        label: t('db.runAllSql'),
        precondition: undefined,
        keybindingContext: undefined,
        keybindings: [KeyMod.chord(KeyMod.CtrlCmd | KeyMod.Shift | KeyCode.Enter, 0)],
        contextMenuGroupId: 'navigation',
        contextMenuOrder: 1.7,
        run: async function () {
            try {
                await onRunAllSql();
            } catch (e: unknown) {
                e instanceof Error && e.message && Msg.error(e.message);
            }
        },
    });

    // 注册快捷键：ctrl + shift + f 格式化sql
    monacoEditor.addAction({
        id: 'format-sql-action' + getKey(),
        label: t('db.formatSql'),
        precondition: undefined,
        keybindingContext: undefined,
        keybindings: [KeyMod.chord(KeyMod.CtrlCmd | KeyMod.Shift | KeyCode.KeyF, 0)],
        contextMenuGroupId: 'navigation',
        contextMenuOrder: 2,
        run: async function () {
            try {
                await onFormatSql();
            } catch (e: unknown) {
                e instanceof Error && e.message && Msg.error(e.message);
            }
        },
    });

    void registerAiActions();

    // 注册快捷键：ctrl + s 保存sql
    monacoEditor.addAction({
        id: 'save-sql-action' + getKey(),
        label: t('db.saveSql'),
        precondition: undefined,
        keybindingContext: undefined,
        keybindings: [KeyMod.chord(KeyMod.CtrlCmd | KeyCode.KeyS, 0)],
        contextMenuGroupId: 'navigation',
        contextMenuOrder: 3,
        run: async function () {
            await saveSql();
        },
    });
};

defineExpose({
    active,
});
</script>

<style lang="scss" scoped>
/* 垂直分割面板 flex 最小尺寸修复：同 ResourceOp 水平分割的 min-width:0 修复，
 * 纵向 flex 对应 min-height:0，允许面板内容（Monaco/表格）不阻止面板收缩 */
:deep(.el-splitter__vertical > .el-splitter-panel) {
    min-height: 0;
}
</style>

<style lang="scss">
.sql-file-exec {
    display: inline-flex;
    flex-direction: row;
    align-items: center;
    justify-content: center;
    vertical-align: middle;
    position: relative;
    text-decoration: none;
}

.editor-move-resize {
    cursor: n-resize;
    height: 3px;
    text-align: center;
}

.sql-exec-res {
    .el-tabs__header {
        margin: 0 0 !important;
    }

    .el-tabs__item {
        font-size: 12px;
        height: 25px;
        margin: 0px;
        padding: 0 6px !important;
    }
}

/*
 * monaco hover（如 find 挂件按钮 tooltip）会向上弹出 splitter 面板顶边，
 * 而 .el-splitter-panel 默认 overflow:auto 会把溢出部分裁剪掉（视觉上像被工具栏按钮遮挡）。
 * hover 显示期间解除裁剪，使 tooltip 能悬浮到上层不被裁剪；
 * :has() 仅显示期间精确匹配，且编辑器高度已恰好填满面板（无滚动条），切换不会引起布局抖动。
 */
.el-splitter-panel:has(.monaco-hover) {
    overflow: visible;
}
</style>
