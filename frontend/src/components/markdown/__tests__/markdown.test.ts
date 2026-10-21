/**
 * Markdown 渲染契约测试
 *
 * 断言只落在「本仓自己的决定」上，不依赖 happy-dom 里 DOMPurify 的白名单
 * （实测该环境下 DOMPurify 会连 <h2>/<thead> 一起丢，按最终 HTML 断言等于测环境）：
 *   1. 解析配置：GFM 表格 + breaks（模型按行输出结论，单换行必须断行）；
 *   2. 安全边界：marked 的产物必须经 DOMPurify 消毒后才返回（去掉消毒即测红）；
 *   3. 组件契约：MarkdownView 走同一套解析+消毒，不自己再解析一遍。
 */
import DOMPurify from 'dompurify';
import { mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import MarkdownView from '../MarkdownView.vue';
import { renderMarkdown } from '../markdown';

afterEach(() => {
    vi.restoreAllMocks();
});

describe('renderMarkdown', () => {
    it('按 GFM + breaks 解析（标题、表格、列表、单换行断行）', () => {
        const sanitize = vi.spyOn(DOMPurify, 'sanitize');
        renderMarkdown('## 结论\n\n| 维度 | 判定 |\n|---|---|\n| 告警数 | 一致 |\n\n- 项目一\n\n第一行\n第二行');

        // 断言「消毒前」的解析产物：证明 gfm/breaks 配置生效，与环境的标签白名单无关
        const parsed = String(sanitize.mock.calls[0]?.[0] ?? '');
        expect(parsed).toContain('<h2');
        expect(parsed).toContain('<table');
        expect(parsed).toContain('<li>');
        expect(parsed).toContain('<br>');
    });

    it('解析结果必须经 DOMPurify 消毒后才返回', () => {
        const sanitize = vi.spyOn(DOMPurify, 'sanitize').mockReturnValue('<p>sanitized</p>');
        expect(renderMarkdown('原文')).toBe('<p>sanitized</p>');
        expect(sanitize).toHaveBeenCalledTimes(1);
    });

    it('空内容不产出 HTML，也不走解析', () => {
        const sanitize = vi.spyOn(DOMPurify, 'sanitize');
        expect(renderMarkdown('')).toBe('');
        expect(sanitize).not.toHaveBeenCalled();
    });
});

describe('MarkdownView', () => {
    it('内容走共享的「解析 + 消毒」链路，产物落进 .markdown-body', () => {
        const sanitize = vi.spyOn(DOMPurify, 'sanitize');
        const wrapper = mount(MarkdownView, { props: { content: '## 标题' } });

        expect(sanitize).toHaveBeenCalledTimes(1);
        expect(wrapper.find('.markdown-body').exists()).toBe(true);
    });

    it('content 缺省时不解析也不产出内容', () => {
        const sanitize = vi.spyOn(DOMPurify, 'sanitize');
        const wrapper = mount(MarkdownView);

        expect(sanitize).not.toHaveBeenCalled();
        expect(wrapper.find('.markdown-body').text()).toBe('');
    });
});
