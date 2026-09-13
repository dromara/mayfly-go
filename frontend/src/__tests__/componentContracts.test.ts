/**
 * 组件事件契约守卫（全仓 SFC）
 *
 * 背景：`defineEmits` 声明的事件名与实际 `emit()` 调用之间没有任何编译期约束，两种失真都能静默上线：
 *   1. 声明了却从不 emit —— 纯噪音，且会被复制粘贴扩散（曾有 6 个资源模块各抄了一份 `init`）；
 *   2. emit 了却没声明 —— 监听方靠 $attrs 透传侥幸生效，但 payload 完全脱离类型检查。
 * 更糟的是 main.ts 里 `app.config.warnHandler = () => null` 静音了所有 Vue 开发告警，
 * 上述失真连控制台都看不到（Element Plus 走自己的 console.warn，所以只有它的告警能看见）。
 *
 * 「修改单元格后提交/取消按钮消失」那次回归属于第 2 类的变体：子组件漏传 payload，父级按 Map.size 判断。
 * 本用例把不变式钉住，按目录结构推导受检文件，新增组件零登记即被覆盖。
 */
import { describe, it, expect } from 'vitest';
import * as fs from 'node:fs';
import * as path from 'node:path';

const SRC_ROOT = path.resolve(__dirname, '..');

/**
 * 不纳入检查的目录。
 *
 * `components/ui` 是 shadcn-vue 注册表生成的第三方组件（99 个 SFC，升级会被 CLI 覆盖，不做手工改造）：
 * 它们的 `update:modelValue` 由 `useVModel(props, 'modelValue', emits)` 这类工具函数代发，
 * 源码里没有 `emit()` 字面量，本守卫的判据对这套写法不适用。其余目录一律受检。
 */
const EXCLUDE_DIRS = ['components/ui'];

/**
 * 这些目录已完成类型式收口，禁止再出现数组式 `defineEmits`。
 * 其余目录存量较大（数组式不检查 payload），由下方的全局棘轮钉住不增。
 */
const TYPED_ONLY_DIRS = ['views/ops/db', 'components/monaco'];

/**
 * 声明了、但由宿主 composable 以 `vnode.props.onXxx` 形式直接调用父级处理器消费的事件。
 *
 * `useAutoFormHost` 的 confirm 走的就是这条路（见其 123 行注释）：源码里没有 `emit('confirm')`，
 * 但父组件的 `@confirm` 确实会被调用，故不能按死声明处理。按「文件 + 事件名」精确登记，
 * 不做整文件豁免，避免豁免面随时间无声扩大。
 */
const CONSUMED_BY_HOST_HANDLER: Record<string, string[]> = {
    'components/auto-form/AutoFormDialog.vue': ['confirm'],
    'components/auto-form/AutoFormDrawer.vue': ['confirm'],
};

/** 递归收集 SFC（`__tests__` 除外） */
function listSfc(dir: string): string[] {
    return fs
        .readdirSync(dir, { withFileTypes: true })
        .flatMap((entry) => {
            const full = path.join(dir, entry.name);
            if (entry.isDirectory()) return entry.name === '__tests__' ? [] : listSfc(full);
            return entry.name.endsWith('.vue') ? [full] : [];
        })
        .sort();
}

/**
 * 剥掉注释。
 *
 * 必须最先做：DbTableData.vue 的文档注释里引用了 `defineEmits([...])` 这个写法本身，
 * 不剥注释就会被当成真实声明，产出一批假阳性。（`(?<!:)` 是为了不错过 `https://` 里的双斜杠。）
 */
