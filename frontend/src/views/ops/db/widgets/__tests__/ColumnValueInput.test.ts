import { beforeEach, describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';

// Monaco 弹窗打桩：这里要钉的是「交给编辑器的内容长什么样」，不是 monaco 本身
const editorBox = vi.fn();
vi.mock('@/components/monaco/MonacoEditorBox', () => ({
    default: (options: { content: string; language?: string }) => editorBox(options),
}));

import { DataType } from '../../dialect/index';
import SvgIcon from '@/components/svg-icon/index.vue';
import BinaryValuePreview from '../BinaryValuePreview.vue';
import ColumnValueInput from '../ColumnValueInput.vue';

/**
 * 列值录入控件的分支守卫
 *
 * 两条判据都只有界面能观察到：二进制列必须出现预览入口（否则用户只能看到一串 hex），
 * JSON 值必须先美化再交给编辑器（否则一行几 KB 的 JSON 只能在单行里左右拖）。
 */

const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });

function mountInput(props: Record<string, unknown>) {
    return mount(ColumnValueInput, {
        props: { dataType: DataType.String, ...props },
        global: { plugins: [i18n, ElementPlus] },
    });
}

/** 按图标名取入口：SvgIcon 是 script setup 组件，没有组件名可按 name 查，只能按实例筛 props */
function iconOf(wrapper: ReturnType<typeof mountInput>, iconName: string) {
    return wrapper.findAllComponents(SvgIcon).find((icon) => icon.props('name') === iconName);
}

beforeEach(() => {
    editorBox.mockReset();
});

describe('ColumnValueInput 二进制列', () => {
    it('binary 为真时出现预览入口，点击即打开只读预览弹窗', async () => {
        const wrapper = mountInput({ binary: true, columnName: 'avatar', modelValue: '616263' });
        const previewIcon = iconOf(wrapper, 'View');
        expect(previewIcon, '二进制列必须有预览入口').toBeTruthy();

        await previewIcon!.trigger('mousedown');
        const dialog = wrapper.findComponent(BinaryValuePreview);
        expect(dialog.exists()).toBe(true);
        expect(dialog.props('visible')).toBe(true);
        expect(dialog.props('hex')).toBe('616263');
    });

    it('非二进制列不给预览入口（把文本列换成只读预览是误伤）', () => {
        const wrapper = mountInput({ binary: false, modelValue: 'plain text' });
        expect(wrapper.findComponent(BinaryValuePreview).props('visible')).toBe(false);
        expect(iconOf(wrapper, 'View')).toBeFalsy();
    });
});

describe('ColumnValueInput 全屏编辑器', () => {
    // 超 50 字符才出全屏入口，用例里的 JSON 均按长值构造
    it('JSON 值打开前先美化，语言仍按 json 交给编辑器', async () => {
        const long = `{"a":1,"list":[${Array.from({ length: 30 }, (_, i) => i).join(',')}],"b":"${'x'.repeat(30)}"}`;
        const wrapper = mountInput({ modelValue: long, columnName: 'ext' });
        const icon = iconOf(wrapper, 'FullScreen');
        expect(icon, '长文本必须给出全屏编辑入口').toBeTruthy();

        await icon!.trigger('mousedown');
        expect(editorBox).toHaveBeenCalledTimes(1);
        expect(editorBox.mock.calls[0][0].language).toBe('json');
        // 美化后的形态：缩进 4 空格、每个键独占一行
        expect(editorBox.mock.calls[0][0].content).toContain('\n    "a"');
        expect(editorBox.mock.calls[0][0].content).not.toBe(long);
    });

    it('非 JSON 的长文本原样交给编辑器，美化失败不改坏内容', async () => {
        const broken = `{"a":1,,,${'y'.repeat(60)}}`;
        const wrapper = mountInput({ modelValue: broken, columnName: 'ext' });
        const icon = iconOf(wrapper, 'FullScreen');
        expect(icon).toBeTruthy();
        await icon!.trigger('mousedown');
        expect(editorBox).toHaveBeenCalledTimes(1);
        expect(editorBox.mock.calls[0][0].content).toBe(broken);
    });
});
