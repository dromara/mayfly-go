/**
 * 动态组件上的运行时指令守卫（全仓 SFC）
 *
 * 背景：写在 `<component :is>` 上的运行时指令（v-show 等）并不作用于这个标签本身，而要靠 fallthrough
 * 透传到「目标组件的根元素」。Vue 的 filterSingleRoot 只在模板顶层恰好有一个非注释根时才找得到那个根；
 * 一旦顶层还有第二个元素/组件根（渲染成 fragment），指令会被挂到 Fragment vnode 上，落不到任何 DOM，
 * 于是静默失效——不报错、界面照常渲染，只是显隐不再受控。
 *
 * 本项目连那条 dev 告警都看不到：main.ts 里 `app.config.warnHandler = () => null` 静音了全部 Vue 告警，
 * "Runtime directive used on component with non-element root node" 也一并被吞掉，没有任何可观测信号。
 *
 * 真实事故：RedisDataOp 模板顶层是「根 div + WorkTicketSubmit」两个元素根，ResourceOp 挂在 `<component>`
 * 上的 v-show 因此失效，该 tab 永远可见并盖住其它 tab（症状：点数据库表后仍显示 Redis 界面）。修法是把
 * v-show 移到包装层 div —— 与 components/tabs/Tabs.vue 的 .mf-tabs__pane 同一做法，容器不再对 tab 组件
 * 的根结构提任何要求（新增 tab 组件无需改动容器，符合开闭原则）。
 *
 * 本用例钉住三条不变式，新增文件零登记即被覆盖：
 *   1. 全仓 `<component>` 上不得出现依赖组件根元素的运行时指令（唯一豁免见 EXEMPT）；
 *   2. 豁免的前提自动复核：parent.vue 的 router-view 可能渲染到的组件必须都是单元素根；
 *   3. ResourceOp 内容区必须保持「v-show 在包装元素上、包装元素内是 `<component>`」的结构。
 * 另有常量自校验用例，防止编译器升级导致节点类型数值漂移、把断言悄悄变成永真。
 */
import { describe, it, expect } from 'vitest';
import { parse } from '@vue/compiler-sfc';
import * as fs from 'node:fs';
import * as path from 'node:path';

const SRC_ROOT = path.resolve(__dirname, '..');

/** `components/ui` 是 shadcn-vue CLI 生成的第三方组件（升级即被覆盖、不做手工改造），与 componentContracts 一致地排除 */
const EXCLUDE_DIRS = ['components/ui'];

/**
 * 节点类型数值。@vue/compiler-sfc 不导出 NodeTypes，compiler-core 在 pnpm 严格模式下也不可直解析，
 * 故取其稳定数值；下方「常量自校验」用内联夹具钉住，数值一旦漂移会立刻失败而非静默永真。
 */
const NODE_ELEMENT = 1;
const NODE_DIRECTIVE = 7;

/**
 * 挂在 `<component>` 上仍然安全的指令：要么在编译期展开（v-if / v-for / v-once…），要么只产出 props
 * 与监听（v-bind / v-on / v-model / v-slot），都不依赖「组件根元素」，多根组件上照常工作。
 * 不在此列的（v-show、v-html、自定义指令…）必须落到根 DOM 上，遇多根组件即静默失效。
 */
const SAFE_DIRECTIVES = new Set(['if', 'else', 'else-if', 'for', 'once', 'memo', 'pre', 'slot', 'bind', 'on', 'model']);

type AstProp = { type: number; name?: string };
type AstNode = { type: number; tag?: string; props?: AstProp[]; children?: AstNode[] };

/** 递归收集匹配文件（跳过 EXCLUDE_DIRS） */
function listFiles(dir: string, test: RegExp): string[] {
    return fs
        .readdirSync(dir, { withFileTypes: true })
        .flatMap((entry) => {
            const full = path.join(dir, entry.name);
            if (entry.isDirectory()) {
                return EXCLUDE_DIRS.includes(path.relative(SRC_ROOT, full).split(path.sep).join('/')) ? [] : listFiles(full, test);
            }
            return test.test(entry.name) ? [full] : [];
        })
        .sort();
}

