/**
 * 芯片序列化的双向契约（文本形态 ↔ 段数据）
 *
 * 为什么单独测：芯片是原子节点，一旦文本形态缺失，任务指令入库的字符串里引用会**凭空消失**
 * （句子读起来缺主语，模型也不知道要对谁执行）；而编辑回填依赖 fromSegment 还原，
 * 还原不出来时必须降级为纯文本，不能造一个点开定位不到资源的假芯片。
 */
import { describe, expect, it } from 'vitest';
import { extractSegmentsFromNode, getChipType } from '../chipRegistry';
// 副作用导入：注册 skill/resource 芯片类型
import '../chipTypes';

describe('芯片文本形态 toText', () => {
    it('技能芯片输出 $技能码（与内置 Runbook 里的引用写法同一约定）', () => {
        expect(getChipType('skill')!.toText({ label: '数据库操作手册', data: { id: 'db-ops-guide' } })).toBe('$db-ops-guide');
    });

    it('资源芯片输出 @标签（定位信息在段 extra 里，派发时展开）', () => {
        expect(getChipType('resource')!.toText({ label: 'aliyun', data: { resourceType: 'machine', id: 11 } })).toBe('@aliyun');
    });
});

describe('段提取的往返保真', () => {
    /** 构造 TipTap 文档形状：一个段落内「文本-芯片-文本-芯片-文本」交替 */
    const docWithAdjacentChips = () => {
        const children = [
            { type: { name: 'text' }, text: '巡检 ' },
            { type: { name: 'chip' }, attrs: { type: 'resource', label: '机器A', data: { resourceType: 'machine' } } },
            // 芯片选中时 insertSelection 补的分隔空格：这一段必须活着走完往返
            { type: { name: 'text' }, text: ' ' },
            { type: { name: 'chip' }, attrs: { type: 'resource', label: '机器B', data: { resourceType: 'machine' } } },
            { type: { name: 'text' }, text: ' 的状态' },
        ];
        return {
            content: {
                forEach: (fn: (block: unknown) => void) =>
                    fn({ content: { forEach: (cf: (child: unknown) => void) => children.forEach(cf) } }),
            },
        };
    };

    it('相邻芯片之间的空白段不得被丢弃', () => {
        const segs = extractSegmentsFromNode(docWithAdjacentChips());
        // 两个 chip 之间必须仍有一个纯空格文本段（旧实现按 trim 过滤会吃掉它）
        const gap = segs[2];
        expect(gap.type).toBe('input_text');
        expect(gap.text).toBe(' ');
    });

    it('段序列拼回的文本与编辑器文本形态一致（不粘连、不丢字）', () => {
        const segs = extractSegmentsFromNode(docWithAdjacentChips());
        const rebuilt = segs
            .map((s) => (s.type === 'input_text' ? s.text : (getChipType(s.type)?.toText({ label: s.text, data: s.extra ?? {} }) ?? s.text)))
            .join('');
        expect(rebuilt).toBe('巡检 @机器A @机器B 的状态');
    });
});

describe('芯片回填 fromSegment', () => {
    it('与 extractSegment 互为逆运算（技能）', () => {
        const attrs = { type: 'skill', label: '数据库操作手册', data: { id: 'db-ops-guide' } };
        const seg = getChipType('skill')!.extractSegment(attrs);
        expect(getChipType('skill')!.fromSegment(seg)).toEqual({ type: 'skill', label: '数据库操作手册', data: { id: 'db-ops-guide' } });
    });

    it('资源段保留完整定位信息（machineId/ip/port/db 等一个都不能掉）', () => {
        const attrs = { type: 'resource', label: 'aliyun', data: { resourceType: 'machine', id: '11', code: 'm11', ip: '10.0.0.11', port: '22' } };
        const restored = getChipType('resource')!.fromSegment(getChipType('resource')!.extractSegment(attrs))!;

        expect(restored.type).toBe('resource');
        expect(restored.label).toBe('aliyun');
        // 逐字段断言而非整体相等：注册表按定位白名单归一（未提供的字段补空串），
        // 那是「渲染器要读哪些键」的契约，不是数据丢失
        expect(restored.data).toMatchObject({ resourceType: 'machine', id: '11', code: 'm11', ip: '10.0.0.11', port: '22' });
    });

    it('缺定位信息的段返回 null（调用方按纯文本降级，不造假芯片）', () => {
        expect(getChipType('resource')!.fromSegment({ type: 'resource', text: 'x', extra: {} })).toBeNull();
        expect(getChipType('skill')!.fromSegment({ type: 'skill', text: 'x', extra: {} })).toBeNull();
    });

    it('未知芯片类型查不到注册项（渲染层据此回落标签文本）', () => {
        expect(getChipType('unknown')).toBeUndefined();
    });
});
