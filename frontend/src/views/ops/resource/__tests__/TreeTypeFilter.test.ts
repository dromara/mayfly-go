import { mount } from '@vue/test-utils';
import { ElDropdown } from 'element-plus';
import { expect, it, vi } from 'vitest';

// SvgIcon 带样式块会触发 postcss 加载；本组件只关心「显示了哪个类型」
vi.mock('@/components/svg-icon/index.vue', () => ({ default: { name: 'SvgIcon', render: () => null } }));

import { registerContributor } from '../tree/registry';
import { registerResource } from '../resourceRegistry';
import TreeTypeFilter from '../TreeTypeFilter.vue';

/**
 * 类型筛选控件契约：选项由「资源注册表 + 根贡献者」推导（新增资产零改动接入），
 * 未筛选态是图标、生效态必须把当前类型写在控件上（窄侧栏自解释要求），命令回传根 kind。
 */

const TYPE = 'ut-filter-type';
const KIND = 'ut-filter-kind';

registerResource(TYPE, { resourceType: TYPE, manager: { componentConf: { name: 'tag.machine' } } });
registerContributor({ kind: KIND, resourceType: TYPE });
// 无根贡献者的资源类型不会出现在选项里，见最后一条用例（传未知 kind 回落图标态）

const mountFilter = (modelValue: string | null = null) =>
    mount(TreeTypeFilter, {
        props: { modelValue },
        global: { mocks: { $t: (key: string) => key }, components: { ElDropdown } },
    });

it('未筛选态只有图标入口，生效态把当前类型写在控件上', () => {
    expect(mountFilter(null).find('.tree-type-trigger').exists()).toBe(true);
    expect(mountFilter(null).find('.tree-type-chip').exists()).toBe(false);

    const active = mountFilter(KIND);
    expect(active.find('.tree-type-chip').exists()).toBe(true);
    expect(active.text()).toContain('tag.machine');
});

it('选择项上抛根 kind，选「全部资源」上抛 null', () => {
    const w = mountFilter(null);
    const dropdown = w.findComponent(ElDropdown);

    dropdown.vm.$emit('command', KIND);
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([KIND]);

    dropdown.vm.$emit('command', '');
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual([null]);
});

it('生效值不在选项里时回落图标态（不显示空胶囊；选项推导见 tagResources 单测）', () => {
    expect(mountFilter('ut-unknown-kind').find('.tree-type-chip').exists()).toBe(false);
});