/** 解析 SFC 源码取模板根 AST；无 `<template>` 或解析失败返回 undefined */
function templateAst(source: string, filename: string): AstNode | undefined {
    try {
        const { descriptor } = parse(source, { filename });
        return descriptor.template?.ast as unknown as AstNode | undefined;
    } catch {
        return undefined;
    }
}

const astOf = (file: string): AstNode | undefined => templateAst(fs.readFileSync(file, 'utf8'), file);

/** 深度遍历全部元素节点 */
function elements(node: AstNode | undefined, out: AstNode[] = []): AstNode[] {
    if (!node) return out;
    if (node.type === NODE_ELEMENT) out.push(node);
    for (const child of node.children ?? []) elements(child, out);
    return out;
}

/** 模板顶层的元素/组件根标签名（注释与空白文本不算根 —— Vue 的 fallthrough 同样会跳过注释） */
function elementRoots(node: AstNode | undefined): string[] {
    return (node?.children ?? []).filter((n) => n.type === NODE_ELEMENT).map((n) => n.tag ?? '?');
}

/** 相对 src 的 posix 风格路径，用作断言里的稳定标识 */
const rel = (file: string) => path.relative(SRC_ROOT, file).split(path.sep).join('/');

describe('动态组件上的运行时指令', () => {
    const sfcs = listFiles(SRC_ROOT, /\.vue$/);

    /**
     * 豁免登记：文件 → 为什么这里不能改成包装层。
     * 按文件精确登记、不做整目录豁免，且豁免的前提由下一条不变式随源码自动复核，不是人工承诺。
     */
    const EXEMPT: Record<string, string> = {
        'layout/routerView/parent.vue':
            '外层是 transition + keep-alive：keep-alive 只接受单个组件子节点，transition 依赖子节点切换触发过场，' +
            '插入包装层会同时破坏路由缓存与页面切换动画。其 v-show 的作用对象只有路由入口组件，均为单元素根（见下方不变式）。',
    };

    it('受检 SFC 数量非零（防遍历规则失配导致断言空转）', () => {
        expect(sfcs.length).toBeGreaterThan(100);
    });

    it('`<component>` 上不得出现依赖组件根元素的运行时指令', () => {
        const offenders: string[] = [];
        for (const file of sfcs) {
            if (EXEMPT[rel(file)]) continue;
            for (const el of elements(astOf(file))) {
                if (el.tag !== 'component') continue;
                const unsafe = (el.props ?? []).filter((p) => p.type === NODE_DIRECTIVE && p.name && !SAFE_DIRECTIVES.has(p.name)).map((p) => `v-${p.name}`);
                if (unsafe.length) offenders.push(`${rel(file)} :: ${unsafe.join(', ')}`);
            }
        }
        expect(
            offenders,
            '这类指令要靠 fallthrough 落到「组件根元素」，目标组件顶层有 ≥2 个元素根时会静默失效' +
                '（本项目 warnHandler 被置空，连告警都没有）。\n' +
                `改法：把指令移到包住 <component> 的普通元素上，别去限制目标组件的根结构。\n${offenders.join('\n')}`
        ).toEqual([]);
    });

    it('豁免登记里的文件确实存在（防改名/删除后登记空转）', () => {
        const missing = Object.keys(EXEMPT).filter((f) => !fs.existsSync(path.join(SRC_ROOT, f)));
        expect(missing, `以下豁免文件已不存在，请连同登记与理由一起删除：${missing.join(', ')}`).toEqual([]);
    });
});

