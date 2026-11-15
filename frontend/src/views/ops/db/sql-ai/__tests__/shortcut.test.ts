import { describe, expect, it } from 'vitest';
import { modKeyLabel } from '../shortcut';

describe('快捷键前缀', () => {
    it('mac 系显示 ⌘，其余显示 Ctrl', () => {
        expect(modKeyLabel('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)')).toBe('⌘');
        expect(modKeyLabel('Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)')).toBe('⌘');
        expect(modKeyLabel('Mozilla/5.0 (Windows NT 10.0; Win64; x64)')).toBe('Ctrl');
        expect(modKeyLabel('')).toBe('Ctrl');
    });
});