function stripComments(src: string): string {
    return src.replace(/<!--[\s\S]*?-->/g, '').replace(/\/\*[\s\S]*?\*\//g, '').replace(/(?<!:)\/\/[^\n]*/g, '');
}

type EmitBlock = { kind: 'array' | 'typed'; text: string };

/**
 * 取出 `defineEmits` 的声明块，并标注写法。
 *
 * 三种写法仓库里都在用，都要认：
 *   数组式        defineEmits(['cancel', 'val-change'])          -> kind: 'array'（payload 不受类型检查）
 *   成员类型式    defineEmits<{ changeUpdatedField: [v: boolean] }>()
 *   签名类型式    defineEmits<{ (e: 'update:ddl', value: string): void }>()
 *
 * 定界符靠配平扫描而非正则：类型块里会出现 `Map<string, () => void>` 等内层尖括号，
 * 用非贪婪 `<[\s\S]*?>` 会在第一个 `>` 处截断，漏认事件名的表现是「明明声明了却报未声明」。
 */
function emitsBlocks(code: string): EmitBlock[] {
    const blocks: EmitBlock[] = [];
    for (const m of code.matchAll(/defineEmits\s*(<|\()/g)) {
        const open = m[1];
        const close = open === '<' ? '>' : ')';
        let i = (m.index ?? 0) + m[0].length;
        let braceDepth = 0; // 花括号配平后遇到的 close 才是收尾；类型里的 `>` 都发生在 depth>0 处
        const start = i;
        while (i < code.length) {
            const ch = code[i];
            if (ch === '{') braceDepth++;
            else if (ch === '}') braceDepth--;
            else if (ch === close && braceDepth <= 0) break;
            i++;
        }
        blocks.push({ kind: open === '(' ? 'array' : 'typed', text: code.slice(start, i) });
    }
    return blocks;
}

/**
 * 从一个声明块里取出事件名。
 *
 * - 成员式：`事件名: [payload 元组]`，且名字紧跟在 `{` 或 `;` 之后。
 *   要求冒号后直接是 `[` 是关键——只按「标识符后跟冒号」匹配，会把 payload 里的内联对象类型
 *   （`[sort: { key: string; order: string }]` 的 `key`/`order`）误认成事件名。
 *   已知代价：`事件名: (...args) => void` 这种函数属性写法不被识别，会报「emit 却未声明」（红，不是假绿）。
 * - 数组式与签名式：取字符串字面量（后跟 `,` `)` `]`，或紧跟在 `(` 之后）。
 */
function declaredNames(block: EmitBlock): string[] {
    const names = new Set<string>();
    const member = /[{;]\s*(?:'([^']+)'|"([^"]+)"|([\w$][\w$-]*))\s*:\s*\[/g;
    for (const m of block.text.matchAll(member)) names.add(m[1] ?? m[2] ?? m[3]);
    const literal = /['"]([^'"]+)['"]\s*[,\])]|\(\s*['"]([^'"]+)['"]/g;
    for (const m of block.text.matchAll(literal)) names.add(m[1] ?? m[2]);
    return [...names];
}

/**
 * 组件自身 emit 的事件名。
 *
 * 前置 `(?<![.\w])` 用来排除其他对象的 emit 方法：资源树用的是类型化事件总线
 * `treeEvents.emit('node:locate', ...)`，那套事件在 `resource/tree/events.ts` 里另有声明映射，
 * 与组件的 emits 契约无关，混进来会产成假阳性。
 * 模板里的 `$emit('x', ...)` 属于真实 emit（事件转发），保留匹配。
 */
function emittedOf(code: string): string[] {
    return [...code.matchAll(/(?<![.\w])(?:emit|emits)\(\s*['"]([^'"]+)['"]/g)].map((m) => m[1]);
}

/**
 * `defineModel('visible')` 等 model 自动产生的事件。
 * 这些 `update:*` 由 Vue 内部 emit，源码里没有对应 emit 调用，故不算死声明。
 *
 * 匿名形式有两种写法，都要认：`defineModel<T>()` 与带选项对象的 `defineModel<T>({ default: false })`，
 * 后者若漏判会把一批 `update:modelValue` 误报成死声明。
 */
function modelEventsOf(code: string): string[] {
    const names = new Set<string>();
    const generic = String.raw`(?:<[^<>]*(?:<[^<>]*>[^<>]*)*>)?`;
    for (const m of code.matchAll(new RegExp(String.raw`defineModel${generic}\(\s*['"]([^'"]+)['"]`, 'g'))) names.add(`update:${m[1]}`);
    if (new RegExp(String.raw`defineModel${generic}\(\s*(?:\)|\{)`).test(code)) names.add('update:modelValue');
    return [...names];
}

/** 解析一个 SFC 的事件契约 */
function analyze(src: string, relPath: string) {
    const code = stripComments(src);
    const blocks = emitsBlocks(code);
    const declared = blocks.flatMap(declaredNames);
    const emitted = emittedOf(code);
    const auto = modelEventsOf(code);
    const consumed = CONSUMED_BY_HOST_HANDLER[relPath] ?? [];
    return {
        /** 存在数组式声明（payload 脱离类型检查），用于判定目录是否已收口 */
        usesArrayStyle: blocks.some((b) => b.kind === 'array'),
        /** 声明了但源码里从未 emit（defineModel 自动事件、宿主处理器消费的除外） */
        dead: declared.filter((e) => !emitted.includes(e) && !auto.includes(e) && !consumed.includes(e)),
        /** emit 了却没声明 */
        undeclared: emitted.filter((e) => !declared.includes(e) && !auto.includes(e)),
    };
}

/** 受检 SFC：src 下全部组件（含 layout、App.vue 等 views/components 之外的），新增目录零登记即被覆盖 */
const sfcs = listSfc(SRC_ROOT)
    .map((file) => path.relative(SRC_ROOT, file))
    .filter((rel) => !EXCLUDE_DIRS.some((dir) => rel.startsWith(dir + path.sep)))
    .map((rel) => ({ rel, ...analyze(fs.readFileSync(path.join(SRC_ROOT, rel), 'utf-8'), rel) }));

describe('组件事件契约', () => {
    it('解析器认全三种声明写法，且不被 payload 类型带偏（判据错了守卫就是假绿灯）', () => {
        const cases: [string, string[]][] = [
            [`const emit = defineEmits(['cancel', 'val-change']);`, ['cancel', 'val-change']],
            [`const emit = defineEmits<{ a: [x: number]; 'b-c': [] }>();`, ['a', 'b-c']],
            [`const emit = defineEmits<{ (e: 'update:ddl', value: string): void }>();`, ['update:ddl']],
            // 成员类型里带泛型与箭头函数时，收尾 `>` 不能被提前截断
            [`const emit = defineEmits<{ m: [map: Map<string, () => void>] }>();`, ['m']],
            // 真实形态一：多行 + 每个成员上方带文档注释（DbTableData.vue）
            [`const e = defineEmits<{\n    /** 说明 */\n    changeUpdatedField: [hasUpdatedFields: boolean];\n    /** 说明 */\n    dataDelete: [rows: Record<string, unknown>[]];\n}>();`, ['changeUpdatedField', 'dataDelete']],
            // 真实形态二：payload 是内联对象类型，其字段名不能被认成事件名（DbTableDataHeader.vue）
            [`const e = defineEmits<{ sortChange: [sort: { key: string; order: string }] }>();`, ['sortChange']],
        ];
        const NOISE = ['x', 'hasUpdatedFields', 'rows', 'value', 'map', 'boolean', 'key', 'order', 'string', 'e', 'void'];
        for (const [src, expected] of cases) {
            const got = declaredNames(emitsBlocks(stripComments(src))[0] ?? { kind: 'typed', text: '' });
            expect(got, src).toEqual(expect.arrayContaining(expected));
            for (const noise of NOISE) expect(got, `${src} 把 ${noise} 误认成事件名`).not.toContain(noise);
        }
        // 数组式与类型式要能区分，否则写法禁令数不清
        expect(emitsBlocks(`defineEmits(['a']);`).map((b) => b.kind)).toEqual(['array']);
        expect(emitsBlocks(`defineEmits<{ a: [] }>();`).map((b) => b.kind)).toEqual(['typed']);
        // 事件总线的 emit 不属于组件契约；模板里的 $emit 属于
        expect(emittedOf(`treeEvents.emit('node:locate', { key });`)).toEqual([]);
        expect(emittedOf(`<child @click="$emit('refresh', 1)" />`)).toEqual(['refresh']);
        // defineModel 的三种形态都要推出自动事件，漏一个就把真声明误判成死声明
        expect(modelEventsOf(`const v = defineModel<boolean>('visible');`)).toEqual(['update:visible']);
        expect(modelEventsOf(`const v = defineModel<string>();`)).toEqual(['update:modelValue']);
        expect(modelEventsOf(`const v = defineModel<boolean>({ default: false });`)).toEqual(['update:modelValue']);
        // 注释里提到的写法必须被忽略
        expect(analyze(`// defineEmits(['ghost'])\nconst e = defineEmits<{ real: [v: boolean] }>();\nemits('real', true);`, 'x.vue').dead).toEqual([]);
    });

    it('受检范围内确有可检查的 SFC（防止路径写错导致用例空跑）', () => {
        expect(sfcs.length).toBeGreaterThan(300);
    });

    it('已收口目录不再使用数组式 defineEmits', () => {
        const offenders = sfcs
            .filter((r) => r.usesArrayStyle && TYPED_ONLY_DIRS.some((dir) => r.rel.startsWith(dir + path.sep)))
            .map((r) => r.rel);
        expect(offenders, `数组式 emits 的 payload 不受类型检查，请改成 defineEmits<{ 事件名: [参数类型] }>()\n${offenders.join('\n')}`).toEqual([]);
    });

    it('不存在「声明却从不 emit」的事件', () => {
        const offenders = sfcs.filter((r) => r.dead.length).map((r) => `${r.rel} :: ${r.dead.join(', ')}`);
        expect(offenders, `死声明会被后续组件复制粘贴扩散，请删除声明或补上 emit\n${offenders.join('\n')}`).toEqual([]);
    });

    it('不存在「emit 却未声明」的事件', () => {
        const offenders = sfcs.filter((r) => r.undeclared.length).map((r) => `${r.rel} :: ${r.undeclared.join(', ')}`);
        expect(offenders, `未声明的 emit 会落进 $attrs 透传到根元素，且 payload 不受类型检查\n${offenders.join('\n')}`).toEqual([]);
    });

    // ==================== 待清的历史债（棘轮：只准减不准增） ====================

    /**
     * 数组式 `defineEmits` 不检查 payload，`emits('x')` 漏传实参在编译期无感（本次回归的根因）。
     * 已收口目录由上面的禁令守住，其余目录按文件数钉住存量：改完一个就下调该数值（或把整个目录加入
     * TYPED_ONLY_DIRS），直到全部归零后删除本棘轮与 TYPED_ONLY_DIRS。
     *
     * 数值为实测存量（受检 342 个 SFC 中仍用数组式的文件数）。
     */
    const ARRAY_STYLE_FILE_BASELINE = 58;

    it('数组式 defineEmits 的文件数不超过存量基线（棘轮）', () => {
        const list = sfcs.filter((r) => r.usesArrayStyle).map((r) => r.rel);
        expect(
            list.length,
            `数组式 emits 的 payload 不受类型检查。剩 ${list.length} 个文件:\n${list.join('\n')}\n新增的一律用 defineEmits<{ 事件名: [参数类型] }>()；顺手清理则下调基线`,
        ).toBeLessThanOrEqual(ARRAY_STYLE_FILE_BASELINE);
    });
});
