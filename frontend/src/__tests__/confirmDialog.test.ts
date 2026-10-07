import { readdirSync, readFileSync } from 'node:fs';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ElMessageBox } from 'element-plus';
import { useI18nConfirm, useI18nDeleteConfirm } from '@/hooks/useI18n';

type MessageBoxData = Awaited<ReturnType<typeof ElMessageBox.confirm>>;

/**
 * 确认框原语的行为契约 + 全仓接线守卫。
 *
 * 背景：原语过去用 reject 表达「用户取消」，于是每个调用点都必须自己记得兜异常；
 * 全仓几十处里大多数是裸 await，点一次取消就留一条未捕获异常。改成返回布尔值之后，
 * 风险翻转了 —— 调用点忘判返回值，就从「取消后什么都不做」变成「取消后照样执行破坏性操作」。
 * 因此除了行为用例，还需要一条仓库级接线断言把「裸调用」判红。
 */

vi.mock('element-plus', () => ({
    ElMessageBox: { confirm: vi.fn(), alert: vi.fn(), prompt: vi.fn() },
}));

/** 递归收集源码文件（不起 shell，也不依赖 find 的行为） */
function sourceFiles(dir = 'src'): string[] {
    return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
        const path = `${dir}/${entry.name}`;
        if (entry.isDirectory()) {
            return path.includes('__tests__') ? [] : sourceFiles(path);
        }
        return path.endsWith('.vue') || path.endsWith('.ts') ? [path] : [];
    });
}

describe('确认框用返回值表达用户选择', () => {
    afterEach(() => vi.mocked(ElMessageBox.confirm).mockReset());

    it('点确认返回 true', async () => {
        vi.mocked(ElMessageBox.confirm).mockResolvedValue('confirm' as MessageBoxData);
        expect(await useI18nConfirm('common.confirm')).toBe(true);
    });

    it('点取消、关掉弹窗都返回 false，且不抛异常', async () => {
        // element-plus 对按钮取消给 'cancel'，对右上角/ESC 给 'close'，也可能给 Error：
        // 对调用点而言三者是同一件事「不继续」，原语必须把它们统一成 false
        for (const reason of ['cancel', 'close', new Error('cancel')]) {
            vi.mocked(ElMessageBox.confirm).mockRejectedValue(reason);
            await expect(useI18nConfirm('common.confirm')).resolves.toBe(false);
        }
    });

    it('删除确认同样不抛异常', async () => {
        vi.mocked(ElMessageBox.confirm).mockRejectedValue('cancel');
        await expect(useI18nDeleteConfirm('订单表')).resolves.toBe(false);
    });
});

describe('接线：不允许出现不判返回值的确认调用', () => {
    const files = sourceFiles();
    const bare = files
        .map((path) => [path, readFileSync(path, 'utf8')] as const)
        .flatMap(([path, text]) =>
            text
                .split('\n')
                .map((line, index) => ({ line: line.trim(), path, index: index + 1 }))
                .filter(
                    (item) => /^\}?\s*await useI18n(Confirm|DeleteConfirm)\(/.test(item.line) || /^(await useI18n(Confirm|DeleteConfirm)\()/.test(item.line)
                )
        );

    it('确认调用一律包在 if 判断里（裸调用等于取消后照样执行）', () => {
        expect(bare.map((item) => `${item.path}:${item.index} ${item.line.slice(0, 60)}`)).toEqual([]);
    });

    it('确认框只有两个出口：通用原语与需要三态的策略确认', () => {
        // 别处直接摇 ElMessageBox 就会绕开统一的标题/按钮/取消语义（milvus 曾这样漏掉了取消处理）
        const sanctioned = new Set(['src/hooks/useI18n.ts', 'src/views/flow/warnAck.ts']);
        const offenders = files.filter((path) => !sanctioned.has(path) && readFileSync(path, 'utf8').includes('ElMessageBox.confirm('));
        expect(offenders).toEqual([]);
    });

    it('扫描真的覆盖了业务视图（防止 find 规则改坏导致用例空跑）', () => {
        expect(files.length).toBeGreaterThan(100);
        expect(files.filter((path) => path.includes('views/'))).not.toHaveLength(0);
    });
});