describe('parent.vue 豁免的前提：其可渲染到的组件必须单元素根', () => {
    /** 取出 .ts 里引用的 .vue（`() => import('...')` 与 `from '...'` 两种形态），解析为绝对路径 */
    function referencedVue(tsFile: string): string[] {
        if (!fs.existsSync(tsFile)) return [];
        const src = fs.readFileSync(tsFile, 'utf8');
        const specs = [...src.matchAll(/import\(\s*['"]([^'"]+\.vue)['"]\s*\)/g), ...src.matchAll(/from\s+['"]([^'"]+\.vue)['"]/g)].map((m) => m[1]);
        return [...new Set(specs)]
            .map((s) => (s.startsWith('@/') ? path.join(SRC_ROOT, s.slice(2)) : path.resolve(path.dirname(tsFile), s)))
            .filter((p) => fs.existsSync(p));
    }

    // views/**/route.ts 是后端菜单动态路由的组件表，其组件被 addRoute 到 layout 之下，即由 parent.vue 的 router-view 渲染
    const routeEntries = listFiles(SRC_ROOT, /route\.(ts|js)$/).flatMap(referencedVue);
    // menuRouteResolver 为外链 / 内嵌 iframe 路由直接指派 link.vue、iframes.vue —— 正是 v-show 需要隐藏的那批组件
    const resolverEntries = referencedVue(path.join(SRC_ROOT, 'router/menuRouteResolver.ts'));
    const entries = [...new Set([...routeEntries, ...resolverEntries])];

    it('确实解析到了路由入口组件（防引用形态变化导致断言空转）', () => {
        expect(entries.length).toBeGreaterThan(10);
        // iframe / 外链路由的组件必须在受检集合内，否则本条不变式就漏掉了 v-show 真正要隐藏的对象
        expect(entries.map(rel).some((f) => f.endsWith('routerView/iframes.vue'))).toBe(true);
    });

    it('全部为单元素根', () => {
        const multi = entries.filter((f) => elementRoots(astOf(f)).length > 1).map((f) => `${rel(f)} :: ${elementRoots(astOf(f)).join(' + ')}`);
        expect(
            multi,
            '这些路由入口组件是 parent.vue 里 `<component v-show="!isIframePage">` 的作用对象：一旦顶层出现第二个元素根，' +
                'v-show 就会静默失效，页面会与其上层的 iframe 面板同时可见。\n' +
                '要么把它们收成单元素根，要么改造 parent.vue 的显隐方式（注意 transition + keep-alive 的结构约束）。\n' +
                multi.join('\n')
        ).toEqual([]);
    });
});

describe('ResourceOp 内容区的修复结构', () => {
    const RESOURCE_OP = path.join(SRC_ROOT, 'views/ops/resource/ResourceOp.vue');

    it('v-show 挂在包装元素上，且包装元素内是 <component>', () => {
        const wrapper = elements(astOf(RESOURCE_OP)).find(
            (el) =>
                el.tag !== 'component' &&
                (el.props ?? []).some((p) => p.type === NODE_DIRECTIVE && p.name === 'show') &&
                (el.children ?? []).some((c) => c.type === NODE_ELEMENT && c.tag === 'component')
        );
        expect(
            wrapper,
            '资源 tab 内容区必须靠「包装元素 + v-show」切换显隐：tab 组件常驻挂载、只显隐不销毁，' +
                '而 v-show 若挂回 <component>，遇到多根 tab 组件（如 RedisDataOp）会静默失效并盖住其它 tab。'
        ).toBeTruthy();
    });
});

describe('常量自校验（防编译器升级后断言永真）', () => {
    /** 事故现场的等价夹具：顶层两个元素根 + 一个注释 */
    const MULTI_ROOT = '<template>\n    <div class="op">x</div>\n    <!-- 注释不算根 -->\n    <SomeDialog />\n</template>\n';
    const SINGLE_ROOT = '<template>\n    <div class="op">x</div>\n</template>\n';

    it('多元素根被识别为多根，注释不计入根', () => {
        expect(elementRoots(templateAst(MULTI_ROOT, 'MultiRoot.vue'))).toEqual(['div', 'SomeDialog']);
        expect(elementRoots(templateAst(SINGLE_ROOT, 'SingleRoot.vue'))).toEqual(['div']);
    });

    it('指令节点能被读到名字（v-show 判为不安全、v-if 判为安全）', () => {
        const dirs = (tpl: string) =>
            elements(templateAst(tpl, 'D.vue'))
                .filter((el) => el.tag === 'component')
                .flatMap((el) => (el.props ?? []).filter((p) => p.type === NODE_DIRECTIVE).map((p) => p.name));
        const names = dirs('<template>\n    <component :is="c" v-show="ok" v-if="live" />\n</template>\n');
        expect(names).toEqual(expect.arrayContaining(['show', 'if']));
        expect(names.filter((n) => n && !SAFE_DIRECTIVES.has(n))).toEqual(['show']);
    });
});
