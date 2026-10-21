import { i18n } from '@/i18n';
import { describe, expect, it } from 'vitest';
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';

// 与运行时同源：合并语言目录下的分文件得到两份完整字典
const zhModules = import.meta.glob<{ default: object }>('@/i18n/zh-cn/*.ts', { eager: true });
const enModules = import.meta.glob<{ default: object }>('@/i18n/en/*.ts', { eager: true });

function merge(modules: Record<string, { default: object }>): object {
    return Object.values(modules).reduce<Record<string, unknown>>((acc, mod) => Object.assign(acc, mod.default), {});
}

/**
 * 代码里写出的 i18n key 必须在两种语言的字典里都存在。
 *
 * vue-i18n 在本项目配了 missingWarn:false，缺 key 不报错而是**把 key 原样显示到界面上**，
 * 英文界面还会直接吐出 `flow.procDesign` 这样的串；类型检查、lint、构建都不管这件事。
 * 实测就漏过一个例子：流程设计抽屉的标题取 `flow.procDesign`，而字典里只有 `flow.flowDesign`。
 */
const dictionaries = { zh: merge(zhModules), en: merge(enModules) };

function listSourceFiles(dir: string, into: string[] = []): string[] {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
        const full = join(dir, entry.name);
        if (entry.isDirectory()) {
            if (entry.name !== 'node_modules' && entry.name !== '__tests__') listSourceFiles(full, into);
        } else if (/\.(vue|ts)$/.test(entry.name)) {
            into.push(full);
        }
    }
    return into;
}

