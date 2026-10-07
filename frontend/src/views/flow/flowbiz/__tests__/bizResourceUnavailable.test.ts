import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it, vi } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';

/**
 * 审批面板在「看不到资源信息」时必须优雅降级。
 *
 * 工单候选人不必是该资源的可见用户（会签的第二人、跨团队审批都属正常），实例也可能早已被删除。
 * 原实现把 `${redis?.host}` 直接渲染成字符串 "undefined"，还会在列表为空时读 `state.redis.code`
 * 抛 TypeError —— 面板被打断，审批人连自己要批的命令都可能看不清。
 */

const unavailableHint = '未找到该资源信息';

const redisListRequest = vi.fn();
const tagListRequest = vi.fn();

vi.mock('@/views/ops/redis/api', () => ({
    redisApi: { redisList: { request: (params: unknown) => redisListRequest(params) } },
}));

vi.mock('@/views/ops/tag/api', () => ({
    tagApi: { listByQuery: { request: (params: unknown) => tagListRequest(params) } },
}));

const { default: RedisRunCmdBiz } = await import('@/views/flow/flowbiz/redis/RedisRunCmdBiz.vue');

const i18n = createI18n({
    legacy: false,
    locale: 'zh',
    messages: { zh: { flow: { resourceUnavailable: unavailableHint }, common: { tag: '标签', code: '标识', name: '名称' } } },
});

function mountBiz(procinst: Record<string, unknown>) {
    return mount(RedisRunCmdBiz, {
        props: { procinst },
        global: { plugins: [ElementPlus, i18n], stubs: { TagCodePath: true, SvgIcon: true } },
    });
}

describe('Redis 审批面板的资源信息降级', () => {
    it('查不到实例时给出明确提示，且不渲染 undefined', async () => {
        redisListRequest.mockResolvedValue({ list: [] });
        tagListRequest.mockResolvedValue([]);

        const wrapper = mountBiz({ bizForm: JSON.stringify({ id: 9, db: 0, cmd: 'SET a b' }), bizHandleRes: '' });
        await flushPromises();

        expect(wrapper.text()).toContain(unavailableHint);
        expect(wrapper.text()).not.toContain('undefined');
        // 实例都没有，就不该再拿空 code 去查标签路径（那行代码以前会抛 TypeError）
        expect(tagListRequest).not.toHaveBeenCalled();
    });

    it('查得到实例时照常展示资源信息', async () => {
        redisListRequest.mockResolvedValue({ list: [{ code: 'Rc1', name: '本地', host: 'localhost', port: 6332, mode: 'standalone' }] });
        tagListRequest.mockResolvedValue([{ codePath: 'default/3|Rc1/' }]);

        const wrapper = mountBiz({ bizForm: JSON.stringify({ id: 9, db: 0, cmd: 'SET a b' }), bizHandleRes: '' });
        await flushPromises();

        expect(wrapper.text()).toContain('localhost');
        expect(wrapper.text()).toContain('standalone');
        expect(wrapper.text()).not.toContain(unavailableHint);
        expect(tagListRequest).toHaveBeenCalledTimes(1);
    });

    it('工单没带业务表单时也要说明情况，而不是静默渲染一排空字段', async () => {
        // 初始值写成空对象就能过前面的用例：只有这条路能区分「没数据」与「数据是空对象」
        const wrapper = mountBiz({ bizForm: '', bizHandleRes: '' });
        await flushPromises();

        expect(wrapper.text()).toContain(unavailableHint);
        expect(wrapper.text()).not.toContain('undefined');
        expect(redisListRequest).not.toHaveBeenCalled();
    });

    it('命令文本与库号始终要可见（审批靠它判断）', async () => {
        redisListRequest.mockResolvedValue({ list: [] });

        const wrapper = mountBiz({ bizForm: JSON.stringify({ id: 9, db: 3, cmd: 'FLUSHALL' }), bizHandleRes: '' });
        await flushPromises();

        const textarea = wrapper.find('textarea');
        expect(textarea.exists()).toBe(true);
        expect(textarea.element.value).toBe('FLUSHALL');
    });
});

describe('数据库审批面板同构守卫', () => {
    // 该面板内嵌 Monaco 编辑器，挂载成本高，按源码接线断言守同一件事
    const source = readFileSync(join(import.meta.dirname, '../dbms/DbSqlExecBiz.vue'), 'utf8');

    it('资源信息按「取不到」分支渲染，不留 undefined 字面量', () => {
        expect(source).toMatch(/<template v-if="db">/);
        expect(source).toContain("$t('flow.resourceUnavailable')");
        expect(source).not.toContain('${db?.host}');
        expect(source).toMatch(/db: null as any/);
    });

    it('查不到实例就提前返回，不再解引用 code', () => {
        const block = source.slice(source.indexOf('const dbRes'), source.indexOf('const dbRes') + 500);
        expect(block).toMatch(/if \(!state\.db\) \{/);
        expect(block.indexOf('if (!state.db)')).toBeLessThan(block.indexOf('state.db.code'));
    });
});
