import { beforeEach, describe, expect, it, vi } from 'vitest';

// mock 掉 i18n 相关依赖，避免加载 element-plus 完整依赖链
vi.mock('@/hooks/useI18n', () => ({
    Msg: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}));
vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key } },
}));

import { copyToClipboard, fuzzyMatchField, isPrefixSubsequence, matchPrefixSubsequence, templateResolve } from '../string';

describe('templateResolve', () => {
    it('解析模板占位符', () => {
        expect(templateResolve('hahaha{name}_{id}', { name: 'hh', id: 1 })).toBe('hahahahh_1');
    });

    it('缺失的占位符替换为空字符串', () => {
        expect(templateResolve('{a}-{b}', { a: 'x' })).toBe('x-');
    });

    it('无占位符的模板原样返回', () => {
        expect(templateResolve('no placeholders', { a: 1 })).toBe('no placeholders');
    });

    it('支持 FormData 参数', () => {
        const fd = new FormData();
        fd.append('name', 'test');
        expect(templateResolve('/path/{name}', fd)).toBe('/path/test');
    });
});

describe('isPrefixSubsequence', () => {
    it('连续子序列匹配', () => {
        expect(isPrefixSubsequence('user', 'username')).toBe(true);
    });

    it('不连续但顺序一致的子序列匹配', () => {
        expect(isPrefixSubsequence('uname', 'username')).toBe(true);
    });

    it('顺序不一致不匹配', () => {
        expect(isPrefixSubsequence('nameuser', 'username')).toBe(false);
    });

    it('包含目标中不存在的字符不匹配', () => {
        expect(isPrefixSubsequence('uname2', 'username')).toBe(false);
    });

    it('空前缀匹配任何目标', () => {
        expect(isPrefixSubsequence('', 'anything')).toBe(true);
    });
});

/**
 * 命中下标与布尔判定必须同源（isPrefixSubsequence 就是它的非 null 包装）：
 * 高亮标的是「哪几个字让这行留下」，两套扫描一旦分叉就会标错色。
 */
describe('matchPrefixSubsequence', () => {
    it('返回命中的目标下标，未命中返回 null', () => {
        expect(matchPrefixSubsequence('uname', 'username')).toEqual([0, 4, 5, 6, 7]);
        expect(matchPrefixSubsequence('user', 'username')).toEqual([0, 1, 2, 3]);
        expect(matchPrefixSubsequence('nameuser', 'username')).toBeNull();
    });

    it('空前缀视为匹配任何目标，命中集为空', () => {
        expect(matchPrefixSubsequence('', 'anything')).toEqual([]);
    });

    it('默认大小写敏感，ignoreCase 下逐字符忽略大小写且不改动原文', () => {
        expect(matchPrefixSubsequence('US', 'username')).toBeNull();
        expect(matchPrefixSubsequence('US', 'username', true)).toEqual([0, 1]);
        expect(matchPrefixSubsequence('us', 'USername', true)).toEqual([0, 1]);
    });

    it('与 isPrefixSubsequence 同口径（含 ignoreCase 透传）', () => {
        expect(isPrefixSubsequence('uname', 'username')).toBe(matchPrefixSubsequence('uname', 'username') !== null);
        expect(isPrefixSubsequence('US', 'username')).toBe(false);
        expect(isPrefixSubsequence('US', 'username', true)).toBe(true);
    });
});

describe('fuzzyMatchField', () => {
    const fields = [
        { name: 'username', comment: '用户名' },
        { name: 'password', comment: '密码' },
        { name: 'email', comment: '邮箱' },
    ];

    it('按字段名模糊匹配', () => {
        const result = fuzzyMatchField('uname', fields, (f) => f.name);
        expect(result).toHaveLength(1);
        expect(result[0].name).toBe('username');
    });

    it('多提取函数匹配', () => {
        const result = fuzzyMatchField(
            '密码',
            fields,
            (f) => f.name,
            (f) => f.comment
        );
        expect(result).toHaveLength(1);
        expect(result[0].name).toBe('password');
    });

    it('大小写不敏感', () => {
        const result = fuzzyMatchField('EMAIL', fields, (f) => f.name);
        expect(result).toHaveLength(1);
        expect(result[0].name).toBe('email');
    });

    it('无匹配返回空数组', () => {
        expect(fuzzyMatchField('zzz', fields, (f) => f.name)).toHaveLength(0);
    });
});

describe('copyToClipboard', () => {
    beforeEach(() => {
        vi.restoreAllMocks();
    });

    it('非安全上下文使用 execCommand 降级方案', async () => {
        const execCommandMock = vi.fn().mockReturnValue(true);
        Object.defineProperty(document, 'execCommand', { value: execCommandMock, writable: true, configurable: true });

        await copyToClipboard('test text');
        expect(execCommandMock).toHaveBeenCalledWith('copy');
    });
});
