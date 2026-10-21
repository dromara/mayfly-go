import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

/**
 * 提案详情抽屉的「审批链可达」接线守卫。
 *
 * 存在理由：提案是「诊断 → 提案 → 审批 → 受控执行 → 验证」闭环的事实源，它记着四个 id：
 * 来源运行、审批工单、执行运行、校验运行。这四个值此前在详情里是**裸数字**——只证明
 * 「有这条记录」，拿不到证据：想看谁批的、执行到底落了什么、校验结论是否成立，都得
 * 回到别的列表页按 id 碰运气（而这些 id 并未作为筛选维度开放）。审计链断在展示层。
 *
 * 这一类「字段在、数字对、就是点不开」的缺陷编译与类型检查全绿，界面截图也看不出问题，
 * 只能把接线钉成守卫：值必须渲染在链接里、必须接到正确的跳转函数、跳转目标容器必须真的挂着。
 */
const readSource = (name: string) => readFileSync(resolve(__dirname, '..', name), 'utf8');

const readI18n = (lang: string) => readFileSync(resolve(__dirname, '..', '..', '..', 'i18n', lang, 'agentops.ts'), 'utf8');

const source = readSource('ProposalList.vue');

/** 取某个描述项的整块源码（开标签到闭合标签）。匹配不到就抛错，避免「没匹配到 → 断言恒真」 */
function itemBlock(field: string): string {
    const block = source.match(new RegExp(`<el-descriptions-item[^>]*v-if="detail\\.data\\.${field}"[\\s\\S]*?</el-descriptions-item>`));
    if (!block) {
        throw new Error(`未找到 detail.data.${field} 的描述项（改名或删掉必须同步本守卫）`);
    }
    return block[0];
}

/** [字段, 期望的跳转函数, 期望的标签 i18n 键] */
const AUDIT_CHAIN: Array<[string, string, string]> = [
    ['runId', 'openRunDetail', 'agentops.sourceRunId'],
    ['procInstId', 'openProcInst', 'agentops.procInstId'],
    ['execRunId', 'openRunDetail', 'agentops.execRunId'],
    ['verifyRunId', 'openRunDetail', 'agentops.verifyRunId'],
];

describe.each(AUDIT_CHAIN)('提案详情的 %s 必须可点开', (field, handler, labelKey) => {
    it('值渲染在 el-link 内并接到正确的跳转函数，而非裸数字', () => {
        const block = itemBlock(field);

        // 精确到「值落在 el-link 内部」：只断言含 el-link 会被兄弟节点上的链接蒙过去
        expect(
            block,
            `detail.data.${field} 必须渲染成可点开的链接（裸数字等于给不了证据，只能回别的页面按 id 碰运气）`,
        ).toMatch(new RegExp(`<el-link[^>]*>\\{\\{ detail\\.data\\.${field} \\}\\}</el-link>`));
        expect(block, `${field} 必须跳到 ${handler}`).toContain(`@click="${handler}(detail.data.${field})"`);
    });

    it('标签走 i18n 且中英文都登记了该键（裸显 key 是同一类静默缺陷）', () => {
        expect(itemBlock(field), `${field} 的标签必须用 $t`).toContain(`:label="$t('${labelKey}')"`);
        const shortKey = labelKey.split('.')[1];
        for (const lang of ['zh-cn', 'en']) {
            expect(readI18n(lang), `${lang}/agentops.ts 缺少 ${labelKey}`).toContain(`${shortKey}:`);
        }
    });
});

it('两个跳转目标容器必须真的挂在模板上并绑到对应状态', () => {
    // 处理函数只置 state、容器没挂或没绑同一个 state，表现是「点了没反应」——
    // 这类漏接比漏写函数更难在评审里看出来，故单独钉
    expect(source).toMatch(/<ProcinstDetail[\s\S]*?v-model:visible="procDetail\.visible"[\s\S]*?:procinst-id="procDetail\.procinstId"/);
    expect(source).toMatch(/<RunDetailDrawer[\s\S]*?v-model:visible="runDetail\.visible"[\s\S]*?:run-id="runDetail\.runId"/);
});

it('行内「回放」与详情里的来源运行链接必须走同一个抽屉', () => {
    // 两个入口打开两种界面，就会出现「同一份轨迹有两个真源」的漂移（此前正是从「回放」
    // 跳进 AI 助手、与执行历史里的只读轨迹不一致）
    const replay = source.match(/const openRunReplay = \(row: AgentProposal\) => (\w+)\(row\.runId\)/);
    expect(replay, '行内回放应复用 openRunDetail').toBeDefined();
    expect(replay![1]).toBe('openRunDetail');
});
