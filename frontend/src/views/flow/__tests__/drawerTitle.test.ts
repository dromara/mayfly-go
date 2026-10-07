import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

/**
 * 抽屉标题必须有兜底。
 *
 * 流程设计与节点属性两个抽屉长期只有「返回」箭头、没有名字：调用方漏传 `title`，
 * 组件的 `title` prop 又没有默认值，界面看着像坏了却不报任何错，tsc / eslint / build 全都放行。
 * 这里守住「绑定的是带 i18n 兜底的值」，避免下次再加抽屉时重演
 */
describe('设计器抽屉的标题兜底', () => {
    const cases = [
        ['流程设计', 'src/views/flow/components/flowdesign/FlowDesignDrawer.vue', 'flow.flowDesign'],
        ['节点属性', 'src/views/flow/components/flowdesign/node/PropSettingDrawer.vue', 'flow.nodeProperty'],
    ] as const;

    for (const [name, file, key] of cases) {
        it(`${name}抽屉即使调用方不传 title 也有标题`, () => {
            const source = readFileSync(join(process.cwd(), file), 'utf8');
            expect(source, '抽屉头部绑定的不是带兜底的 headerTitle').toMatch(/:header="headerTitle"/);
            expect(source, '缺少指向已存在 i18n key 的兜底标题').toMatch(new RegExp(`props\\.title\\s*\\|\\|\\s*t\\('${key}'\\)`));
        });
    }
});
