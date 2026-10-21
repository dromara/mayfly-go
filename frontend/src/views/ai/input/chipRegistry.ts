/**
 * chipRegistry - 芯片类型注册表
 *
 * 职责：芯片的视觉呈现（图标/配色）与数据提取（extractSegment）
 * 新增芯片类型只需调用 registerChipType()，零修改编辑器/消息渲染代码
 * （触发字符管理在 triggers/ 子系统，与呈现解耦）
 */
import type { ContentSegmentType } from '../protocol/types';

/** 芯片段：发送时提取的数据结构（type 对齐协议层 ContentSegmentType，与后端同形） */
export interface ChipSegment {
    type: ContentSegmentType;
    text: string;
    extra: Record<string, unknown>;
}

/** 图标解析入参：不直接用 chipNode 的 ChipNodeAttrs，避免注册表反向依赖节点定义造成循环 */
export interface ChipIconContext {
    label: string;
    data: Record<string, unknown>;
}

/**
 * 芯片图标可用的 mask 名集合（与 chipStyles.scss 里的 `.chat-chip__icon[data-icon=…]` 一一对应）。
 *
 * CSS 无法由注册表生成，所以这里显式登记并靠单测卡住漏写：新增图标名而忘了补 mask，
 * 表现只是「芯片里少个图标」，没有任何报错
 */
export const SUPPORTED_CHIP_ICONS = ['zap', 'server', 'database'] as const;

/** 芯片类型注册信息（消费方：chipNode 编辑器渲染 + InlineChip 消息回显，配色/图标同源） */
export interface ChipTypeRegistration {
    /** 芯片类型标识 */
    readonly type: string;
    /**
     * 图标解析（返回 chipStyles 支持的 mask 名）。
     *
     * 定成函数而不是固定字符串：resource 这类芯片的图标要看子类型（机器=server、数据库=database），
     * 此前该判定在 chipNode 与 InlineChip 各写一遍，两处一旦分叉就是「编辑器里一个样、
     * 消息气泡里另一个样」
     */
    readonly icon: (ctx: ChipIconContext) => string;
    /** 配色 */
    readonly color: {
        bg: string;
        text: string;
        border: string;
    };
    /** 从芯片节点提取段数据（发送时序列化） */
    extractSegment(nodeAttrs: Record<string, unknown>): ChipSegment;
    /**
     * 芯片的纯文本形态（编辑器 getText() 与落库的指令文本用它表示这个引用）。
     *
     * 没有它时 TipTap 会把原子节点渲染成空串：复制粘贴、只读回显、以及任务指令入库的
     * 文本形态都会「芯片凭空消失」，剩下的句子读起来就缺主语
     */
    toText(nodeAttrs: Record<string, unknown>): string;
    /**
     * 由段数据还原芯片节点属性（编辑回填：任务指令/队列消息重新进编辑器时用）。
     * 无法还原（如未知类型）时返回 null，调用方按纯文本段降级展示，不能凭空造一个假引用
     */
    fromSegment(seg: ChipSegment): { type: string; label: string; data: Record<string, unknown> } | null;
}

const registry = new Map<string, ChipTypeRegistration>();

/**
 * 未注册芯片类型的兜底配色（与注册项的 color 同形）。
 *
 * 收在注册表而不是写在 chipNode 里：芯片配色的真源就是注册表，兜底色写在渲染端
 * 就成了第二处「颜色从哪来」，改主题时容易只改一处。
 * 取值用主题变量而不是十六进制：注册项全部是 var(--el-color-*)，兜底写死亮白块
 * 会在暗色主题下变成刺眼白块（同一张表两种介质）
 */
export const FALLBACK_CHIP_COLOR = {
    bg: 'var(--el-fill-color-light)',
    text: 'var(--el-text-color-regular)',
    border: 'var(--el-border-color)',
};

/**
 * 段序列 → 展示/提交文本（单一真源）。
 *
 * 芯片段必须走注册表的文本形态（`@标签` / `$技能码`）：任何一处自写「只取文本段」的白名单，
 * 都会让产出文本丢掉引用标记（历史重放、队列预览都踩过）。所有「段→文本」需求一律调本函数。
 */
export function segmentsToText(segments: Array<{ type: string; text?: string; extra?: Record<string, unknown> }> | undefined): string {
    if (!Array.isArray(segments)) return '';
    return segments
        .map((s) => {
            const text = s.text ?? '';
            const chip = getChipType(s.type);
            return chip ? chip.toText({ label: text, data: s.extra ?? {} }) : text;
        })
        .join('');
}

/** 注册芯片类型，返回卸载函数 */
export function registerChipType(reg: ChipTypeRegistration): () => void {
    registry.set(reg.type, reg);
    return () => registry.delete(reg.type);
}

/** 获取芯片类型注册 */
export function getChipType(type: string): ChipTypeRegistration | undefined {
    return registry.get(type);
}

/**
 * 芯片图标解析单点（编辑器与消息回显共用）：未注册类型回退 zap，与 chipNode 的历史行为一致
 */
export function chipIcon(type: string, ctx: ChipIconContext = { label: '', data: {} }): string {
    return getChipType(type)?.icon(ctx) ?? 'zap';
}

/** 从文档节点提取所有段 */
export function extractSegmentsFromNode(node: { content: { forEach: (fn: (child: any) => void) => void } }): ChipSegment[] {
    const segments: ChipSegment[] = [];
    const doc = node.content;

    doc.forEach((block: any) => {
        block.content?.forEach((child: any) => {
            if (child.type.name === 'chip') {
                const chipType = child.attrs.type as string;
                const reg = getChipType(chipType);
                if (reg) {
                    segments.push(reg.extractSegment(child.attrs));
                }
            } else if (child.type.name === 'text' && child.text) {
                segments.push({
                    type: 'input_text',
                    text: child.text,
                    extra: {},
                });
            }
        });
    });

    // 合并相邻文本段
    const merged: ChipSegment[] = [];
    for (const seg of segments) {
        if (seg.type === 'input_text' && merged.length > 0) {
            const last = merged[merged.length - 1];
            if (last.type === 'input_text') {
                last.text += seg.text;
                continue;
            }
        }
        merged.push({ ...seg });
    }

    // 只丢真正的空段，纯空白段必须保留：芯片选中时会补一个空格分隔后续输入（见
    // useTriggerState.insertSelection），按 trim 过滤会把段序列里的空格吃掉，相邻两个芯片
    // 回填时粘成 `@a@b`，再经受控同步把变形文本写回表单——用户保存到的就是被改形的指令
    return merged.filter((s) => s.type !== 'input_text' || s.text.length > 0);
}
