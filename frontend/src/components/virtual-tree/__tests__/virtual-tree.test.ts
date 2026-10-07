/**
 * VirtualTree 契约测试
 *
 * 三层验证，确保「容器只认语义化契约、引擎差异全收口适配器」这一约束成立：
 *   1. 容器（stub 适配器）：语义化 props 转发、归一化实例方法转发、受控展开兜底、归一化事件回传；
 *   2. elTreeV2Adapter（纯函数）：字段映射、check 全量口径 → 叶子口径归一、实例方法翻译；
 *   3. VirtualTree × 真实 el-tree-v2 集成：首验两项引擎行为差异——
 *      ① expandedKeys 受控性（追加数据后展开态保留）；② checkOnClickLeaf 勾选 + check 事件叶子归一。
 */
import { mount } from '@vue/test-utils';
import { defineComponent, h, markRaw, nextTick } from 'vue';
import { ElTreeV2 } from 'element-plus';
import { describe, expect, it, vi } from 'vitest';

import { VirtualTree } from '../index';
import { elTreeV2Adapter } from '../adapters';
import type { TreeAdapterPropsInput, VirtualTreeAdapter, VirtualTreeInstance } from '../adapters';

// happy-dom 下 el-tree-v2 的虚拟列表需要确定像素高度才渲染行
const H = 300;

/** 一个父节点 + 两个叶子；父节点持有非空 children（引擎据此判定非叶子） */
const makeTreeData = () => [
    {
        key: 'p',
        label: 'parent',
        children: [
            { key: 'c1', label: 'child-one' },
            { key: 'c2', label: 'child-two' },
        ],
    },
];

/** 构造记录型 stub 适配器：捕获 getProps 入参、暴露可断言的实例 spy */
function makeStubAdapter() {
    const instance: VirtualTreeInstance = {
        setCurrentKey: vi.fn(),
        scrollToNode: vi.fn(),
        scrollTo: vi.fn(),
        getCheckedKeys: vi.fn(() => ['leaf-a']),
        setCheckedKeys: vi.fn(),
        setExpandedKeys: vi.fn(),
        filter: vi.fn(),
    };
    const inputs: TreeAdapterPropsInput[] = [];
    const adapter: VirtualTreeAdapter = {
        name: 'stub',
        component: markRaw(defineComponent({ name: 'StubTree', setup: () => () => h('div', { class: 'stub-tree' }) })),
        getProps: (input) => {
            inputs.push(input);
            return {};
        },
        getInstance: () => instance,
    };
    return { adapter, instance, inputs, latest: () => inputs[inputs.length - 1] };
}

describe('VirtualTree 容器契约（stub 适配器）', () => {
    it('语义化 props 原样交给适配器，容器不出现引擎 prop 名', () => {
        const stub = makeStubAdapter();
        const fieldNames = { value: 'id', label: 'name', children: 'kids' };
        mount(VirtualTree, {
            props: {
                data: makeTreeData(),
                adapter: stub.adapter,
                fieldNames,
                height: H,
                checkable: true,
                checkOnClickLeaf: true,
                highlightCurrent: false,
                expandOnClickNode: true,
                expandedKeys: ['p'],
            },
        });

        const input = stub.latest();
        expect(input.checkable).toBe(true);
        expect(input.checkOnClickLeaf).toBe(true);
        expect(input.highlightCurrent).toBe(false);
        expect(input.expandOnClickNode).toBe(true);
        expect(input.fieldNames).toEqual(fieldNames);
        expect(input.expandedKeys).toEqual(['p']);
        expect(input.height).toBe(H);
    });

    it('归一化实例方法转发到适配器实例', () => {
        const stub = makeStubAdapter();
        const wrapper = mount(VirtualTree, { props: { data: makeTreeData(), adapter: stub.adapter, height: H } });

        wrapper.vm.setCurrentKey('c1');
        wrapper.vm.scrollToNode('c1', 'center');
        wrapper.vm.scrollTo(120);
        wrapper.vm.setCheckedKeys(['c1', 'c2']);
        wrapper.vm.setExpandedKeys(['p']);
        wrapper.vm.filter('foo');

        expect(stub.instance.setCurrentKey).toHaveBeenCalledWith('c1');
        expect(stub.instance.scrollToNode).toHaveBeenCalledWith('c1', 'center');
        expect(stub.instance.scrollTo).toHaveBeenCalledWith(120);
        expect(stub.instance.setCheckedKeys).toHaveBeenCalledWith(['c1', 'c2']);
        expect(stub.instance.setExpandedKeys).toHaveBeenCalledWith(['p']);
        expect(stub.instance.filter).toHaveBeenCalledWith('foo');
        expect(wrapper.vm.getCheckedKeys(true)).toEqual(['leaf-a']);
        expect(stub.instance.getCheckedKeys).toHaveBeenCalledWith(true);
    });

    it('受控展开兜底：expandedKeys 变化后回写引擎 setExpandedKeys', async () => {
        const stub = makeStubAdapter();
        const wrapper = mount(VirtualTree, { props: { data: makeTreeData(), adapter: stub.adapter, height: H, expandedKeys: ['p'] } });

        (stub.instance.setExpandedKeys as ReturnType<typeof vi.fn>).mockClear();
        await wrapper.setProps({ expandedKeys: ['p', 'c1'] });
        await nextTick();
        await nextTick();

        expect(stub.instance.setExpandedKeys).toHaveBeenCalledWith(['p', 'c1']);
    });

    it('引擎事件经归一化后由容器回传', async () => {
        const stub = makeStubAdapter();
        const wrapper = mount(VirtualTree, { props: { data: makeTreeData(), adapter: stub.adapter, height: H } });

        const events = stub.latest().events;
        events.onNodeClick?.({ key: 'c1' });
        events.onNodeExpand?.({ key: 'p' });
        events.onNodeCollapse?.({ key: 'p' });
        events.onCheck?.(['c1', 'c2']);
        await nextTick();

        expect(wrapper.emitted('nodeClick')?.[0]).toEqual([{ key: 'c1' }]);
        expect(wrapper.emitted('nodeExpand')?.[0]).toEqual([{ key: 'p' }]);
        expect(wrapper.emitted('nodeCollapse')?.[0]).toEqual([{ key: 'p' }]);
        expect(wrapper.emitted('check')?.[0]).toEqual([['c1', 'c2']]);
    });
});

