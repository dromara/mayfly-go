import { describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import type { VueWrapper } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';

import type { TableColumnDef } from '../../types';
import BinaryValuePreview from '../../widgets/BinaryValuePreview.vue';
import DbTableRowDetail from '../DbTableRowDetail.vue';

/**
 * 行详情面板的呈现守卫
 *
 * 钉的是四种「在横向网格里会被看成别的东西」的情况：NULL、脱敏掩码、二进制 hex、长文本。
 * 面板只读且不做任何值加工，所以判据都在「标注是否到位、复制入口是否会误导」。
 */

const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });

const column = (overrides: Partial<TableColumnDef>): TableColumnDef => ({ columnName: 'c', ...overrides });

function mountDetail(columns: TableColumnDef[], row: Record<string, unknown>, dbType = 'mysql') {
    return mount(DbTableRowDetail, {
        props: { columns, row, dbType, visible: true, title: 'detail' },
        global: {
            plugins: [i18n, ElementPlus],
            // el-dialog 把内容 teleport 到 body，不抹平就查不到面板里的任何节点
            stubs: { ElDialog: { template: '<div><slot /></div>' } },
        },
    });
}

function rowBlocks(wrapper: ReturnType<typeof mountDetail>) {
    return wrapper.findAll('.row-detail__row');
}

describe('DbTableRowDetail 行详情呈现', () => {
    it('列类型角标由 columnType 经方言归一现算（12/ab），完整类型改由悬浮 tooltip 提供', () => {
        const wrapper = mountDetail([column({ columnName: 'id', columnType: 'int8' }), column({ columnName: 'name', columnType: 'varchar(20)' })], { id: 7, name: 'a' });
        const blocks = rowBlocks(wrapper);
        expect(blocks).toHaveLength(2);
        // 角标只显简称，完整列类型不再常驻文本
        expect(blocks[0].find('.row-detail__subscript').text()).toBe('12');
        expect(blocks[0].text()).not.toContain('int8');
        // 完整类型由悬浮 tooltip 给出
        expect(blocks[0].findComponent({ name: 'ElTooltip' }).props('content')).toBe('int8');
        expect(blocks[1].find('.row-detail__subscript').text()).toBe('ab');
    });

    it('日期时间列角标走时钟图标分支，不把内部键 icon-clock 当文字漏到界面', () => {
        const wrapper = mountDetail([column({ columnName: 'create_time', columnType: 'datetime' })], { create_time: '2026-09-17 19:56:22' });
        const block = rowBlocks(wrapper)[0];
        expect(block.find('.row-detail__subscript').exists()).toBe(true);
        expect(block.text()).not.toContain('icon-clock');
    });

    it('无方言（拿不到归一）时回退完整类型标签，不致类型信息丢失', () => {
        const wrapper = mountDetail([column({ columnName: 'id', columnType: 'int8' })], { id: 7 }, '');
        expect(rowBlocks(wrapper)[0].text()).toContain('int8');
    });

    it('NULL 显式标注且禁用复制（复制一个空值出去会被当成空串）', () => {
        const wrapper = mountDetail([column({ columnName: 'memo' })], { memo: null });
        const block = rowBlocks(wrapper)[0];
        expect(block.text()).toContain('NULL');
        const copyButton = block.findAllComponents({ name: 'ElButton' }).at(-1);
        expect(copyButton?.props('disabled')).toBe(true);
    });

    it('脱敏列带标记：掩码文本不标出来就会被当真值复制走', () => {
        const wrapper = mountDetail([column({ columnName: 'phone', masked: true })], { phone: '138****0000' });
        expect(rowBlocks(wrapper)[0].text()).toContain('db.maskedTag');
    });

    it('主键列带标记', () => {
        const wrapper = mountDetail([column({ columnName: 'id', isPrimaryKey: true })], { id: 1 });
        expect(rowBlocks(wrapper)[0].text()).toContain('db.primaryKey');
    });

    it('二进制列给预览入口并带上该列的 hex 值', async () => {
        const wrapper = mountDetail([column({ columnName: 'avatar', columnType: 'bytea' })], { avatar: '616263' });
        const block = rowBlocks(wrapper)[0];
        const previewButton = block.findAllComponents({ name: 'ElButton' }).find((btn: VueWrapper) => btn.text().includes('db.binaryPreviewEntry'));
        expect(previewButton, '二进制列必须有预览入口').toBeTruthy();

        await previewButton!.trigger('click');
        const dialog = wrapper.findComponent(BinaryValuePreview);
        expect(dialog.props('visible')).toBe(true);
        expect(dialog.props('hex')).toBe('616263');
    });

    it('非二进制列不给预览入口', () => {
        const wrapper = mountDetail([column({ columnName: 'name', columnType: 'varchar' })], { name: 'abc' });
        expect(rowBlocks(wrapper)[0].text()).not.toContain('db.binaryPreviewEntry');
    });

    it('长值与含换行的值改用块级等宽展示，不挤成一行', () => {
        const wrapper = mountDetail([column({ columnName: 'a' }), column({ columnName: 'b' })], {
            a: 'x'.repeat(120),
            b: 'line1\nline2',
        });
        const blocks = rowBlocks(wrapper);
        expect(blocks[0].find('.row-detail__pre').exists()).toBe(true);
        expect(blocks[1].find('.row-detail__pre').exists()).toBe(true);
    });
});
