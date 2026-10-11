import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';

import type { SqlExplainPlan } from '../../types';
import SqlPlanView from '../SqlPlanView.vue';

/**
 * 执行计划视图的呈现守卫
 *
 * 钉的是「不许把观测缺口画成好消息」：拿不到计划时必须走说明分支，
 * 而不是渲染一张空节点表；行数是优化器估算这件事必须常驻在界面上；
 * 层级与全表扫描标记都只转译后端给的事实，前端不再判一遍。
 */

// 空字典：t() 原样返回键名，断言直接针对键，改文案不会误伤判据
const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });

function mountView(plan: SqlExplainPlan) {
    return mount(SqlPlanView, { props: { plan }, global: { plugins: [i18n, ElementPlus] } });
}

describe('SqlPlanView 计划呈现', () => {
    it('拿不到计划时只给说明，不渲染节点区（空图会被读成「没有扫描」）', () => {
        const wrapper = mountView({ sql: 'DELETE FROM t', stmtType: 'delete', note: 'plan unavailable' });
        expect(wrapper.text()).toContain('plan unavailable');
        expect(wrapper.findAll('.plan-node')).toHaveLength(0);
    });

    it('既无计划也无说明时给明确空态，不留白', () => {
        const wrapper = mountView({ sql: 'SELECT 1', stmtType: 'select' });
        expect(wrapper.text()).toContain('db.planNone');
        expect(wrapper.findAll('.plan-node')).toHaveLength(0);
    });

    it('估算口径与「未执行语句」必须常驻，全表扫描节点按后端标记高亮', () => {
        const wrapper = mountView({
            sql: 'SELECT * FROM t WHERE name = 1',
            stmtType: 'select',
            result: {
                rows: 1000,
                estimateKind: 'explain',
                fullScan: true,
                plan: [
                    { nodeKind: 'ALL', table: 't', estRows: 1000, fullScan: true },
                    { nodeKind: 'index', table: 't', estRows: 3, fullScan: false },
                ],
            },
        });
        const text = wrapper.text();
        expect(text).toContain('db.planRowsEstimated');
        expect(text).toContain('db.planNotExecuted');
        expect(text).toContain('db.planFullScan');

        const nodes = wrapper.findAll('.plan-node');
        expect(nodes).toHaveLength(2);
        expect(nodes[0].classes()).toContain('is-full-scan');
        expect(nodes[1].classes()).not.toContain('is-full-scan');
    });

    it('平铺且多节点时说明「该方言没给父子关系」，单节点不必提示', () => {
        const flat = mountView({
            sql: 'SELECT 1',
            stmtType: 'select',
            result: {
                rows: 2,
                estimateKind: 'explain',
                fullScan: false,
                plan: [
                    { nodeKind: 'ALL', estRows: 2 },
                    { nodeKind: 'const', estRows: 0 },
                ],
            },
        });
        expect(flat.text()).toContain('db.planFlatHint');

        const single = mountView({
            sql: 'SELECT 1',
            stmtType: 'select',
            result: { rows: 1, estimateKind: 'explain', fullScan: false, plan: [{ nodeKind: 'ALL', estRows: 1 }] },
        });
        expect(single.text()).not.toContain('db.planFlatHint');
    });

    it('层级按后端 children 缩进，前端不按 nodeKind 文本重算结构', () => {
        const wrapper = mountView({
            sql: 'SELECT 1',
            stmtType: 'select',
            result: {
                rows: 10,
                estimateKind: 'explain',
                fullScan: false,
                plan: [
                    { nodeKind: 'Hash Join', estRows: 10, children: [1, 2] },
                    { nodeKind: 'Seq Scan', estRows: 100, fullScan: true },
                    { nodeKind: 'Index Scan', estRows: 1, indexName: 'idx_a' },
                ],
            },
        });
        const nodes = wrapper.findAll('.plan-node');
        expect(nodes[0].attributes('style')).toContain('margin-left: 0px');
        expect(nodes[1].attributes('style')).toContain('margin-left: 16px');
        expect(nodes[2].attributes('style')).toContain('margin-left: 16px');
        // 索引名照实呈现，供用户核对「到底走了哪个索引」
        expect(nodes[2].text()).toContain('idx_a');
    });
});