describe('elTreeV2Adapter 归一化（纯函数）', () => {
    const baseInput = (over: Partial<TreeAdapterPropsInput> = {}): TreeAdapterPropsInput => ({
        data: makeTreeData(),
        fieldNames: { value: 'key', label: 'label', children: 'children', disabled: 'disabled' },
        height: H,
        expandedKeys: ['p'],
        checkable: true,
        checkOnClickLeaf: true,
        highlightCurrent: true,
        expandOnClickNode: false,
        events: {},
        ...over,
    });

    it('语义 props → el-tree-v2 引擎 props（字段映射 / 勾选 / 展开）', () => {
        const props = elTreeV2Adapter.getProps(baseInput());
        expect(props.props).toEqual({ value: 'key', label: 'label', children: 'children', disabled: 'disabled' });
        expect(props.showCheckbox).toBe(true);
        expect(props.checkOnClickLeaf).toBe(true);
        expect(props.defaultExpandedKeys).toEqual(['p']);
        expect(props.height).toBe(H);
    });

    it('check 事件全量口径 → 叶子 key 列表', () => {
        const onCheck = vi.fn();
        const props = elTreeV2Adapter.getProps(baseInput({ events: { onCheck } }));
        const handler = props.onCheck as (data: unknown, info: { checkedNodes?: Record<string, unknown>[] }) => void;

        const data = makeTreeData();
        const parent = data[0];
        const leaf = parent.children[0];
        // 引擎给全量勾选节点（含父），适配器只保留叶子
        handler(leaf, { checkedNodes: [parent, leaf] });

        expect(onCheck).toHaveBeenCalledWith(['c1']);
    });

    it('getInstance 翻译引擎实例方法并归一 getCheckedKeys 为字符串', () => {
        const engine = {
            setCurrentKey: vi.fn(),
            scrollToNode: vi.fn(),
            scrollTo: vi.fn(),
            getCheckedKeys: vi.fn(() => [1, 2]),
            setCheckedKeys: vi.fn(),
            setExpandedKeys: vi.fn(),
            filter: vi.fn(),
        };
        const inst = elTreeV2Adapter.getInstance({ value: engine });

        inst.setCurrentKey('c1');
        expect(engine.setCurrentKey).toHaveBeenCalledWith('c1');
        expect(inst.getCheckedKeys(true)).toEqual(['1', '2']);
        expect(engine.getCheckedKeys).toHaveBeenCalledWith(true);
    });

    it('引擎实例为空时归一化实例方法安全 no-op', () => {
        const inst = elTreeV2Adapter.getInstance({ value: null });
        expect(() => inst.setCurrentKey('x')).not.toThrow();
        expect(inst.getCheckedKeys()).toEqual([]);
    });
});

describe('VirtualTree × 真实 el-tree-v2 集成（引擎行为首验）', () => {
    it('expandedKeys 受控性：追加数据后展开态保留', async () => {
        const wrapper = mount(VirtualTree, {
            props: { data: makeTreeData(), height: H, expandedKeys: ['p'] },
        });
        await nextTick();
        await nextTick();

        // 初始：父节点展开，叶子已渲染
        expect(wrapper.html()).toContain('child-one');

        // 追加一个新顶层节点（数据重建），同时保持 expandedKeys=['p']（新引用触发容器兜底同步）
        const appended = [...makeTreeData(), { key: 'p2', label: 'parent-two', children: [{ key: 'c3', label: 'child-three' }] }];
        await wrapper.setProps({ data: appended, expandedKeys: ['p'] });
        await nextTick();
        await nextTick();

        // 'p' 仍处于展开态：其叶子未丢失
        expect(wrapper.html()).toContain('child-one');
    });

    it('checkOnClickLeaf 勾选：check 事件归一为叶子 key', async () => {
        const wrapper = mount(VirtualTree, {
            props: { data: makeTreeData(), height: H, checkable: true, checkOnClickLeaf: true, expandedKeys: ['p'] },
        });
        await nextTick();
        await nextTick();

        const treeV2 = wrapper.findComponent(ElTreeV2);
        expect(treeV2.exists()).toBe(true);

        const data = makeTreeData();
        const parent = data[0];
        const leaf = parent.children[0];
        // 模拟引擎在点击叶子后回传的 check 事件（全量勾选节点含父）
        treeV2.vm.$emit('check', leaf, {
            checkedNodes: [parent, leaf],
            checkedKeys: ['p', 'c1'],
            halfCheckedKeys: [],
            halfCheckedNodes: [],
        });
        await nextTick();

        expect(wrapper.emitted('check')?.[0]).toEqual([['c1']]);
    });
});