/** 取 `$t('a.b.c')` / `t('a.b.c')` 里写死的 key（只认字面量单引号，动态拼接不在本判据内） */
function referencedKeys(source: string): string[] {
    // `\b` 保证命中的是独立的 t/$t 调用，不会把 emit(...) 这类词尾字母当成翻译函数
    return [...source.matchAll(/\$?\bt\(\s*'([a-z]+\.[A-Za-z0-9_.]+)'/g)].map((m) => m[1]);
}

/**
 * 取写死的「无命名空间」key（`t('params')` 这类漏了前缀的写法）。
 *
 * 字典顶层全是命名空间（common / msg / agentops / flow…），所以这种 key 永远解析不到值；
 * 而 missingWarn:false 会让它原样显示到界面上（实测：发送弹窗的字段标签就是英文原词
 * `params`，kafka 消费组计数处是 `count`）。上面的 referencedKeys 只收「带点」的 key，
 * 整类漏前缀写法对它完全不可见，所以单独一条判据
 */
function dotlessKeys(source: string): string[] {
    return [...source.matchAll(/\$?\bt\(\s*'([A-Za-z0-9_]+)'/g)].map((m) => m[1]);
}

/**
 * 剔掉注释行。
 *
 * 不剔不行：注释里会合法地引用「旧的错误写法」作为反例（本仓就有几处写明「之前这里是
 * 裸键名」），拿整文件做包含判定会把注释当成界面缺陷，红得毫无意义。只做行级识别，
 * 不拆字符串里的 `//`（URL）——那类内容不会出现在行首
 */
function stripComments(source: string): string {
    const kept: string[] = [];
    let inHtmlComment = false;
    for (const line of source.split('\n')) {
        const text = line.trim();
        if (inHtmlComment) {
            if (text.includes('-->')) inHtmlComment = false;
            continue;
        }
        if (text.startsWith('<!--')) {
            if (!text.includes('-->')) inHtmlComment = true;
            continue;
        }
        if (text.startsWith('//') || text.startsWith('/*') || text.startsWith('*') || text.startsWith('*/')) continue;
        kept.push(line);
    }
    return kept.join('\n');
}

/**
 * 取 `props: { ... }` 里**没被 t() 包起来**的点号串字面量。
 *
 * auto-form 的渲染器只翻译 label/tooltip，`props` 是原样透传给底层组件的，
 * 所以把 i18n 键写进 props 不会报错、也不会被类型检查拦住，而是把裸键名显示到界面上。
 * 实测踩过：告警规则抽屉的 4 个时间/次数输入框占位符显示成 `alert.groupWaitPlaceholder`。
 */
function bareKeysInProps(source: string): { prop: string; key: string }[] {
    const out: { prop: string; key: string }[] = [];
    for (const obj of source.matchAll(/props:\s*\{([^{}]*)\}/g)) {
        // 先摘掉正确写法 t('a.b.c')，剩下的点号串才是漏翻译的
        const body = obj[1].replace(/\$?\bt\(\s*['"][^'"]+['"]\s*\)/g, 'T()');
        for (const m of body.matchAll(/([A-Za-z]+):\s*['"]([a-z]+\.[A-Za-z0-9_.]+)['"]/g)) {
            out.push({ prop: m[1], key: m[2] });
        }
    }
    return out;
}

function lookup(root: object, path: string): boolean {
    return (
        path.split('.').reduce<unknown>((node, key) => {
            if (!node || typeof node !== 'object') return undefined;
            return Reflect.get(node, key);
        }, root) !== undefined
    );
}

/** 收集字典里所有叶子文案的完整 key 路径（`a.b.c`） */
function flattenKeys(node: unknown, prefix: string, into: string[] = []): string[] {
    if (!node || typeof node !== 'object') return into;
    for (const [key, val] of Object.entries(node as Record<string, unknown>)) {
        const path = prefix ? `${prefix}.${key}` : key;
        if (typeof val === 'string') into.push(path);
        else flattenKeys(val, path, into);
    }
    return into;
}

describe('i18n key 与文案存在性', () => {
    const files = listSourceFiles(join(process.cwd(), 'src'));

    it('扫到了足量引用（防止判据本身失效导致空跑）', () => {
        const total = files.reduce((sum, file) => sum + referencedKeys(readFileSync(file, 'utf8')).length, 0);
        expect(total).toBeGreaterThan(200);
    });

    it('不得出现漏了命名空间的 t(\'xxx\') 调用（字典顶层全是命名空间，这类 key 永远取不到值）', () => {
        // 先用一段合成源码自测扫描式：不这样做，某天改了模板写法会让本判据安静地空跑
        expect(dotlessKeys("<span>{{ $t('count') }}</span>"), '扫描式已失配，本判据在空跑').toEqual(['count']);

        const bare: string[] = [];
        for (const file of files) {
            for (const key of new Set(dotlessKeys(stripComments(readFileSync(file, 'utf8'))))) {
                bare.push(`${file.replace(process.cwd() + '/', '')}: t('${key}')`);
            }
        }
        expect(bare, '以下调用会在界面上直接吐出英文原词:\n' + bare.join('\n')).toEqual([]);
    });

    it('写死的 key 在中文与英文字典里都能取到值', () => {
        const missing: string[] = [];
        for (const file of files) {
            const source = stripComments(readFileSync(file, 'utf8'));
            for (const key of new Set(referencedKeys(source))) {
                // 前缀不在字典顶层（动态拼出来的 key 不在本判据内）就跳过，其余要求两种语言都有值
                if (!(key.split('.')[0] in dictionaries.zh)) continue;
                for (const [locale, root] of Object.entries(dictionaries)) {
                    if (!lookup(root, key)) missing.push(`${file.replace(process.cwd() + '/', '')}: ${key}（${locale}）`);
                }
            }
        }
        expect(missing, '界面上会直接吐出裸 key:\n' + missing.join('\n')).toEqual([]);
    });

    it('props 里不得直接写 i18n 键（渲染器不翻译 props，界面会显示裸键名）', () => {
        const bare: string[] = [];
        let propsObjects = 0;
        for (const file of files) {
            const source = readFileSync(file, 'utf8');
            propsObjects += (source.match(/props:\s*\{/g) ?? []).length;
            for (const { prop, key } of bareKeysInProps(source)) {
                // 只认「确实是字典里的键」的点号串，避免把 'notifyConfig.groupWait' 这类
                // 字段路径误判成漏翻译
                if (lookup(dictionaries.zh, key)) bare.push(`${file.replace(process.cwd() + '/', '')}: props.${prop} = '${key}'`);
            }
        }
        // 判据本身要能扫到东西，否则改了正则也不会红
        expect(propsObjects, '没扫到任何 props 对象，判据已失效').toBeGreaterThan(30);
        expect(bare, '以下 props 值会在界面上显示成裸键名，应改为 t(...) 取值:\n' + bare.join('\n')).toEqual([]);
    });

    /**
     * 每条文案都必须能被 vue-i18n 的消息编译器编译。
     *
     * 判据存在的原因：本项目的占位符写的是单花括号 `{name}`，写成 `{{name}}` 时消息编译器抛
     * "Not allowed nest placeholder"、文案里出现裸 `@` 会被当作 linked message 报 "Invalid linked format"，
     * 而这类错误只在**渲染期真正翻译那条文案时**才暴露：实测「活动会话」面板里的 `更新于 {{time}}`
     * 让整块组件渲染抛错、页签内一片空白，而类型检查、lint、构建与上面的 key 存在性判据全部绿灯
     * （key 确实存在、值也确实有）。字面量花括号/`@` 需按 `{'{{.x}}'}`、`{'@'}` 转义。
     */
    it('两种语言的每条文案都能被 vue-i18n 编译（写法非法会导致渲染期抛错白屏）', () => {
        const broken: string[] = [];
        let checked = 0;

        // 直接用运行时那份 i18n 实例与它已装载的字典：判据与界面行为同源
        const originalLocale = i18n.global.locale.value;
        for (const locale of ['zh-cn', 'en']) {
            // 逐语言切换 locale 再按组件里的同款标量调用翻译：走的就是渲染期那条路径，
            // 传对象形式会被判为带参数调用而对纯文案报 Invalid arguments
            i18n.global.locale.value = locale;
            for (const path of flattenKeys(i18n.global.getLocaleMessage(locale), '')) {
                checked++;
                try {
                    // 占位符缺值只会被记为参数缺失告警（本项目已关这类告警），不影响消息编译这一步
                    i18n.global.t(path);
                } catch (e: unknown) {
                    broken.push(`${locale} · ${path}: ${e instanceof Error ? e.message.split('\n')[0] : String(e)}`);
                }
            }
        }
        i18n.global.locale.value = originalLocale;

        // 字典未加载会让本判据静默空跑，先要求它确实扫到了两个语言包的文案
        expect(checked, '两种语言的文案未被真正加载').toBeGreaterThan(500);
        expect(broken, '以下文案无法被消息编译器解析:\n' + broken.join('\n')).toEqual([]);
    });
});
