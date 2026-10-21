/**
 * Markdown → HTML 转换（marked 解析 + DOMPurify 消毒）
 *
 * 为什么放在 components/markdown 而不是某个业务视图下：聊天消息与运维运行/提案的结论展示共用
 * 同一套解析与消毒规则——结论是模型产出的 Markdown，两处必须渲染成同一个样子，也必须走同一道
 * XSS 过滤（消毒规则若分叉，最可能的后果是某一面漏过滤，而不是某一面不好看）
 */
import DOMPurify from 'dompurify';
import { marked } from 'marked';

marked.setOptions({
    gfm: true, // 表格、删除线、任务列表等 GFM 扩展
    breaks: true, // 单个换行渲染为 <br>
});

export function renderMarkdown(text: string): string {
    if (!text) return '';
    const html = marked.parse(text, { async: false });
    // 消毒防 XSS：仅保留安全的 HTML 标签与属性，允许 a/target 以支持外链新窗口打开
    return DOMPurify.sanitize(html, { ADD_ATTR: ['target'] });
}
