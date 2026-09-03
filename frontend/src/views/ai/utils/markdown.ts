/**
 * Markdown → HTML 转换（marked 解析 + DOMPurify 消毒）
 */
import DOMPurify from 'dompurify';
import { marked } from 'marked';

marked.setOptions({
    gfm: true, // 表格、删除线、任务列表等 GFM 扩展
    breaks: true, // 单个换行渲染为 <br>
});

export function renderMarkdown(text: string): string {
    if (!text) return '';
    const html = marked.parse(text, { async: false }) as string;
    // 消毒防 XSS：仅保留安全的 HTML 标签与属性，允许 a/target 以支持外链新窗口打开
    return DOMPurify.sanitize(html, { ADD_ATTR: ['target'] });
}
