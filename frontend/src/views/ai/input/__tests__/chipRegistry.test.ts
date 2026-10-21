/**
 * 芯片注册表的两个契约（都属于「漏一处不报错、只表现为图标不对」的形态）
 *
 * 1. 图标名必须落在 chipStyles.scss 支持的 mask 集合内：新增资源类型或改了 icon() 返回值
 *    而忘了补 CSS，界面只是芯片里少个图标，没有任何报错。
 * 2. resource 图标按子类型细分，且判定只在注册处一份：编辑器与消息气泡必须取到同一结果。
 */
import { describe, expect, it } from 'vitest';
// 副作用导入：注册 skill/resource 两类芯片（不导入则注册表为空，本文件断言会全绿假通过）
import '../chipTypes';
import { SUPPORTED_CHIP_ICONS, chipIcon, getChipType } from '../chipRegistry';

describe('芯片图标注册表', () => {
    it('已注册类型的图标都在 CSS 支持的 mask 集合内', () => {
        for (const type of ['skill', 'resource']) {
            expect(getChipType(type), `芯片类型 ${type} 必须已注册`).toBeDefined();
            // 覆盖 resource 的两条分支（db / 非 db），skill 忽略入参恒定
            for (const data of [{}, { resourceType: 'db' }, { resourceType: 'machine' }]) {
                const icon = chipIcon(type, { label: 'x', data });
                expect(SUPPORTED_CHIP_ICONS as readonly string[]).toContain(icon);
            }
        }
    });

    it('resource 图标按子类型细分，未知类型回退 zap', () => {
        expect(chipIcon('resource', { label: 'x', data: { resourceType: 'db' } })).toBe('database');
        expect(chipIcon('resource', { label: 'x', data: { resourceType: 'machine' } })).toBe('server');
        expect(chipIcon('not-registered', { label: 'x', data: {} })).toBe('zap');
    });
});
