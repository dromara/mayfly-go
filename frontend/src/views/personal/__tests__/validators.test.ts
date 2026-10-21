import { describe, expect, it, vi } from 'vitest';

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key } },
}));

import { validatePasswordChange } from '../validators';

describe('personal/validators validatePasswordChange', () => {
    it('符合强度且两次一致才放行', () => {
        expect(validatePasswordChange({ password: 'Abc123!@#', confirmPassword: 'Abc123!@#' })).toBe('ok');
    });

    it('强度不足时先报强度问题（不要求两次一致）', () => {
        expect(validatePasswordChange({ password: '12345678', confirmPassword: '12345678' })).toBe('strength');
        expect(validatePasswordChange({ password: 'Abc123', confirmPassword: '' })).toBe('strength');
    });

    it('强度达标但两次不一致时判为 mismatch', () => {
        expect(validatePasswordChange({ password: 'Abc123!@#', confirmPassword: 'Abc123!@#x' })).toBe('mismatch');
        expect(validatePasswordChange({ password: 'Abc123!@#', confirmPassword: '' })).toBe('mismatch');
    });
});
