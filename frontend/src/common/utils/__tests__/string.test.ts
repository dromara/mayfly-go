import { beforeEach, describe, expect, it, vi } from 'vitest';

// mock 掉 i18n 相关依赖，避免加载 element-plus 完整依赖链
vi.mock('@/hooks/useI18n', () => ({
    Msg: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}));
vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key } },
}));

import { copyToClipboard, fuzzyMatchField, isPrefixSubsequence, templateResolve } from '../string';

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

describe('fuzzyMatchField', () => {
    const fields = [{ name: 'username', comment: '用户名' }, { name: 'password', comment: '密码' }, { name: 'email', comment: '邮箱' }];

    it('按字段名模糊匹配', () => {
        const result = fuzzyMatchField('uname', fields, (f) => f.name);
        expect(result).toHaveLength(1);
        expect(result[0].name).toBe('username');
    });

    it('多提取函数匹配', () => {
        const result = fuzzyMatchField('密码', fields, (f) => f.name, (f) => f.comment);
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
